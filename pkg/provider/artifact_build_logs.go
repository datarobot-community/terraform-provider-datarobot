package provider

import (
	"context"
	"fmt"
)

const artifactBuildLogsSeparator = "----------------------------------------"

// artifactBuildLogsURL returns the DataRobot UI link for artifact image build logs.
func artifactBuildLogsURL(baseURL, artifactRepositoryID, artifactID string) string {
	if artifactRepositoryID == "" || artifactID == "" {
		return baseURL + "/registry/service-artifacts"
	}
	return baseURL + "/registry/service-artifacts/" + artifactRepositoryID + "/artifacts/" + artifactID + "/build-log"
}

// artifactBuildErrorMessageWithLogs builds a diagnostic message for a failed
// artifact image build, appending retrieved logs (or the log retrieval error) and a
// link to the full logs in the DataRobot UI.
func artifactBuildErrorMessageWithLogs(baseMessage, logs string, logErr error, logsURL string) string {
	if logErr != nil {
		return fmt.Sprintf(
			"%s (failed to retrieve artifact build logs: %s)\n%s\nSee full logs at: %s",
			baseMessage, logErr, artifactBuildLogsSeparator, logsURL,
		)
	}
	if logs == "" {
		return fmt.Sprintf("%s\n%s\nSee full logs at: %s", baseMessage, artifactBuildLogsSeparator, logsURL)
	}
	return fmt.Sprintf(
		"%s\n%s\nArtifact image build logs:\n%s\n%s\nSee full logs at: %s",
		baseMessage, artifactBuildLogsSeparator, logs, artifactBuildLogsSeparator, logsURL,
	)
}

type artifactBuildEnrichedError struct {
	message string
	cause   error
}

func (e *artifactBuildEnrichedError) Error() string {
	return e.message
}

func (e *artifactBuildEnrichedError) Unwrap() error {
	return e.cause
}

func (r *ArtifactResource) enrichArtifactBuildError(
	ctx context.Context,
	artifactID, artifactRepositoryID, buildID string,
	cause error,
) error {
	if cause == nil {
		return nil
	}

	logsURL := artifactBuildLogsURL(r.provider.service.BaseURL(), artifactRepositoryID, artifactID)

	var logs string
	var logErr error
	if buildID != "" {
		traceAPICall("GetArtifactBuildLogs")
		logs, logErr = r.provider.service.GetArtifactBuildLogs(ctx, artifactID, buildID)
	}

	return &artifactBuildEnrichedError{
		message: artifactBuildErrorMessageWithLogs(cause.Error(), logs, logErr, logsURL),
		cause:   cause,
	}
}
