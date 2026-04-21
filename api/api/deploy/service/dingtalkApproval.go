package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"dodevops-api/common/config"
)

const dingtalkAPIBaseURL = "https://api.dingtalk.com"

type IDingtalkApprovalService interface {
	IsConfigured() bool
	GetAccessToken(ctx context.Context) (string, error)
	CreateProcessInstance(ctx context.Context, req *DingtalkCreateProcessInstanceRequest) (*DingtalkCreateProcessInstanceResponse, error)
	GetProcessInstance(ctx context.Context, processInstanceID string) (*DingtalkProcessInstanceDetailResponse, error)
}

type DingtalkApprovalService struct {
	httpClient     *http.Client
	baseURL        string
	accessTokenMu  sync.Mutex
	accessToken    string
	tokenExpiresAt time.Time
}

type DingtalkCreateProcessInstanceRequest struct {
	ProcessCode         string                       `json:"processCode"`
	OriginatorUserID    string                       `json:"originatorUserId"`
	DeptID              int64                        `json:"deptId,omitempty"`
	Approvers           []DingtalkApprovalNode       `json:"approvers,omitempty"`
	FormComponentValues []DingtalkFormComponentValue `json:"formComponentValues"`
}

type DingtalkApprovalNode struct {
	ActionType string   `json:"actionType"`
	UserIDs    []string `json:"userIds"`
}

type DingtalkFormComponentValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type DingtalkCreateProcessInstanceResponse struct {
	ProcessInstanceID string `json:"processInstanceId"`
	InstanceID        string `json:"instanceId"`
}

type DingtalkProcessInstanceDetailResponse struct {
	ProcessInstanceID string `json:"processInstanceId"`
	Status            string `json:"status"`
	Result            string `json:"result"`
}

type dingtalkAccessTokenResponse struct {
	AccessToken string `json:"accessToken"`
	ExpireIn    int64  `json:"expireIn"`
}

type dingtalkAPIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type dingtalkProcessInstanceEnvelope struct {
	ProcessInstanceID string `json:"processInstanceId"`
}

type dingtalkProcessInstanceQueryEnvelope struct {
	Result dingtalkProcessInstanceQueryResult `json:"result"`
}

type dingtalkProcessInstanceQueryResult struct {
	ProcessInstanceID string `json:"processInstanceId"`
	Status            string `json:"status"`
	Result            string `json:"result"`
}

func NewDingtalkApprovalService() IDingtalkApprovalService {
	return &DingtalkApprovalService{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    dingtalkAPIBaseURL,
	}
}

func (s *DingtalkApprovalService) IsConfigured() bool {
	cfg := config.Config.DingtalkApproval
	return cfg.ClientID != "" && cfg.ClientSecret != ""
}

func (s *DingtalkApprovalService) GetAccessToken(ctx context.Context) (string, error) {
	if !s.IsConfigured() {
		return "", fmt.Errorf("钉钉审批未配置 client_id/client_secret")
	}

	now := time.Now()
	s.accessTokenMu.Lock()
	if s.accessToken != "" && now.Before(s.tokenExpiresAt.Add(-30*time.Second)) {
		token := s.accessToken
		s.accessTokenMu.Unlock()
		return token, nil
	}
	s.accessTokenMu.Unlock()

	payload, err := json.Marshal(map[string]string{
		"appKey":    config.Config.DingtalkApproval.ClientID,
		"appSecret": config.Config.DingtalkApproval.ClientSecret,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/v1.0/oauth2/accessToken", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("获取钉钉 access token 失败: http %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp dingtalkAccessTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("钉钉 access token 为空")
	}
	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpireIn) * time.Second)
	s.accessTokenMu.Lock()
	s.accessToken = tokenResp.AccessToken
	s.tokenExpiresAt = expiresAt
	s.accessTokenMu.Unlock()
	return tokenResp.AccessToken, nil
}

func (s *DingtalkApprovalService) CreateProcessInstance(ctx context.Context, req *DingtalkCreateProcessInstanceRequest) (*DingtalkCreateProcessInstanceResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("审批实例请求不能为空")
	}
	if req.ProcessCode == "" {
		return nil, fmt.Errorf("钉钉审批流程 processCode 未配置")
	}

	token, err := s.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/v1.0/workflow/processInstances", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-acs-dingtalk-access-token", token)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("发起钉钉审批实例失败: http %d: %s", resp.StatusCode, string(body))
	}

	var result DingtalkCreateProcessInstanceResponse
	if err := json.Unmarshal(body, &result); err != nil {
		var apiErr dingtalkAPIError
		if json.Unmarshal(body, &apiErr) == nil && (apiErr.Code != "" || apiErr.Message != "") {
			return nil, fmt.Errorf("发起钉钉审批实例失败: %s %s", apiErr.Code, apiErr.Message)
		}
		return nil, err
	}
	if result.ProcessInstanceID == "" {
		var envelope dingtalkProcessInstanceEnvelope
		if err := json.Unmarshal(body, &envelope); err == nil {
			result.ProcessInstanceID = envelope.ProcessInstanceID
		}
	}
	if result.ProcessInstanceID == "" {
		result.ProcessInstanceID = strings.TrimSpace(result.InstanceID)
	}
	return &result, nil
}

func (s *DingtalkApprovalService) GetProcessInstance(ctx context.Context, processInstanceID string) (*DingtalkProcessInstanceDetailResponse, error) {
	if strings.TrimSpace(processInstanceID) == "" {
		return nil, fmt.Errorf("审批实例ID不能为空")
	}

	token, err := s.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	queryURL := s.baseURL + "/v1.0/workflow/processInstances?processInstanceId=" + url.QueryEscape(strings.TrimSpace(processInstanceID))
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("x-acs-dingtalk-access-token", token)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("查询钉钉审批实例失败: http %d: %s", resp.StatusCode, string(body))
	}

	var result DingtalkProcessInstanceDetailResponse
	if err := json.Unmarshal(body, &result); err == nil && (result.ProcessInstanceID != "" || result.Status != "" || result.Result != "") {
		return &result, nil
	}

	var envelope dingtalkProcessInstanceQueryEnvelope
	if err := json.Unmarshal(body, &envelope); err == nil && (envelope.Result.ProcessInstanceID != "" || envelope.Result.Status != "" || envelope.Result.Result != "") {
		return &DingtalkProcessInstanceDetailResponse{
			ProcessInstanceID: envelope.Result.ProcessInstanceID,
			Status:            envelope.Result.Status,
			Result:            envelope.Result.Result,
		}, nil
	}

	var apiErr dingtalkAPIError
	if json.Unmarshal(body, &apiErr) == nil && (apiErr.Code != "" || apiErr.Message != "") {
		return nil, fmt.Errorf("查询钉钉审批实例失败: %s %s", apiErr.Code, apiErr.Message)
	}
	return nil, fmt.Errorf("查询钉钉审批实例失败: 无法解析响应: %s", string(body))
}
