// Artifact image build client methods ported from cli/internal/workload/build.go.

package client

import (
	"context"
	"fmt"
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
	ArtifactBuildPollIntervalEnvVar = "DATAROBOT_ARTIFACT_BUILD_POLL_INTERVAL"
	ArtifactBuildPollTimeoutEnvVar  = "DATAROBOT_ARTIFACT_BUILD_POLL_TIMEOUT"

	defaultArtifactBuildPollInterval = 10 * time.Second
	defaultArtifactBuildPollTimeout  = 10 * time.Minute
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
	PollInterval time.Duration
	Timeout      time.Duration
}

type ArtifactBuildFailedError struct {
	BuildID string
	Status  string
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

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		build, err := s.GetArtifactBuild(ctx, artifactID, buildID)
		if err != nil {
			return nil, fmt.Errorf("poll artifact build %s: %w", buildID, err)
		}

		if IsTerminalArtifactBuildStatus(build.Status) {
			if IsArtifactBuildErrorStatus(build.Status) {
				return build, &ArtifactBuildFailedError{BuildID: buildID, Status: build.Status}
			}
			return build, nil
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
