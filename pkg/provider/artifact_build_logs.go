package provider

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

const artifactBuildLogsSeparator = "----------------------------------------"
const defaultArtifactBuildLogsTailLines = 30

// artifactBuildLogWriter receives live artifact apply progress and build log
// lines during apply. Written to the provider's stderr. Terraform runs the
// provider as a go-plugin child process and routes that stderr through its own
// logging pipeline, so these lines surface only when TF_LOG is set (e.g.
// TF_LOG=DEBUG) - not on a plain `terraform apply`, regardless of writing here
// vs. through tflog. The build-failure error message includes a tailed log
// excerpt and a logs link unconditionally, so a failure is diagnosable either
// way.
var artifactBuildLogWriter io.Writer = os.Stderr

// artifactBuildLogLinePrefix labels a build log line with the build it came from.
// Terraform runs resources in parallel, so two artifact builds in one apply stream
// through this same writer; without the label their lines are indistinguishable even
// though each stream is already filtered to its own build_id.
func artifactBuildLogLinePrefix(buildID string) string {
	if buildID == "" {
		return ""
	}
	return "[build " + buildID + "] "
}

// emitArtifactBuildLogLine writes one build log record, labelled with its build. A
// record may span several lines (an OTEL entry carrying a stack trace), so the label
// is repeated on continuation lines and the whole record goes out in a single write -
// concurrent builds share the writer, and one write per record keeps another build
// from slipping a line into the middle of this one.
func emitArtifactBuildLogLine(buildID, line string) {
	if line == "" {
		return
	}
	prefix := artifactBuildLogLinePrefix(buildID)
	emitArtifactApplyProgress(prefix + strings.ReplaceAll(line, "\n", "\n"+prefix))
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
