package service

import (
	"strings"
	"testing"

	deploymodel "dodevops-api/api/deploy/model"
)

func TestExecuteDeployRequestInternalRequiresApprovedStatus(t *testing.T) {
	service := &DeployService{}
	req := &deploymodel.DeployRequest{
		ApprovalStatus:  deploymodel.ApprovalStatusPending,
		ExecutionStatus: deploymodel.ExecutionStatusPending,
	}

	_, _, err := service.executeDeployRequestInternal(req, "auto execute")
	if err == nil {
		t.Fatal("expected approval guard error")
	}
	if !strings.Contains(err.Error(), "未审批通过") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteDeployRequestInternalRejectsSucceededRequest(t *testing.T) {
	service := &DeployService{}
	req := &deploymodel.DeployRequest{
		ApprovalStatus:  deploymodel.ApprovalStatusApproved,
		ExecutionStatus: deploymodel.ExecutionStatusSucceeded,
	}

	_, _, err := service.executeDeployRequestInternal(req, "auto execute")
	if err == nil {
		t.Fatal("expected execution status guard error")
	}
	if !strings.Contains(err.Error(), "已执行成功") {
		t.Fatalf("unexpected error: %v", err)
	}
}
