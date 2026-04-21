package scheduler

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	deploydao "dodevops-api/api/deploy/dao"
	deploymodel "dodevops-api/api/deploy/model"
	deployservice "dodevops-api/api/deploy/service"
	"dodevops-api/common"
	"dodevops-api/common/config"
)

type DeployApprovalSyncScheduler struct {
	mu      sync.Mutex
	ticker  *time.Ticker
	stopCh  chan struct{}
	running bool
}

func NewDeployApprovalSyncScheduler() *DeployApprovalSyncScheduler {
	return &DeployApprovalSyncScheduler{}
}

func (s *DeployApprovalSyncScheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	interval := 30 * time.Second
	if config.Config != nil && config.Config.DingtalkApproval.PollIntervalSeconds > 0 {
		interval = time.Duration(config.Config.DingtalkApproval.PollIntervalSeconds) * time.Second
	}
	s.ticker = time.NewTicker(interval)
	s.stopCh = make(chan struct{})
	s.running = true

	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.syncOnce()
			case <-s.stopCh:
				return
			}
		}
	}()

	log.Printf("部署审批同步调度器已启动（每 %s 执行）", interval.String())
	return nil
}

func (s *DeployApprovalSyncScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}
	if s.ticker != nil {
		s.ticker.Stop()
	}
	if s.stopCh != nil {
		close(s.stopCh)
	}
	s.running = false
	log.Println("部署审批同步调度器已停止")
}

func (s *DeployApprovalSyncScheduler) syncOnce() {
	db := common.GetDB()
	if db == nil {
		return
	}

	dao := deploydao.NewDeployDao(db)
	requests, err := dao.ListPendingApprovalSyncRequests(50)
	if err != nil {
		log.Printf("[DeployApprovalSync] 查询待同步审批申请失败: %v", err)
		return
	}
	if len(requests) == 0 {
		return
	}

	client := deployservice.NewDingtalkApprovalService()
	if !client.IsConfigured() {
		return
	}

	for _, req := range requests {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		detail, err := client.GetProcessInstance(ctx, req.DingtalkProcessInstanceID)
		cancel()
		if err != nil {
			log.Printf("[DeployApprovalSync] 查询审批实例失败 requestNo=%s err=%v", req.RequestNo, err)
			continue
		}

		approvalStatus, requestStatus := mapApprovalFromDingtalkResult(detail.Status, detail.Result)
		updates := map[string]interface{}{
			"updated_at":                time.Now(),
			"approval_dispatch_status":  deploymodel.ApprovalDispatchStatusDispatched,
			"approval_dispatch_message": fmt.Sprintf("钉钉审批实例状态: status=%s result=%s", detail.Status, detail.Result),
		}
		if approvalStatus != "" {
			updates["approval_status"] = approvalStatus
		}
		if requestStatus != "" {
			updates["request_status"] = requestStatus
		}
		if approvalStatus == deploymodel.ApprovalStatusApproved {
			updates["approved_at"] = time.Now()
		}
		if approvalStatus == deploymodel.ApprovalStatusRejected {
			updates["rejected_at"] = time.Now()
		}

		if err := dao.UpdateDeployRequest(req.ID, updates); err != nil {
			log.Printf("[DeployApprovalSync] 回写审批状态失败 requestNo=%s err=%v", req.RequestNo, err)
			continue
		}
		if approvalStatus == deploymodel.ApprovalStatusApproved && req.ExecutionStatus == deploymodel.ExecutionStatusPending {
			if req.WorkflowKind == deploymodel.WorkflowKindBuildDeploy {
				log.Printf("[DeployApprovalSync] 审批通过，build_deploy 工作流交由流水线调度器执行 requestNo=%s", req.RequestNo)
				continue
			}
			if _, _, err := deployservice.AutoExecuteApprovedDeployRequest(db, req.ID, "auto execute after scheduled dingtalk approval sync"); err != nil {
				log.Printf("[DeployApprovalSync] 审批通过后自动执行失败 requestNo=%s err=%v", req.RequestNo, err)
				continue
			}
			log.Printf("[DeployApprovalSync] 审批通过后已自动执行 requestNo=%s", req.RequestNo)
		}
	}
}

func mapApprovalFromDingtalkResult(status, result string) (string, string) {
	upperStatus := strings.ToUpper(strings.TrimSpace(status))
	upperResult := strings.ToUpper(strings.TrimSpace(result))

	if upperResult == "AGREE" || upperResult == "APPROVED" || upperResult == "PASS" {
		return deploymodel.ApprovalStatusApproved, deploymodel.DeployRequestStatusApproved
	}
	if upperResult == "REFUSE" || upperResult == "REFUSED" || upperResult == "REJECT" || upperResult == "REJECTED" {
		return deploymodel.ApprovalStatusRejected, deploymodel.DeployRequestStatusRejected
	}
	if upperStatus == "COMPLETED" && upperResult == "" {
		return deploymodel.ApprovalStatusApproved, deploymodel.DeployRequestStatusApproved
	}
	if upperStatus == "TERMINATED" || upperStatus == "CANCELED" || upperStatus == "CANCELLED" {
		return deploymodel.ApprovalStatusRejected, deploymodel.DeployRequestStatusRejected
	}
	return "", ""
}
