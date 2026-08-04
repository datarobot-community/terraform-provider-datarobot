package provider

import (
	"errors"
	"strings"
	"testing"
)

func TestArtifactBuildErrorMessageWithLogs(t *testing.T) {
	t.Run("includes logs and UI link on success", func(t *testing.T) {
		logsURL := "https://app.datarobot.com/registry/service-artifacts/repo-1/artifacts/art-1/build-log"
		msg := artifactBuildErrorMessageWithLogs(
			"wait for artifact build: artifact build build-1 ended with status FAILED",
			"line1\nline2",
			nil,
			logsURL,
		)

		for _, want := range []string{
			"wait for artifact build: artifact build build-1 ended with status FAILED",
			artifactBuildLogsSeparator,
			"line1\nline2",
			"See full logs at: " + logsURL,
		} {
			if !strings.Contains(msg, want) {
				t.Errorf("expected message to contain %q, got:\n%s", want, msg)
			}
		}
	})

	t.Run("falls back to UI link when log retrieval fails", func(t *testing.T) {
		logsURL := "https://app.datarobot.com/registry/service-artifacts/repo-1/artifacts/art-1/build-log"
		msg := artifactBuildErrorMessageWithLogs(
			"trigger artifact build: boom",
			"",
			errors.New("logs unavailable"),
			logsURL,
		)

		for _, want := range []string{
			"trigger artifact build: boom",
			"failed to retrieve artifact build logs: logs unavailable",
			"See full logs at: " + logsURL,
		} {
			if !strings.Contains(msg, want) {
				t.Errorf("expected message to contain %q, got:\n%s", want, msg)
			}
		}
	})
}

func TestArtifactBuildLogsURL(t *testing.T) {
	got := artifactBuildLogsURL(
		"https://app.datarobot.com",
		"repo-1",
		"art-1",
	)
	want := "https://app.datarobot.com/registry/service-artifacts/repo-1/artifacts/art-1/build-log"
	if got != want {
		t.Fatalf("artifactBuildLogsURL() = %q, want %q", got, want)
	}
}
