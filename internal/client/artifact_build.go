// Artifact image build client methods ported from cli/internal/workload/build.go.

package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Artifact build status values returned by the Workload API (external enum).
// BUILT is a non-terminal intermediate status during image push/backup phases.
const (
	ArtifactBuildStatusPending    = "PENDING"
	ArtifactBuildStatusInProgress = "IN_PROGRESS"
	ArtifactBuildStatusBuilt      = "BUILT"
	ArtifactBuildStatusCompleted  = "COMPLETED"
	ArtifactBuildStatusFailed     = "FAILED"
	ArtifactBuildStatusCancelled  = "CANCELLED"
)

const (
	ArtifactBuildPollIntervalEnvVar   = "DATAROBOT_ARTIFACT_BUILD_POLL_INTERVAL"
	ArtifactBuildPollTimeoutEnvVar    = "DATAROBOT_ARTIFACT_BUILD_POLL_TIMEOUT"
	ArtifactBuildLogsTailLinesEnvVar  = "DATAROBOT_ARTIFACT_BUILD_LOGS_TAIL_LINES"
	defaultArtifactBuildPollInterval  = 10 * time.Second
	defaultArtifactBuildPollTimeout   = 10 * time.Minute
	defaultArtifactBuildLogsTailLines = 30
)

// ArtifactBuildTriggerResponse is the body returned by POST /artifacts/{id}/builds/.
type ArtifactBuildTriggerResponse struct {
	BuildIDs []string `json:"buildIds"`
}

// ArtifactBuild is a single image build resource owned by the Workload API.
type ArtifactBuild struct {
	ID         string `json:"id"`
	Name       string `json:"name,omitempty"`
	ArtifactID string `json:"artifactId"`
	Status     string `json:"status"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

type WaitForArtifactBuildOptions struct {
	PollInterval  time.Duration
	Timeout       time.Duration
	OnOtelLogLine func(OtelLogEntry)
	OnPoll        func(*ArtifactBuild)
}

type ArtifactBuildFailedError struct {
	BuildID string
	Status  string
}

// ArtifactBuildLogEntry is one JSONL record from GET /artifacts/{id}/builds/{buildId}/logs.
type ArtifactBuildLogEntry struct {
	Asctime   string `json:"asctime"`
	Levelname string `json:"levelname"`
	Name      string `json:"name,omitempty"`
	Message   string `json:"message"`
	BuildID   string `json:"build_id,omitempty"`
}

func (e *ArtifactBuildFailedError) Error() string {
	return fmt.Sprintf("artifact build %s ended with status %s", e.BuildID, e.Status)
}

type ArtifactBuildTimeoutError struct {
	ArtifactID string
	BuildID    string
	Timeout    time.Duration
	LastStatus string
}

func (e *ArtifactBuildTimeoutError) Error() string {
	return fmt.Sprintf(
		"timeout waiting for artifact %s build %s after %s (last status: %s)",
		e.ArtifactID,
		e.BuildID,
		e.Timeout,
		e.LastStatus,
	)
}

func artifactBuildPollInterval() time.Duration {
	return durationFromEnv(ArtifactBuildPollIntervalEnvVar, defaultArtifactBuildPollInterval)
}

func artifactBuildPollTimeout() time.Duration {
	return durationFromEnv(ArtifactBuildPollTimeoutEnvVar, defaultArtifactBuildPollTimeout)
}

// IsTerminalArtifactBuildStatus reports whether s is a state from which the build
// will not progress further (COMPLETED, FAILED, CANCELLED).
func IsTerminalArtifactBuildStatus(status string) bool {
	switch status {
	case ArtifactBuildStatusCompleted, ArtifactBuildStatusFailed, ArtifactBuildStatusCancelled:
		return true
	default:
		return false
	}
}

// IsArtifactBuildErrorStatus reports whether s is a terminal failure.
func IsArtifactBuildErrorStatus(status string) bool {
	switch status {
	case ArtifactBuildStatusFailed, ArtifactBuildStatusCancelled:
		return true
	default:
		return false
	}
}

func (s *ServiceImpl) TriggerArtifactBuild(ctx context.Context, artifactID string) (*ArtifactBuildTriggerResponse, error) {
	return Post[ArtifactBuildTriggerResponse](s.client, ctx, "/artifacts/"+artifactID+"/builds/", map[string]any{})
}

func (s *ServiceImpl) GetArtifactBuild(ctx context.Context, artifactID, buildID string) (*ArtifactBuild, error) {
	return Get[ArtifactBuild](s.client, ctx, "/artifacts/"+artifactID+"/builds/"+buildID+"/")
}

func artifactBuildLogsTailLines() int {
	tailLines, err := strconv.Atoi(os.Getenv(ArtifactBuildLogsTailLinesEnvVar))
	if err != nil || tailLines <= 0 {
		return defaultArtifactBuildLogsTailLines
	}
	return tailLines
}

// formatArtifactBuildLogLines formats each line of the build log body independently:
// a line that parses as a structured JSONL entry is rendered as "[timestamp] LEVEL:
// message"; any other line is kept as-is (IBS forwards BuildKit stderr as plain text -
// see workload-api acceptance tests). Falling back per line, rather than treating the
// whole body as JSON the moment any one line parses, means a single JSON line
// elsewhere in the body can't cause a plain-text line - e.g. the "ERROR: failed to
// solve" line this feature exists to surface - to be silently dropped.
func formatArtifactBuildLogLines(body []byte) []string {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry ArtifactBuildLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			lines = append(lines, formatArtifactBuildLogEntry(entry))
			continue
		}
		lines = append(lines, line)
	}

	return lines
}

func tailPlainTextLogLines(text string, n int) string {
	lines := strings.Split(text, "\n")
	if n <= 0 || len(lines) <= n {
		return strings.TrimRight(text, "\n")
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

func formatArtifactBuildLogsBody(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return ""
	}

	return tailPlainTextLogLines(strings.Join(formatArtifactBuildLogLines(body), "\n"), artifactBuildLogsTailLines())
}

func formatArtifactBuildLogEntry(entry ArtifactBuildLogEntry) string {
	level := strings.ToUpper(entry.Levelname)
	if level == "" {
		level = "INFO"
	}
	timestamp := entry.Asctime
	if timestamp == "" {
		timestamp = "unknown"
	}
	return fmt.Sprintf("[%s] %s: %s", timestamp, level, entry.Message)
}

func (s *ServiceImpl) GetArtifactBuildLogs(ctx context.Context, artifactID, buildID string) (string, error) {
	body, err := getRaw(s.client, ctx, "/artifacts/"+artifactID+"/builds/"+buildID+"/logs")
	if err != nil {
		otelLogs, otelErr := s.getOtelEntityLogs(ctx, "artifact", artifactID, artifactBuildLogsTailLines(), buildID)
		if otelErr == nil && otelLogs != "" {
			return otelLogs, nil
		}
		return "", err
	}

	logs := formatArtifactBuildLogsBody(body)
	if logs != "" {
		return logs, nil
	}

	otelLogs, otelErr := s.getOtelEntityLogs(ctx, "artifact", artifactID, artifactBuildLogsTailLines(), buildID)
	if otelErr == nil && otelLogs != "" {
		return otelLogs, nil
	}

	return "", nil
}

func (s *ServiceImpl) WaitForArtifactBuild(
	ctx context.Context,
	artifactID, buildID string,
	opts *WaitForArtifactBuildOptions,
) (*ArtifactBuild, error) {
	pollInterval := artifactBuildPollInterval()
	timeout := artifactBuildPollTimeout()
	if opts != nil {
		if opts.PollInterval > 0 {
			pollInterval = opts.PollInterval
		}
		if opts.Timeout > 0 {
			timeout = opts.Timeout
		}
	}

	deadline := time.Now().Add(timeout)

	var logState *otelLogStreamState
	if opts != nil && opts.OnOtelLogLine != nil {
		logState = newOtelLogStreamState()
	}
	pollOtelLogs := func() {
		if logState == nil {
			return
		}
		s.pollNewOtelEntityLogs(ctx, "artifact", artifactID, buildID, logState, opts.OnOtelLogLine)
	}

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		pollOtelLogs()

		build, err := s.GetArtifactBuild(ctx, artifactID, buildID)
		if err != nil {
			return nil, fmt.Errorf("poll artifact build %s: %w", buildID, err)
		}

		if IsTerminalArtifactBuildStatus(build.Status) {
			pollOtelLogs()
			if IsArtifactBuildErrorStatus(build.Status) {
				return build, &ArtifactBuildFailedError{BuildID: buildID, Status: build.Status}
			}
			return build, nil
		}

		if opts != nil && opts.OnPoll != nil {
			opts.OnPoll(build)
		}

		if time.Now().After(deadline) {
			return build, &ArtifactBuildTimeoutError{
				ArtifactID: artifactID,
				BuildID:    buildID,
				Timeout:    timeout,
				LastStatus: build.Status,
			}
		}

		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
