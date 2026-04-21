package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	ccdao "dodevops-api/api/configcenter/dao"
	"dodevops-api/api/deploy/dao"
	"dodevops-api/api/deploy/model"

	"gorm.io/gorm"
)

const (
	defaultPipelineDeployTimeout = 30 * time.Minute
)

type IPipelineService interface {
	// CreatePipelineRun creates a new pipeline run for a deploy request
	CreatePipelineRun(req *model.CreatePipelineRunRequest) (*model.PipelineRun, error)

	// StartPipelineRun starts executing a pipeline run (build → scan → deploy → notify)
	StartPipelineRun(pipelineRunID uint) error

	// ProcessPendingPipelineRuns finds and processes all pending pipeline runs
	ProcessPendingPipelineRuns(limit int) error

	// RecoverStalePipelineRuns finds and fails pipeline runs stuck for longer than timeout
	RecoverStalePipelineRuns(timeout time.Duration, limit int) error

	// GetPipelineRunWithStages returns a pipeline run with its stage records
	GetPipelineRunWithStages(pipelineRunID uint) (*model.PipelineRunResponse, error)

	// GetPipelineRunByRequestID returns the pipeline run for a deploy request
	GetPipelineRunByRequestID(requestID uint) (*model.PipelineRunResponse, error)
}

type PipelineService struct {
	db             *gorm.DB
	pipelineDao    dao.IPipelineDao
	deployDao      dao.IDeployDao
	jenkinsAdapter IJenkinsPipelineAdapter
	harborAdapter  IHarborAdapter
	notifier       IDeployNotifier
}

func newPipelineService(db *gorm.DB) *PipelineService {
	return &PipelineService{
		db:             db,
		pipelineDao:    dao.NewPipelineDao(db),
		deployDao:      dao.NewDeployDao(db),
		jenkinsAdapter: NewJenkinsPipelineAdapter(),
		harborAdapter:  NewHarborAdapter(HarborAdapterOptions{AllowInsecureHTTP: true}),
		notifier:       NewDeployNotifier(db),
	}
}

func NewPipelineService(db *gorm.DB) IPipelineService {
	return newPipelineService(db)
}

func (s *PipelineService) CreatePipelineRun(req *model.CreatePipelineRunRequest) (*model.PipelineRun, error) {
	if req == nil {
		return nil, fmt.Errorf("创建流水线运行请求不能为空")
	}
	if _, err := s.deployDao.GetDeployRequestByID(req.RequestID); err != nil {
		return nil, fmt.Errorf("读取部署申请失败: %v", err)
	}
	if existing, err := s.pipelineDao.GetPipelineRunByRequestID(req.RequestID); err == nil && existing != nil {
		return nil, fmt.Errorf("部署申请已存在流水线运行: %d", existing.ID)
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("查询部署申请关联流水线失败: %v", err)
	}

	buildParamsJSON, err := marshalPipelineJSON(req.BuildParams)
	if err != nil {
		return nil, fmt.Errorf("序列化构建参数失败: %v", err)
	}
	scanPolicyJSON, err := marshalPipelineJSON(req.ScanPolicy)
	if err != nil {
		return nil, fmt.Errorf("序列化扫描策略失败: %v", err)
	}

	run := &model.PipelineRun{
		RequestID:              req.RequestID,
		ApplicationID:          req.ApplicationID,
		Status:                 model.PipelineStatusPending,
		CurrentStage:           "",
		JenkinsServerID:        req.JenkinsServerID,
		JenkinsJobNameSnapshot: strings.TrimSpace(req.JenkinsJobNameSnapshot),
		GitRef:                 strings.TrimSpace(req.GitRef),
		BuildParamsJSON:        buildParamsJSON,
		HarborServerID:         req.HarborServerID,
		HarborProject:          strings.TrimSpace(req.HarborProject),
		HarborRepository:       strings.TrimSpace(req.HarborRepository),
		ArtifactTag:            strings.TrimSpace(req.ArtifactTag),
		PlannedImageRef:        strings.TrimSpace(req.PlannedImageRef),
		ScanPolicyJSON:         scanPolicyJSON,
	}
	if err := s.pipelineDao.CreatePipelineRun(run); err != nil {
		return nil, fmt.Errorf("创建流水线运行失败: %v", err)
	}
	if err := s.deployDao.UpdateDeployRequest(req.RequestID, map[string]interface{}{
		"pipeline_status":        model.PipelineStatusPending,
		"current_pipeline_stage": "",
		"pipeline_error_message": "",
		"updated_at":             time.Now(),
	}); err != nil {
		return nil, fmt.Errorf("更新部署申请流水线状态失败: %v", err)
	}

	return run, nil
}

func (s *PipelineService) StartPipelineRun(pipelineRunID uint) error {
	run, err := s.pipelineDao.GetPipelineRunByID(pipelineRunID)
	if err != nil {
		return fmt.Errorf("读取流水线运行失败: %v", err)
	}
	if run.Status != model.PipelineStatusPending && run.Status != model.PipelineStatusBuilding {
		return fmt.Errorf("流水线运行状态非法，当前状态=%s", run.Status)
	}

	log.Printf("pipeline run started: pipelineRunID=%d requestID=%d", run.ID, run.RequestID)
	if err := s.executeBuildStage(run); err != nil {
		s.finalizePipelineFailure(run, err)
		return err
	}
	if err := s.executeScanStage(run); err != nil {
		s.finalizePipelineFailure(run, err)
		return err
	}
	req, record, err := s.executeDeployStage(run)
	if err != nil {
		s.finalizePipelineFailure(run, err)
		return err
	}
	s.executeNotifyStage(run, req, record)
	log.Printf("pipeline run finished: pipelineRunID=%d requestID=%d status=%s", run.ID, run.RequestID, run.Status)
	return nil
}

func (s *PipelineService) ProcessPendingPipelineRuns(limit int) error {
	runs, err := s.pipelineDao.ListPendingApprovedPipelineRuns(limit)
	if err != nil {
		return fmt.Errorf("查询待处理流水线运行失败: %v", err)
	}

	var firstErr error
	for _, run := range runs {
		claimed, err := s.pipelineDao.ClaimPipelineRun(run.ID, "")
		if err != nil {
			log.Printf("pipeline run claim failed: pipelineRunID=%d requestID=%d err=%v", run.ID, run.RequestID, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if !claimed {
			log.Printf("pipeline run already claimed by another instance: pipelineRunID=%d requestID=%d", run.ID, run.RequestID)
			continue
		}
		if err := s.StartPipelineRun(run.ID); err != nil {
			log.Printf("pipeline run processing failed: pipelineRunID=%d requestID=%d err=%v", run.ID, run.RequestID, err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (s *PipelineService) RecoverStalePipelineRuns(timeout time.Duration, limit int) error {
	runs, err := s.pipelineDao.ListStalePipelineRuns(timeout, limit)
	if err != nil {
		return fmt.Errorf("查询僵死流水线运行失败: %v", err)
	}
	for _, run := range runs {
		log.Printf("pipeline run stale recovery: pipelineRunID=%d requestID=%d status=%s updated_at=%s", run.ID, run.RequestID, run.Status, run.UpdatedAt)
		s.finalizePipelineFailure(&run, fmt.Errorf("流水线阶段超时（超过 %s），自动标记为失败", timeout))
	}
	return nil
}

func (s *PipelineService) GetPipelineRunWithStages(pipelineRunID uint) (*model.PipelineRunResponse, error) {
	run, err := s.pipelineDao.GetPipelineRunByID(pipelineRunID)
	if err != nil {
		return nil, fmt.Errorf("读取流水线运行失败: %v", err)
	}
	records, err := s.pipelineDao.GetPipelineStageRecordsByPipelineRunID(pipelineRunID)
	if err != nil {
		return nil, fmt.Errorf("读取流水线阶段记录失败: %v", err)
	}
	return buildPipelineRunResponse(run, records), nil
}

func (s *PipelineService) GetPipelineRunByRequestID(requestID uint) (*model.PipelineRunResponse, error) {
	run, err := s.pipelineDao.GetPipelineRunByRequestID(requestID)
	if err != nil {
		return nil, fmt.Errorf("读取流水线运行失败: %v", err)
	}
	records, err := s.pipelineDao.GetPipelineStageRecordsByPipelineRunID(run.ID)
	if err != nil {
		return nil, fmt.Errorf("读取流水线阶段记录失败: %v", err)
	}
	return buildPipelineRunResponse(run, records), nil
}

func (s *PipelineService) executeBuildStage(run *model.PipelineRun) error {
	if s.jenkinsAdapter == nil {
		return s.failPipelineStage(run, nil, model.PipelineStageBuild, "Jenkins Pipeline 适配器未初始化", "")
	}
	if run.JenkinsServerID == 0 {
		return s.failPipelineStage(run, nil, model.PipelineStageBuild, "Jenkins 服务器未配置", "")
	}
	if strings.TrimSpace(run.JenkinsJobNameSnapshot) == "" {
		return s.failPipelineStage(run, nil, model.PipelineStageBuild, "Jenkins 任务名不能为空", "")
	}
	if err := s.transitionStage(run, model.PipelineStageBuild, model.PipelineStatusBuilding, ""); err != nil {
		return err
	}
	record, err := s.createStageRecord(run.ID, model.PipelineStageBuild)
	if err != nil {
		return fmt.Errorf("创建构建阶段记录失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultJenkinsPipelineTimeout)
	defer cancel()

	params, err := parseBuildParams(run.BuildParamsJSON)
	if err != nil {
		return s.failPipelineStage(run, record, model.PipelineStageBuild, fmt.Sprintf("解析构建参数失败: %v", err), "")
	}
	if params == nil {
		params = make(map[string]string)
	}
	if strings.TrimSpace(run.GitRef) != "" {
		params["GIT_REF"] = strings.TrimSpace(run.GitRef)
	}
	log.Printf("pipeline build stage started: pipelineRunID=%d requestID=%d job=%s gitRef=%s", run.ID, run.RequestID, run.JenkinsJobNameSnapshot, run.GitRef)

	queueID, err := s.jenkinsAdapter.TriggerBuild(ctx, run.JenkinsServerID, run.JenkinsJobNameSnapshot, params)
	if err != nil {
		return s.failPipelineStage(run, record, model.PipelineStageBuild, fmt.Sprintf("触发 Jenkins 构建失败: %v", err), "")
	}
	if err := s.applyRunUpdates(run, map[string]interface{}{
		"jenkins_queue_id": queueID,
	}, nil); err != nil {
		return s.failPipelineStage(run, record, model.PipelineStageBuild, fmt.Sprintf("更新 Jenkins 队列信息失败: %v", err), "")
	}

	buildNumber, err := s.jenkinsAdapter.GetBuildNumberFromQueue(ctx, run.JenkinsServerID, queueID)
	if err != nil {
		return s.failPipelineStage(run, record, model.PipelineStageBuild, fmt.Sprintf("获取 Jenkins 构建编号失败: %v", err), marshalPipelineJSONString(map[string]interface{}{"queueID": queueID}))
	}
	buildDetail, err := s.jenkinsAdapter.PollBuildUntilComplete(ctx, run.JenkinsServerID, run.JenkinsJobNameSnapshot, buildNumber, defaultJenkinsPipelineTimeout)
	if err != nil {
		return s.failPipelineStage(run, record, model.PipelineStageBuild, fmt.Sprintf("轮询 Jenkins 构建结果失败: %v", err), marshalPipelineJSONString(map[string]interface{}{"queueID": queueID, "buildNumber": buildNumber}))
	}
	if buildDetail == nil {
		return s.failPipelineStage(run, record, model.PipelineStageBuild, "Jenkins 构建详情为空", marshalPipelineJSONString(map[string]interface{}{"queueID": queueID, "buildNumber": buildNumber}))
	}
	if !strings.EqualFold(strings.TrimSpace(buildDetail.Result), "SUCCESS") {
		return s.failPipelineStage(run, record, model.PipelineStageBuild, fmt.Sprintf("Jenkins 构建失败，结果=%s", buildDetail.Result), marshalPipelineJSONString(buildDetail))
	}
	artifactTag, err := s.jenkinsAdapter.ExtractImageTagFromBuildLog(ctx, run.JenkinsServerID, run.JenkinsJobNameSnapshot, buildNumber)
	if err != nil {
		return s.failPipelineStage(run, record, model.PipelineStageBuild, fmt.Sprintf("提取镜像标签失败: %v", err), marshalPipelineJSONString(buildDetail))
	}

	buildStageDetail := marshalPipelineJSONString(map[string]interface{}{
		"queueID":       queueID,
		"buildNumber":   buildNumber,
		"artifactTag":   artifactTag,
		"buildDetail":   buildDetail,
		"jenkinsJob":    run.JenkinsJobNameSnapshot,
		"jenkinsServer": run.JenkinsServerID,
	})
	if err := s.pipelineDao.UpdatePipelineStageRecord(record.ID, map[string]interface{}{
		"external_id":  strconv.Itoa(buildNumber),
		"external_url": buildDetail.URL,
		"updated_at":   time.Now(),
	}); err != nil {
		return s.failPipelineStage(run, record, model.PipelineStageBuild, fmt.Sprintf("更新构建阶段外部信息失败: %v", err), buildStageDetail)
	}
	record.ExternalID = strconv.Itoa(buildNumber)
	record.ExternalURL = buildDetail.URL
	if err := s.completeStageRecord(record, model.PipelineStageStatusSucceeded, buildStageDetail, ""); err != nil {
		return s.failPipelineStage(run, record, model.PipelineStageBuild, fmt.Sprintf("完成构建阶段记录失败: %v", err), buildStageDetail)
	}
	if err := s.applyRunUpdates(run, map[string]interface{}{
		"jenkins_queue_id":     queueID,
		"jenkins_build_number": buildNumber,
		"jenkins_build_url":    buildDetail.URL,
		"artifact_tag":         artifactTag,
	}, nil); err != nil {
		return s.failPipelineStage(run, record, model.PipelineStageBuild, fmt.Sprintf("更新构建结果失败: %v", err), buildStageDetail)
	}
	if err := s.transitionStage(run, model.PipelineStageScan, model.PipelineStatusScanning, ""); err != nil {
		return err
	}
	log.Printf("pipeline build stage succeeded: pipelineRunID=%d requestID=%d buildNumber=%d artifactTag=%s", run.ID, run.RequestID, buildNumber, artifactTag)
	return nil
}

func (s *PipelineService) executeScanStage(run *model.PipelineRun) error {
	if s.harborAdapter == nil {
		return s.failPipelineStage(run, nil, model.PipelineStageScan, "Harbor 适配器未初始化", "")
	}
	if run.HarborServerID == 0 {
		return s.failPipelineStage(run, nil, model.PipelineStageScan, "Harbor 服务器未配置", "")
	}
	if strings.TrimSpace(run.HarborProject) == "" || strings.TrimSpace(run.HarborRepository) == "" {
		return s.failPipelineStage(run, nil, model.PipelineStageScan, "Harbor 项目或仓库配置缺失", "")
	}
	if strings.TrimSpace(run.ArtifactTag) == "" {
		return s.failPipelineStage(run, nil, model.PipelineStageScan, "镜像标签为空，无法执行扫描", "")
	}
	record, err := s.createStageRecord(run.ID, model.PipelineStageScan)
	if err != nil {
		return fmt.Errorf("创建扫描阶段记录失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultHarborScanTimeout)
	defer cancel()

	log.Printf("pipeline scan stage started: pipelineRunID=%d requestID=%d project=%s repository=%s tag=%s", run.ID, run.RequestID, run.HarborProject, run.HarborRepository, run.ArtifactTag)
	artifact, err := s.harborAdapter.GetArtifact(ctx, run.HarborServerID, run.HarborProject, run.HarborRepository, run.ArtifactTag)
	if err != nil {
		return s.failPipelineStage(run, record, model.PipelineStageScan, fmt.Sprintf("读取 Harbor 制品失败: %v", err), "")
	}
	if artifact == nil || strings.TrimSpace(artifact.Digest) == "" {
		return s.failPipelineStage(run, record, model.PipelineStageScan, "Harbor 制品摘要为空", marshalPipelineJSONString(artifact))
	}
	overview, err := s.harborAdapter.PollScanUntilComplete(ctx, run.HarborServerID, run.HarborProject, run.HarborRepository, artifact.Digest, defaultHarborScanTimeout)
	if err != nil {
		reportJSON := marshalPipelineJSONString(map[string]interface{}{"artifact": artifact, "overview": overview})
		if applyErr := s.applyRunUpdates(run, map[string]interface{}{
			"artifact_digest":  artifact.Digest,
			"scan_report_json": reportJSON,
		}, nil); applyErr != nil {
			log.Printf("pipeline scan stage partial update failed: pipelineRunID=%d err=%v", run.ID, applyErr)
		}
		return s.failPipelineStage(run, record, model.PipelineStageScan, fmt.Sprintf("等待 Harbor 扫描完成失败: %v", err), reportJSON)
	}
	policy, err := parseScanPolicy(run.ScanPolicyJSON)
	if err != nil {
		return s.failPipelineStage(run, record, model.PipelineStageScan, fmt.Sprintf("解析扫描策略失败: %v", err), marshalPipelineJSONString(map[string]interface{}{"artifact": artifact, "overview": overview}))
	}
	evaluation, err := s.harborAdapter.EvaluateScanPolicy(overview, policy)
	if err != nil {
		return s.failPipelineStage(run, record, model.PipelineStageScan, fmt.Sprintf("执行扫描策略评估失败: %v", err), marshalPipelineJSONString(map[string]interface{}{"artifact": artifact, "overview": overview, "policy": policy}))
	}
	reportJSON := marshalPipelineJSONString(map[string]interface{}{
		"artifact":   artifact,
		"overview":   overview,
		"policy":     policy,
		"evaluation": evaluation,
	})
	if err := s.applyRunUpdates(run, map[string]interface{}{
		"artifact_digest":  artifact.Digest,
		"scan_report_json": reportJSON,
	}, nil); err != nil {
		return s.failPipelineStage(run, record, model.PipelineStageScan, fmt.Sprintf("更新扫描结果失败: %v", err), reportJSON)
	}
	if evaluation == nil || !evaluation.Passed {
		errMsg := "镜像扫描策略未通过"
		if evaluation != nil && strings.TrimSpace(evaluation.Message) != "" {
			errMsg = evaluation.Message
		}
		return s.failPipelineStage(run, record, model.PipelineStageScan, errMsg, reportJSON)
	}
	if err := s.completeStageRecord(record, model.PipelineStageStatusSucceeded, reportJSON, ""); err != nil {
		return s.failPipelineStage(run, record, model.PipelineStageScan, fmt.Sprintf("完成扫描阶段记录失败: %v", err), reportJSON)
	}
	if err := s.transitionStage(run, model.PipelineStageDeploy, model.PipelineStatusDeploying, ""); err != nil {
		return err
	}
	log.Printf("pipeline scan stage succeeded: pipelineRunID=%d requestID=%d digest=%s", run.ID, run.RequestID, artifact.Digest)
	return nil
}

func (s *PipelineService) executeDeployStage(run *model.PipelineRun) (*model.DeployRequest, *model.ExecutionRecord, error) {
	record, err := s.createStageRecord(run.ID, model.PipelineStageDeploy)
	if err != nil {
		return nil, nil, fmt.Errorf("创建部署阶段记录失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultPipelineDeployTimeout)
	defer cancel()
	select {
	case <-ctx.Done():
		return nil, nil, s.failPipelineStage(run, record, model.PipelineStageDeploy, fmt.Sprintf("部署阶段上下文初始化失败: %v", ctx.Err()), "")
	default:
	}

	finalImageRef := strings.TrimSpace(s.buildImageRef(run))
	if finalImageRef == "" {
		finalImageRef = strings.TrimSpace(run.PlannedImageRef)
	}
	if finalImageRef == "" {
		return nil, nil, s.failPipelineStage(run, record, model.PipelineStageDeploy, "无法构建最终镜像地址", "")
	}
	if err := s.applyRunUpdates(run, map[string]interface{}{
		"final_image_ref": finalImageRef,
	}, map[string]interface{}{
		"image":      finalImageRef,
		"updated_at": time.Now(),
	}); err != nil {
		return nil, nil, s.failPipelineStage(run, record, model.PipelineStageDeploy, fmt.Sprintf("更新最终镜像地址失败: %v", err), "")
	}

	log.Printf("pipeline deploy stage started: pipelineRunID=%d requestID=%d image=%s", run.ID, run.RequestID, finalImageRef)
	updatedReq, execRecord, execErr := s.executeDeployWithSkipNotify(run.RequestID, "pipeline deploy stage")
	if ctx.Err() != nil && execErr == nil {
		execErr = fmt.Errorf("部署阶段超时: %w", ctx.Err())
	}

	stageDetail := marshalPipelineJSONString(map[string]interface{}{
		"deployRequest":   updatedReq,
		"executionRecord": execRecord,
		"finalImageRef":   finalImageRef,
	})
	if execErr != nil {
		errMsg := execErr.Error()
		if msg := execErrorMessage(execRecord); strings.TrimSpace(msg) != "" {
			errMsg = msg
		}
		return updatedReq, execRecord, s.failPipelineStage(run, record, model.PipelineStageDeploy, errMsg, stageDetail)
	}
	if execRecord == nil {
		return updatedReq, nil, s.failPipelineStage(run, record, model.PipelineStageDeploy, "部署执行记录为空", stageDetail)
	}

	if strings.TrimSpace(execRecord.Status) != model.ExecutionStatusSucceeded {
		errMsg := firstNonEmptyPipelineString(execErrorMessage(execRecord), fmt.Sprintf("部署执行未成功，状态=%s", execRecord.Status))
		return updatedReq, execRecord, s.failPipelineStage(run, record, model.PipelineStageDeploy, errMsg, stageDetail)
	}
	if err := s.completeStageRecord(record, model.PipelineStageStatusSucceeded, stageDetail, ""); err != nil {
		return updatedReq, execRecord, s.failPipelineStage(run, record, model.PipelineStageDeploy, fmt.Sprintf("完成部署阶段记录失败: %v", err), stageDetail)
	}
	if err := s.transitionStage(run, model.PipelineStageDeploy, model.PipelineStatusSucceeded, ""); err != nil {
		return updatedReq, execRecord, err
	}
	log.Printf("pipeline deploy stage succeeded: pipelineRunID=%d requestID=%d executionRecordID=%d", run.ID, run.RequestID, execRecord.ID)
	return updatedReq, execRecord, nil
}

func (s *PipelineService) executeNotifyStage(run *model.PipelineRun, req *model.DeployRequest, execRecord *model.ExecutionRecord) {
	if s.notifier == nil || req == nil || execRecord == nil {
		return
	}
	record, err := s.createStageRecord(run.ID, model.PipelineStageNotify)
	if err != nil {
		log.Printf("pipeline notify stage create record failed: pipelineRunID=%d requestID=%d err=%v", run.ID, run.RequestID, err)
		return
	}
	log.Printf("pipeline notify stage started: pipelineRunID=%d requestID=%d", run.ID, run.RequestID)
	err = s.notifier.NotifyExecutionResult(req, execRecord)
	detailJSON := marshalPipelineJSONString(map[string]interface{}{
		"requestNo":         req.RequestNo,
		"executionStatus":   execRecord.Status,
		"executionRecordID": execRecord.ID,
	})
	if err != nil {
		if completeErr := s.completeStageRecord(record, model.PipelineStageStatusFailed, detailJSON, err.Error()); completeErr != nil {
			log.Printf("pipeline notify stage completion failed: pipelineRunID=%d requestID=%d err=%v", run.ID, run.RequestID, completeErr)
		}
		log.Printf("pipeline notify stage failed: pipelineRunID=%d requestID=%d err=%v", run.ID, run.RequestID, err)
		return
	}
	if err := s.completeStageRecord(record, model.PipelineStageStatusSucceeded, detailJSON, ""); err != nil {
		log.Printf("pipeline notify stage completion failed: pipelineRunID=%d requestID=%d err=%v", run.ID, run.RequestID, err)
		return
	}
	log.Printf("pipeline notify stage succeeded: pipelineRunID=%d requestID=%d", run.ID, run.RequestID)
}

func (s *PipelineService) finalizePipelineFailure(run *model.PipelineRun, err error) {
	if run == nil || err == nil {
		return
	}
	errMsg := err.Error()
	now := time.Now()
	runUpdates := map[string]interface{}{
		"status":      model.PipelineStatusFailed,
		"last_error":  errMsg,
		"updated_at":  now,
		"finished_at": now,
	}
	requestUpdates := map[string]interface{}{
		"pipeline_status":        model.PipelineStatusFailed,
		"pipeline_error_message": errMsg,
		"request_status":         model.DeployRequestStatusFailed,
		"execution_status":       model.ExecutionStatusFailed,
		"updated_at":             now,
		"finished_at":            now,
	}
	if applyErr := s.applyRunUpdates(run, runUpdates, requestUpdates); applyErr != nil {
		log.Printf("pipeline finalize failure update failed: pipelineRunID=%d err=%v", run.ID, applyErr)
	}
	_ = s.deployDao.DeactivateResourceOwnersByRequestID(run.RequestID)
	stageRecords, _ := s.pipelineDao.GetPipelineStageRecordsByPipelineRunID(run.ID)
	for _, sr := range stageRecords {
		if sr.Status == model.PipelineStageStatusRunning {
			_ = s.completeStageRecord(&sr, model.PipelineStageStatusFailed, "", errMsg)
		}
	}
	req, _ := s.deployDao.GetDeployRequestByID(run.RequestID)
	record := &model.ExecutionRecord{
		RequestID: run.RequestID,
		Status:    model.ExecutionStatusFailed,
		DetailJSON: marshalPipelineJSONString(map[string]string{
			"pipelineRunID": fmt.Sprintf("%d", run.ID),
			"error":         errMsg,
		}),
		StartedAt: &now,
		EndedAt:   &now,
	}
	_ = s.deployDao.CreateExecutionRecord(record)
	s.executeNotifyStage(run, req, record)
}

func (s *PipelineService) transitionStage(run *model.PipelineRun, stage, status, errMsg string) error {
	if run == nil {
		return fmt.Errorf("流水线运行不能为空")
	}
	now := time.Now()
	runUpdates := map[string]interface{}{
		"status":        status,
		"current_stage": stage,
		"last_error":    strings.TrimSpace(errMsg),
		"updated_at":    now,
	}
	requestUpdates := map[string]interface{}{
		"pipeline_status":        status,
		"current_pipeline_stage": stage,
		"pipeline_error_message": strings.TrimSpace(errMsg),
		"updated_at":             now,
	}
	if run.StartedAt == nil && status != model.PipelineStatusPending {
		runUpdates["started_at"] = now
	}
	if status == model.PipelineStatusSucceeded || status == model.PipelineStatusFailed {
		runUpdates["finished_at"] = now
	}
	if err := s.applyRunUpdates(run, runUpdates, requestUpdates); err != nil {
		return fmt.Errorf("更新流水线阶段失败: %v", err)
	}
	log.Printf("pipeline stage transition: pipelineRunID=%d requestID=%d stage=%s status=%s err=%s", run.ID, run.RequestID, stage, status, strings.TrimSpace(errMsg))
	return nil
}

func (s *PipelineService) createStageRecord(runID uint, stage string) (*model.PipelineStageRecord, error) {
	now := time.Now()
	record := &model.PipelineStageRecord{
		PipelineRunID: runID,
		Stage:         stage,
		Status:        model.PipelineStageStatusRunning,
		StartedAt:     &now,
	}
	if err := s.pipelineDao.CreatePipelineStageRecord(record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *PipelineService) completeStageRecord(record *model.PipelineStageRecord, status, detailJSON, errMsg string) error {
	if record == nil {
		return fmt.Errorf("流水线阶段记录不能为空")
	}
	now := time.Now()
	updates := map[string]interface{}{
		"status":        status,
		"detail_json":   detailJSON,
		"error_message": errMsg,
		"finished_at":   now,
		"updated_at":    now,
	}
	if err := s.pipelineDao.UpdatePipelineStageRecord(record.ID, updates); err != nil {
		return err
	}
	record.Status = status
	record.DetailJSON = detailJSON
	record.ErrorMessage = errMsg
	record.FinishedAt = &now
	return nil
}

func (s *PipelineService) buildImageRef(run *model.PipelineRun) string {
	if run == nil {
		return ""
	}
	tag := strings.TrimSpace(run.ArtifactTag)
	project := strings.Trim(strings.TrimSpace(run.HarborProject), "/")
	repository := strings.Trim(strings.TrimSpace(run.HarborRepository), "/")
	if tag == "" || project == "" || repository == "" {
		return strings.TrimSpace(run.PlannedImageRef)
	}

	account, err := ccdao.NewAccountAuthDao().GetByID(run.HarborServerID)
	if err != nil || account == nil {
		return firstNonEmptyPipelineString(strings.TrimSpace(run.PlannedImageRef), fmt.Sprintf("%s/%s:%s", project, repository, tag))
	}

	host := strings.TrimSpace(account.Host)
	if host == "" {
		return firstNonEmptyPipelineString(strings.TrimSpace(run.PlannedImageRef), fmt.Sprintf("%s/%s:%s", project, repository, tag))
	}
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}
	parsed, err := url.Parse(host)
	if err != nil || parsed.Host == "" {
		registryHost := strings.TrimSpace(account.Host)
		if account.Port > 0 && !strings.Contains(registryHost, ":") {
			registryHost += ":" + strconv.Itoa(account.Port)
		}
		registryHost = strings.Trim(registryHost, "/")
		if registryHost == "" {
			return firstNonEmptyPipelineString(strings.TrimSpace(run.PlannedImageRef), fmt.Sprintf("%s/%s:%s", project, repository, tag))
		}
		return fmt.Sprintf("%s/%s/%s:%s", registryHost, project, repository, tag)
	}
	registryHost := parsed.Host
	if parsed.Port() == "" && account.Port > 0 {
		if (parsed.Scheme == "https" && account.Port != 443) || (parsed.Scheme == "http" && account.Port != 80) || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			registryHost = parsed.Hostname() + ":" + strconv.Itoa(account.Port)
		} else {
			registryHost = parsed.Hostname()
		}
	} else if (parsed.Scheme == "https" && parsed.Port() == "443") || (parsed.Scheme == "http" && parsed.Port() == "80") {
		registryHost = parsed.Hostname()
	}
	registryHost = strings.Trim(registryHost, "/")
	if registryHost == "" {
		return firstNonEmptyPipelineString(strings.TrimSpace(run.PlannedImageRef), fmt.Sprintf("%s/%s:%s", project, repository, tag))
	}
	return fmt.Sprintf("%s/%s/%s:%s", registryHost, project, repository, tag)
}

func (s *PipelineService) applyRunUpdates(run *model.PipelineRun, runUpdates map[string]interface{}, requestUpdates map[string]interface{}) error {
	if run == nil {
		return fmt.Errorf("流水线运行不能为空")
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if len(runUpdates) > 0 {
			if err := tx.Model(&model.PipelineRun{}).Where("id = ?", run.ID).Updates(runUpdates).Error; err != nil {
				return err
			}
		}
		if len(requestUpdates) > 0 {
			if err := tx.Model(&model.DeployRequest{}).Where("id = ?", run.RequestID).Updates(requestUpdates).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	latest, err := s.pipelineDao.GetPipelineRunByID(run.ID)
	if err != nil {
		return err
	}
	*run = *latest
	return nil
}

func (s *PipelineService) failPipelineStage(run *model.PipelineRun, record *model.PipelineStageRecord, stage, errMsg, detailJSON string) error {
	if strings.TrimSpace(detailJSON) == "" {
		detailJSON = marshalPipelineJSONString(map[string]string{"error": errMsg})
	}
	if record != nil {
		if err := s.completeStageRecord(record, model.PipelineStageStatusFailed, detailJSON, errMsg); err != nil {
			log.Printf("pipeline stage record failure update failed: pipelineRunID=%d stage=%s err=%v", run.ID, stage, err)
		}
	}
	if err := s.transitionStage(run, stage, model.PipelineStatusFailed, errMsg); err != nil {
		return fmt.Errorf("%s；并且更新流水线状态失败: %v", errMsg, err)
	}
	log.Printf("pipeline stage failed: pipelineRunID=%d requestID=%d stage=%s err=%s", run.ID, run.RequestID, stage, errMsg)
	return fmt.Errorf("%s", errMsg)
}

func (s *PipelineService) executeDeployWithSkipNotify(requestID uint, comment string) (*model.DeployRequest, *model.ExecutionRecord, error) {
	service := newDeployService(s.db)
	deployRequest, err := service.dao.GetDeployRequestByID(requestID)
	if err != nil {
		return nil, nil, err
	}
	return service.executeDeployRequestInternal(deployRequest, comment, true)
}

func buildPipelineRunResponse(run *model.PipelineRun, records []model.PipelineStageRecord) *model.PipelineRunResponse {
	if run == nil {
		return nil
	}
	resp := &model.PipelineRunResponse{
		ID:                     run.ID,
		RequestID:              run.RequestID,
		ApplicationID:          run.ApplicationID,
		Status:                 run.Status,
		CurrentStage:           run.CurrentStage,
		JenkinsServerID:        run.JenkinsServerID,
		JenkinsJobNameSnapshot: run.JenkinsJobNameSnapshot,
		GitRef:                 run.GitRef,
		BuildParamsJSON:        run.BuildParamsJSON,
		JenkinsQueueID:         run.JenkinsQueueID,
		JenkinsBuildNumber:     run.JenkinsBuildNumber,
		JenkinsBuildURL:        run.JenkinsBuildURL,
		HarborServerID:         run.HarborServerID,
		HarborProject:          run.HarborProject,
		HarborRepository:       run.HarborRepository,
		ArtifactTag:            run.ArtifactTag,
		ArtifactDigest:         run.ArtifactDigest,
		PlannedImageRef:        run.PlannedImageRef,
		FinalImageRef:          run.FinalImageRef,
		ScanPolicyJSON:         run.ScanPolicyJSON,
		ScanReportJSON:         run.ScanReportJSON,
		LastError:              run.LastError,
		StartedAt:              run.StartedAt,
		FinishedAt:             run.FinishedAt,
		CreatedAt:              run.CreatedAt,
		UpdatedAt:              run.UpdatedAt,
	}
	if len(records) > 0 {
		resp.Stages = make([]model.PipelineStageRecordResponse, 0, len(records))
		for _, record := range records {
			resp.Stages = append(resp.Stages, model.PipelineStageRecordResponse{
				ID:            record.ID,
				PipelineRunID: record.PipelineRunID,
				Stage:         record.Stage,
				Status:        record.Status,
				ExternalID:    record.ExternalID,
				ExternalURL:   record.ExternalURL,
				DetailJSON:    record.DetailJSON,
				ErrorMessage:  record.ErrorMessage,
				StartedAt:     record.StartedAt,
				FinishedAt:    record.FinishedAt,
				RetryCount:    record.RetryCount,
				CreatedAt:     record.CreatedAt,
				UpdatedAt:     record.UpdatedAt,
			})
		}
	}
	return resp
}

func parseBuildParams(raw string) (map[string]string, error) {
	params := map[string]string{}
	if strings.TrimSpace(raw) == "" {
		return params, nil
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	for key, value := range payload {
		key = strings.TrimSpace(key)
		if key == "" || value == nil {
			continue
		}
		params[key] = fmt.Sprint(value)
	}
	return params, nil
}

func parseScanPolicy(raw string) (*ScanPolicy, error) {
	policy := &ScanPolicy{MaxCritical: 0, MaxHigh: 0}
	if strings.TrimSpace(raw) == "" {
		return policy, nil
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	policy.MaxCritical = intValueFromMap(payload, "max_critical", "maxCritical")
	policy.MaxHigh = intValueFromMap(payload, "max_high", "maxHigh")
	policy.MaxMedium = intValueFromMap(payload, "max_medium", "maxMedium")
	return policy, nil
}

func intValueFromMap(payload map[string]interface{}, keys ...string) int {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case float64:
			return int(typed)
		case float32:
			return int(typed)
		case int:
			return typed
		case int32:
			return int(typed)
		case int64:
			return int(typed)
		case string:
			parsed, err := strconv.Atoi(strings.TrimSpace(typed))
			if err == nil {
				return parsed
			}
		}
	}
	return 0
}

func marshalPipelineJSON(value interface{}) (string, error) {
	if value == nil {
		return "", nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func marshalPipelineJSONString(value interface{}) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func firstNonEmptyPipelineString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
