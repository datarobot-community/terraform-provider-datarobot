package provider

import (
	"bytes"
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

func TestEmitArtifactBuildLogLine(t *testing.T) {
	var buf bytes.Buffer
	oldWriter := artifactBuildLogWriter
	artifactBuildLogWriter = &buf
	defer func() {
		artifactBuildLogWriter = oldWriter
	}()

	emitArtifactBuildLogLine("build-1", "step 1")
	emitArtifactBuildLogLine("build-2", "step 1")
	// A record carrying a stack trace must keep every continuation line attributed.
	emitArtifactBuildLogLine("build-1", "boom\n  at frame 1")
	emitArtifactBuildLogLine("", "no build id")
	emitArtifactBuildLogLine("build-1", "")

	got := buf.String()
	want := "[build build-1] step 1\n" +
		"[build build-2] step 1\n" +
		"[build build-1] boom\n[build build-1]   at frame 1\n" +
		"no build id\n"
	if got != want {
		t.Fatalf("emitArtifactBuildLogLine() wrote\n%q\nwant\n%q", got, want)
	}
}

// Two builds running in the same apply share one writer; the build label is what makes
// their interleaved lines separable.
func TestEmitArtifactBuildLogLineDistinguishesConcurrentBuilds(t *testing.T) {
	var buf bytes.Buffer
	oldWriter := artifactBuildLogWriter
	artifactBuildLogWriter = &buf
	defer func() {
		artifactBuildLogWriter = oldWriter
	}()

	for _, buildID := range []string{"build-a", "build-b", "build-a"} {
		emitArtifactBuildLogLine(buildID, "#7 extracting sha256:abc 0.0s done")
	}

	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), buf.String())
	}
	for i, wantBuild := range []string{"build-a", "build-b", "build-a"} {
		if !strings.HasPrefix(lines[i], "[build "+wantBuild+"] ") {
			t.Errorf("line %d = %q, want prefix for %s", i, lines[i], wantBuild)
		}
	}
}

// The Update draft path patches an existing artifact, so the upload line must not claim
// the artifact was just created.
func TestArtifactApplyProgressMessages(t *testing.T) {
	var buf bytes.Buffer
	oldWriter := artifactBuildLogWriter
	artifactBuildLogWriter = &buf
	defer func() {
		artifactBuildLogWriter = oldWriter
	}()

	artifactApplyProgressUploading("art-1")
	artifactApplyProgressBuildStatus("art-1", "build-1", "IN_PROGRESS")

	got := buf.String()
	want := "Uploading code to artifact with id art-1...\n" +
		"Build build-1 for artifact art-1: IN_PROGRESS\n"
	if got != want {
		t.Fatalf("progress output =\n%q\nwant\n%q", got, want)
	}
	if strings.Contains(got, "Created artifact") {
		t.Error("upload progress must not claim the artifact was created")
	}
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
