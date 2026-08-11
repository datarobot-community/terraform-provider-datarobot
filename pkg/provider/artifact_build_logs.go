package provider

import (
	"context"
	"fmt"
	"io"
	"os"
)

const artifactBuildLogsSeparator = "----------------------------------------"
const defaultArtifactBuildLogsTailLines = 30

// artifactBuildLogWriter receives live artifact apply progress and build log lines.
// Terraform hides tflog output unless TF_LOG is set, so apply progress is written
// to stderr by default.
var artifactBuildLogWriter io.Writer = os.Stderr

func emitArtifactBuildLogLine(line string) {
	emitArtifactApplyProgress(line)
}

func emitArtifactApplyProgress(line string) {
	if artifactBuildLogWriter == nil || line == "" {
		return
	}
	_, _ = artifactBuildLogWriter.Write([]byte(line + "\n"))
}

func artifactApplyProgressCreating() {
	emitArtifactApplyProgress("Creating artifact...")
}

func artifactApplyProgressUploading(artifactID string) {
	emitArtifactApplyProgress(fmt.Sprintf("Created artifact with id %s. Uploading code...", artifactID))
}

func artifactApplyProgressBuilding(artifactID string) {
	emitArtifactApplyProgress(fmt.Sprintf("Building artifact with id %s...", artifactID))
}

func artifactApplyProgressBuildPolling(artifactID, buildID string) {
	emitArtifactApplyProgress(fmt.Sprintf("Build %s in progress for %s...", buildID, artifactID))
}

// artifactOtelBuildLogsURL returns the DataRobot public API URL for OTEL build logs
// stored in datavolt (GET /api/v2/otel/artifact/{id}/logs/), scoped to one build.
func artifactOtelBuildLogsURL(baseURL, artifactID, buildID string, limit int) string {
	if artifactID == "" {
		return baseURL + "/registry/service-artifacts"
	}
	if limit <= 0 {
		limit = defaultArtifactBuildLogsTailLines
	}
	query := fmt.Sprintf("limit=%d", limit)
	if buildID != "" {
		query += "&searchKeys=build_id&searchValues=" + buildID
	}
	return baseURL + "/api/v2/otel/artifact/" + artifactID + "/logs/?" + query
}

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

	baseURL := r.provider.service.BaseURL()
	logsURL := artifactBuildLogsURL(baseURL, artifactRepositoryID, artifactID)
	if buildID != "" {
		logsURL = artifactOtelBuildLogsURL(
			baseURL,
			artifactID,
			buildID,
			defaultArtifactBuildLogsTailLines,
		)
	}

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
