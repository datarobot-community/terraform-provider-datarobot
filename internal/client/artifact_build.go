// Artifact image build client methods ported from cli/internal/workload/build.go.

package client

import (
	"context"
	"fmt"
	"os"
	"slices"
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
	// FailureReason is the platform's account of a FAILED or CANCELLED build.
	FailureReason string `json:"failureReason,omitempty"`
}

type WaitForArtifactBuildOptions struct {
	PollInterval  time.Duration
	Timeout       time.Duration
	OnOtelLogLine func(OtelLogEntry)
	OnPoll        func(*ArtifactBuild)
}

type ArtifactBuildFailedError struct {
	BuildID       string
	Status        string
	FailureReason string
}

func (e *ArtifactBuildFailedError) Error() string {
	msg := fmt.Sprintf("artifact build %s ended with status %s", e.BuildID, e.Status)
	if e.FailureReason != "" {
		msg += ": " + e.FailureReason
	}
	return msg
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

// How long GetArtifactBuildLogs waits for a build's log to stop growing. Variables so
// tests can shrink them.
var (
	artifactBuildLogsSettleInterval = 10 * time.Second
	artifactBuildLogsSettleTimeout  = 60 * time.Second
)

// GetArtifactBuildLogs returns the tail of one build's log, oldest line first, from the
// artifact's OTEL stream filtered to that build. Log ingestion trails the builder: a
// build already reported FAILED can have no records yet, or only the lines before its
// error. So the tail is read again until two reads artifactBuildLogsSettleInterval apart
// agree, or artifactBuildLogsSettleTimeout passes. No records is not an error: the
// caller then shows only the link to the UI's build log page.
func (s *ServiceImpl) GetArtifactBuildLogs(ctx context.Context, artifactID, buildID string) (string, error) {
	deadline := time.Now().Add(artifactBuildLogsSettleTimeout)
	logs := ""
	for {
		next, err := s.readArtifactBuildLogTail(ctx, artifactID, buildID)
		if err != nil {
			if logs == "" {
				return "", err
			}
			return logs, nil
		}
		if next != "" && next == logs {
			return logs, nil
		}
		logs = next
		if time.Now().Add(artifactBuildLogsSettleInterval).Before(deadline) &&
			sleepCtx(ctx, artifactBuildLogsSettleInterval) == nil {
			continue
		}
		return logs, nil
	}
}

func (s *ServiceImpl) readArtifactBuildLogTail(ctx context.Context, artifactID, buildID string) (string, error) {
	entries, err := s.getOtelEntityLogEntries(ctx, "artifact", artifactID, artifactBuildLogsTailLines(), buildID)
	if err != nil {
		return "", err
	}

	// The server returns the newest records first, but orders them only to the
	// millisecond, so lines logged within one millisecond arrive shuffled (a BuildKit
	// error ahead of the step that raised it). Sort on the full timestamp.
	slices.Reverse(entries)
	slices.SortStableFunc(entries, func(a, b OtelLogEntry) int {
		return strings.Compare(a.Timestamp, b.Timestamp)
	})
	return formatOtelLogEntries(entries), nil
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
		s.pollNewArtifactOtelLogs(ctx, artifactID, buildID, logState, opts.OnOtelLogLine)
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
				return build, &ArtifactBuildFailedError{
					BuildID:       buildID,
					Status:        build.Status,
					FailureReason: build.FailureReason,
				}
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

		if err := sleepCtx(ctx, pollInterval); err != nil {
			return nil, err
		}
	}
}
