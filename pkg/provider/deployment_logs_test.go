package provider

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
)

func TestDeploymentErrorMessageWithLogs(t *testing.T) {
	t.Run("includes logs and UI link on success", func(t *testing.T) {
		msg := deploymentErrorMessageWithLogs("deployment has errored", "line1\nline2", nil, "https://app.datarobot.com/console-nextgen/deployments/dep-1/activity-log/otel-logs")

		for _, want := range []string{
			"deployment has errored",
			deploymentLogsSeparator,
			"line1\nline2",
			"See full logs at: https://app.datarobot.com/console-nextgen/deployments/dep-1/activity-log/otel-logs",
		} {
			if !strings.Contains(msg, want) {
				t.Errorf("expected message to contain %q, got:\n%s", want, msg)
			}
		}
	})

	t.Run("falls back to UI link when log retrieval fails", func(t *testing.T) {
		msg := deploymentErrorMessageWithLogs("deployment has errored", "", errors.New("boom"), "https://app.datarobot.com/console-nextgen/deployments/dep-1/activity-log/otel-logs")

		for _, want := range []string{
			"deployment has errored",
			"failed to retrieve deployment logs: boom",
			"See full logs at: https://app.datarobot.com/console-nextgen/deployments/dep-1/activity-log/otel-logs",
		} {
			if !strings.Contains(msg, want) {
				t.Errorf("expected message to contain %q, got:\n%s", want, msg)
			}
		}
	})
}

func TestWaitForDeploymentStatusErroredIncludesLogsAndUIURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	r := &DeploymentResource{provider: &Provider{service: mockService}}

	mockService.EXPECT().GetDeployment(gomock.Any(), "dep-1").Return(&client.Deployment{Status: "error"}, nil)
	mockService.EXPECT().GetDeploymentLogs(gomock.Any(), "dep-1").Return("[2026-07-09T16:14:50Z] ERROR: failed to load model", nil)
	mockService.EXPECT().BaseURL().Return("https://app.datarobot.com")

	_, err := r.waitForDeploymentToBeReady(context.Background(), "dep-1")
	if err == nil {
		t.Fatal("expected an error")
	}

	for _, want := range []string{
		"deployment has errored",
		deploymentLogsSeparator,
		"failed to load model",
		"See full logs at: https://app.datarobot.com/console-nextgen/deployments/dep-1/activity-log/otel-logs",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected error to contain %q, got:\n%s", want, err.Error())
		}
	}
}

func TestWaitForDeploymentStatusErroredLogRetrievalFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	r := &DeploymentResource{provider: &Provider{service: mockService}}

	mockService.EXPECT().GetDeployment(gomock.Any(), "dep-1").Return(&client.Deployment{Status: "error"}, nil)
	mockService.EXPECT().GetDeploymentLogs(gomock.Any(), "dep-1").Return("", errors.New("logs unavailable"))
	mockService.EXPECT().BaseURL().Return("https://app.datarobot.com")

	_, err := r.waitForDeploymentToBeReady(context.Background(), "dep-1")
	if err == nil {
		t.Fatal("expected an error")
	}

	for _, want := range []string{
		"deployment has errored",
		"failed to retrieve deployment logs: logs unavailable",
		"See full logs at: https://app.datarobot.com/console-nextgen/deployments/dep-1/activity-log/otel-logs",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected error to contain %q, got:\n%s", want, err.Error())
		}
	}
}
