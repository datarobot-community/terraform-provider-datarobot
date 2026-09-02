package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
)

func TestWaitForModelReplacementTaskToCompleteTerminalFailure(t *testing.T) {
	for _, status := range []string{"ERROR", "ABORTED", "EXPIRED"} {
		t.Run(status, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mock_client.NewMockService(ctrl)
			mockService.EXPECT().
				GetTaskStatus(gomock.Any(), "task-1").
				Return(&client.TaskStatusResponse{Status: status, Message: "pod failed to start"}, nil)

			err := waitForModelReplacementTaskToComplete(context.Background(), mockService, "task-1")
			if err == nil {
				t.Fatalf("expected error for terminal status %s", status)
			}
			if !strings.Contains(err.Error(), "pod failed to start") {
				t.Errorf("expected task message in error, got: %s", err.Error())
			}
		})
	}
}

func TestWaitForModelReplacementTaskToCompleteTerminalFailureNoMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	mockService.EXPECT().
		GetTaskStatus(gomock.Any(), "task-1").
		Return(&client.TaskStatusResponse{Status: "ABORTED"}, nil)

	err := waitForModelReplacementTaskToComplete(context.Background(), mockService, "task-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "ABORTED") {
		t.Errorf("expected fallback message to include status, got: %s", err.Error())
	}
}

func TestWaitForModelReplacementTaskToCompleteSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	mockService.EXPECT().
		GetTaskStatus(gomock.Any(), "task-1").
		Return(&client.TaskStatusResponse{Status: "COMPLETED"}, nil)

	if err := waitForModelReplacementTaskToComplete(context.Background(), mockService, "task-1"); err != nil {
		t.Fatalf("expected success, got: %s", err.Error())
	}
}
