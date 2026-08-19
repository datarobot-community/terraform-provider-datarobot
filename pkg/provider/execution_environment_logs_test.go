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

func TestExecutionEnvironmentErrorMessageWithLogs(t *testing.T) {
	t.Run("includes logs and UI link on success", func(t *testing.T) {
		msg := executionEnvironmentErrorMessageWithLogs("line1\nline2", nil, "https://app.datarobot.com/registry/execution-environments/env-1/buildsLogs?executionEnvironmentVersion=ver-1")

		for _, want := range []string{
			"execution environment failed to build",
			executionEnvironmentLogsSeparator,
			"line1\nline2",
			"See full logs at: https://app.datarobot.com/registry/execution-environments/env-1/buildsLogs?executionEnvironmentVersion=ver-1",
		} {
			if !strings.Contains(msg, want) {
				t.Errorf("expected message to contain %q, got:\n%s", want, msg)
			}
		}
	})

	t.Run("falls back to UI link when log retrieval fails", func(t *testing.T) {
		msg := executionEnvironmentErrorMessageWithLogs("", errors.New("boom"), "https://app.datarobot.com/registry/execution-environments/env-1/buildsLogs?executionEnvironmentVersion=ver-1")

		for _, want := range []string{
			"execution environment failed to build",
			"failed to retrieve build logs: boom",
			"See full logs at: https://app.datarobot.com/registry/execution-environments/env-1/buildsLogs?executionEnvironmentVersion=ver-1",
		} {
			if !strings.Contains(msg, want) {
				t.Errorf("expected message to contain %q, got:\n%s", want, msg)
			}
		}
	})

	t.Run("explains no logs yet when retrieval succeeds but returns nothing", func(t *testing.T) {
		msg := executionEnvironmentErrorMessageWithLogs("", nil, "https://app.datarobot.com/registry/execution-environments/env-1/buildsLogs?executionEnvironmentVersion=ver-1")

		for _, want := range []string{
			"execution environment failed to build",
			"No build logs are available yet",
			"See full logs at: https://app.datarobot.com/registry/execution-environments/env-1/buildsLogs?executionEnvironmentVersion=ver-1",
		} {
			if !strings.Contains(msg, want) {
				t.Errorf("expected message to contain %q, got:\n%s", want, msg)
			}
		}
		if strings.Contains(msg, "failed to retrieve build logs") {
			t.Errorf("expected message to not treat empty logs as a retrieval failure, got:\n%s", msg)
		}
	})
}

func TestWaitForExecutionEnvironmentToBeReadyFailedIncludesLogsAndUIURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)

	mockService.EXPECT().GetExecutionEnvironment(gomock.Any(), "env-1").Return(&client.ExecutionEnvironment{
		ID:            "env-1",
		LatestVersion: client.ExecutionEnvironmentVersion{ID: "ver-1"},
	}, nil)
	mockService.EXPECT().GetExecutionEnvironmentVersion(gomock.Any(), "env-1", "ver-1").Return(&client.ExecutionEnvironmentVersion{
		ID:          "ver-1",
		BuildStatus: "failed",
		BuildID:     "build-1",
	}, nil)
	mockService.EXPECT().GetExecutionEnvironmentVersionBuildLog(gomock.Any(), "env-1", "ver-1", "build-1").Return("line1\nline2: pip install failed", nil)
	mockService.EXPECT().BaseURL().Return("https://app.datarobot.com")

	_, err := waitForExecutionEnvironmentToBeReady(context.Background(), mockService, "env-1")
	if err == nil {
		t.Fatal("expected an error")
	}

	for _, want := range []string{
		"execution environment failed to build",
		executionEnvironmentLogsSeparator,
		"pip install failed",
		"See full logs at: https://app.datarobot.com/registry/execution-environments/env-1/buildsLogs?executionEnvironmentVersion=ver-1",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected error to contain %q, got:\n%s", want, err.Error())
		}
	}
}

func TestWaitForExecutionEnvironmentToBeReadyFailedNoLogsYet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)

	mockService.EXPECT().GetExecutionEnvironment(gomock.Any(), "env-1").Return(&client.ExecutionEnvironment{
		ID:            "env-1",
		LatestVersion: client.ExecutionEnvironmentVersion{ID: "ver-1"},
	}, nil)
	mockService.EXPECT().GetExecutionEnvironmentVersion(gomock.Any(), "env-1", "ver-1").Return(&client.ExecutionEnvironmentVersion{
		ID:          "ver-1",
		BuildStatus: "failed",
		BuildID:     "build-1",
	}, nil)
	// Build log dump support is still being rolled out on the backend, so the OTel
	// endpoint currently responds successfully with zero log lines rather than an error.
	mockService.EXPECT().GetExecutionEnvironmentVersionBuildLog(gomock.Any(), "env-1", "ver-1", "build-1").Return("", nil)
	mockService.EXPECT().BaseURL().Return("https://app.datarobot.com")

	_, err := waitForExecutionEnvironmentToBeReady(context.Background(), mockService, "env-1")
	if err == nil {
		t.Fatal("expected an error")
	}

	for _, want := range []string{
		"execution environment failed to build",
		"No build logs are available yet",
		"See full logs at: https://app.datarobot.com/registry/execution-environments/env-1/buildsLogs?executionEnvironmentVersion=ver-1",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected error to contain %q, got:\n%s", want, err.Error())
		}
	}
}

func TestWaitForExecutionEnvironmentToBeReadyPassesBuildIDNotVersionID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)

	mockService.EXPECT().GetExecutionEnvironment(gomock.Any(), "env-1").Return(&client.ExecutionEnvironment{
		ID:            "env-1",
		LatestVersion: client.ExecutionEnvironmentVersion{ID: "ver-1"},
	}, nil)
	mockService.EXPECT().GetExecutionEnvironmentVersion(gomock.Any(), "env-1", "ver-1").Return(&client.ExecutionEnvironmentVersion{
		ID:          "ver-1",
		BuildStatus: "failed",
		BuildID:     "build-1",
	}, nil)
	// Distinct values for versionId ("ver-1") and buildId ("build-1") — a mock
	// expectation requiring exactly these two args (in this order) fails if the
	// resource code ever passes versionId as the build ID again.
	mockService.EXPECT().GetExecutionEnvironmentVersionBuildLog(gomock.Any(), "env-1", "ver-1", "build-1").Return("some log", nil)
	mockService.EXPECT().BaseURL().Return("https://app.datarobot.com")

	_, err := waitForExecutionEnvironmentToBeReady(context.Background(), mockService, "env-1")
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestWaitForExecutionEnvironmentToBeReadyLogRetrievalFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)

	mockService.EXPECT().GetExecutionEnvironment(gomock.Any(), "env-1").Return(&client.ExecutionEnvironment{
		ID:            "env-1",
		LatestVersion: client.ExecutionEnvironmentVersion{ID: "ver-1"},
	}, nil)
	mockService.EXPECT().GetExecutionEnvironmentVersion(gomock.Any(), "env-1", "ver-1").Return(&client.ExecutionEnvironmentVersion{
		ID:          "ver-1",
		BuildStatus: "failed",
		BuildID:     "build-1",
	}, nil)
	mockService.EXPECT().GetExecutionEnvironmentVersionBuildLog(gomock.Any(), "env-1", "ver-1", "build-1").Return("", errors.New("logs unavailable"))
	mockService.EXPECT().BaseURL().Return("https://app.datarobot.com")

	_, err := waitForExecutionEnvironmentToBeReady(context.Background(), mockService, "env-1")
	if err == nil {
		t.Fatal("expected an error")
	}

	for _, want := range []string{
		"execution environment failed to build",
		"failed to retrieve build logs: logs unavailable",
		"See full logs at: https://app.datarobot.com/registry/execution-environments/env-1/buildsLogs?executionEnvironmentVersion=ver-1",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected error to contain %q, got:\n%s", want, err.Error())
		}
	}
}
