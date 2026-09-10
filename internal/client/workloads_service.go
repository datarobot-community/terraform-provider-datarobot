package client

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
)

type WorkloadImportance string
type ProtonStatus string

const (
	WorkloadImportanceCritical WorkloadImportance = "critical"
	WorkloadImportanceHigh     WorkloadImportance = "high"
	WorkloadImportanceModerate WorkloadImportance = "moderate"
	WorkloadImportanceLow      WorkloadImportance = "low"

	ProtonStatusUnknown      ProtonStatus = "unknown"
	ProtonStatusSubmitted    ProtonStatus = "submitted"
	ProtonStatusInitializing ProtonStatus = "initializing"
	ProtonStatusRunning      ProtonStatus = "running"
	ProtonStatusStopping     ProtonStatus = "stopping"
	ProtonStatusStopped      ProtonStatus = "stopped"
	ProtonStatusErrored      ProtonStatus = "errored"
)

type AutoscalingPolicy struct {
	ScalingMetric string  `json:"scalingMetric"`
	Target        float64 `json:"target"`
}

type AutoscalingProperties struct {
	Enabled         *bool               `json:"enabled,omitempty"`
	MinReplicaCount int64               `json:"minReplicaCount"`
	MaxReplicaCount int64               `json:"maxReplicaCount"`
	Policies        []AutoscalingPolicy `json:"policies"`
}

type ResourceAllocation struct {
	CPU       *float64 `json:"cpu,omitempty"`
	GPU       *float64 `json:"gpu,omitempty"`
	GPUMemory *int64   `json:"gpuMemory,omitempty"`
	Memory    *int64   `json:"memory,omitempty"`
}

type ContainerOverride struct {
	Name               string              `json:"name"`
	ResourceAllocation *ResourceAllocation `json:"resourceAllocation,omitempty"`
}

type GroupRuntime struct {
	Name                  string                 `json:"name,omitempty"`
	Containers            []ContainerOverride    `json:"containers,omitempty"`
	Autoscaling           *AutoscalingProperties `json:"autoscaling,omitempty"`
	BundleSelectionPolicy *string                `json:"bundleSelectionPolicy,omitempty"`
	ReplicaCount          *int64                 `json:"replicaCount,omitempty"`
	ResourceBundles       []string               `json:"resourceBundles,omitempty"`
}

type WorkloadRuntime struct {
	ContainerGroups []GroupRuntime `json:"containerGroups,omitempty"`
}

type Workload struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Status      ProtonStatus         `json:"status"`
	Importance  WorkloadImportance   `json:"importance"`
	Type        ArtifactType         `json:"type,omitempty"`
	ArtifactID  *string              `json:"artifactId"`
	Endpoint    *string              `json:"endpoint"`
	Runtime     WorkloadRuntime      `json:"runtime"`
	ProtonID    *string              `json:"protonId"`
	Replacement *WorkloadReplacement `json:"replacement"`
}

type CreateWorkloadRequest struct {
	Name        string             `json:"name"`
	Runtime     WorkloadRuntime    `json:"runtime"`
	ArtifactID  *string            `json:"artifactId,omitempty"`
	Description string             `json:"description,omitempty"`
	Importance  WorkloadImportance `json:"importance,omitempty"`
}

type UpdateWorkloadRequest struct {
	Name        *string             `json:"name,omitempty"`
	Description *string             `json:"description,omitempty"`
	Importance  *WorkloadImportance `json:"importance,omitempty"`
}

func (s *ServiceImpl) CreateWorkload(ctx context.Context, req *CreateWorkloadRequest) (*Workload, error) {
	return Post[Workload](s.client, ctx, "/workloads/", req)
}

func (s *ServiceImpl) GetWorkload(ctx context.Context, id string) (*Workload, error) {
	return Get[Workload](s.client, ctx, "/workloads/"+id+"/")
}

func (s *ServiceImpl) UpdateWorkloadMetadata(ctx context.Context, id string, req *UpdateWorkloadRequest) (*Workload, error) {
	return Patch[Workload](s.client, ctx, "/workloads/"+id+"/", req)
}

func (s *ServiceImpl) DeleteWorkload(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/workloads/"+id+"/")
}

type ReplacementStrategy string
type ReplacementStatus string

const (
	ReplacementStrategyRolling ReplacementStrategy = "rolling"

	ReplacementStatusUnknown      ReplacementStatus = "unknown"
	ReplacementStatusSubmitted    ReplacementStatus = "submitted"
	ReplacementStatusInitializing ReplacementStatus = "initializing"
	ReplacementStatusStaged       ReplacementStatus = "staged"
	ReplacementStatusPromoting    ReplacementStatus = "promoting"
	ReplacementStatusCanceling    ReplacementStatus = "canceling"
	ReplacementStatusFinalizing   ReplacementStatus = "finalizing"
	ReplacementStatusCompleted    ReplacementStatus = "completed"
	ReplacementStatusFailed       ReplacementStatus = "failed"
	ReplacementStatusErrored      ReplacementStatus = "errored"
)

const (
	WorkloadReplacementPollIntervalEnvVar = "DATAROBOT_WORKLOAD_REPLACEMENT_POLL_INTERVAL"
	WorkloadReplacementPollTimeoutEnvVar  = "DATAROBOT_WORKLOAD_REPLACEMENT_POLL_TIMEOUT"

	defaultReplacementPollInterval = 5 * time.Second
	defaultReplacementPollTimeout  = 30 * time.Minute
)

func workloadReplacementPollInterval() time.Duration {
	return durationFromEnv(WorkloadReplacementPollIntervalEnvVar, defaultReplacementPollInterval)
}

func workloadReplacementPollTimeout() time.Duration {
	return durationFromEnv(WorkloadReplacementPollTimeoutEnvVar, defaultReplacementPollTimeout)
}

func durationFromEnv(envVar string, fallback time.Duration) time.Duration {
	raw := os.Getenv(envVar)
	if raw == "" {
		return fallback
	}

	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return fallback
	}

	return d
}

type ReplacementConfig struct {
	WarmupDurationMinutes int64 `json:"warmupDurationMinutes,omitempty"`
	KeepOldVersionMinutes int64 `json:"keepOldVersionMinutes,omitempty"`
}

type StartReplacementRequest struct {
	ArtifactID string              `json:"artifactId"`
	Strategy   ReplacementStrategy `json:"strategy"`
	Config     ReplacementConfig   `json:"config,omitempty"`
	Runtime    *WorkloadRuntime    `json:"runtime,omitempty"`
}

type UpdateWorkloadSettingsRequest struct {
	Runtime WorkloadRuntime `json:"runtime"`
}

type WorkloadReplacement struct {
	ID                  string              `json:"id"`
	WorkloadID          string              `json:"workloadId"`
	CandidateArtifactID string              `json:"candidateArtifactId"`
	Status              ReplacementStatus   `json:"status"`
	Strategy            ReplacementStrategy `json:"strategy"`
	Config              ReplacementConfig   `json:"config,omitempty"`
	Runtime             WorkloadRuntime     `json:"runtime,omitempty"`
	Message             *string             `json:"message,omitempty"`
}

type WaitForWorkloadReplacementOptions struct {
	PollInterval time.Duration
	Timeout      time.Duration
	// ExpectedArtifactID is the artifact the replacement was asked to promote.
	// When set, a replacement that settles while the workload still serves a
	// different artifact is reported as a ReplacementFailedError instead of
	// success. The Workload API drops the replacement record when the new
	// version never becomes ready and keeps the old version serving, which is
	// otherwise indistinguishable from a completed rollout.
	ExpectedArtifactID string
}

type ReplacementFailedError struct {
	Message string
}

func (e *ReplacementFailedError) Error() string {
	return e.Message
}

// IsReplacementFailed reports a status the replacement ended on without
// promoting its candidate. The Workload API's own documentation names only
// "completed" and "failed"; "errored" is what a candidate that never became
// healthy produced on staging. Both count, because a status read as still in
// flight is later seen as a cleared record, which is how a failed rollout gets
// reported as a success.
//
// The comparison folds case: the platform is not consistent about it across
// resources (artifact build statuses come back upper case where these are
// lower), and the cost of being wrong is one-sided.
func IsReplacementFailed(status ReplacementStatus) bool {
	return strings.EqualFold(string(status), string(ReplacementStatusFailed)) ||
		strings.EqualFold(string(status), string(ReplacementStatusErrored))
}

func IsReplacementTerminal(status ReplacementStatus) bool {
	return IsReplacementFailed(status) ||
		strings.EqualFold(string(status), string(ReplacementStatusCompleted))
}

func IsReplacementActive(status ReplacementStatus) bool {
	return !IsReplacementTerminal(status)
}

func (s *ServiceImpl) StartWorkloadReplacement(ctx context.Context, workloadID string, req *StartReplacementRequest) (*WorkloadReplacement, error) {
	return Post[WorkloadReplacement](s.client, ctx, "/workloads/"+workloadID+"/replacement", req)
}

func (s *ServiceImpl) GetWorkloadReplacement(ctx context.Context, workloadID string) (*WorkloadReplacement, error) {
	return Get[WorkloadReplacement](s.client, ctx, "/workloads/"+workloadID+"/replacement")
}

func (s *ServiceImpl) UpdateWorkloadSettings(ctx context.Context, workloadID string, req *UpdateWorkloadSettingsRequest) (*WorkloadReplacement, error) {
	return Patch[WorkloadReplacement](s.client, ctx, "/workloads/"+workloadID+"/settings", req)
}

// WaitForWorkloadReplacement polls workload.replacement (via GetWorkload) until
// the in-flight replacement settles. It avoids the /replacement endpoint because
// a "completed" record is deleted within ~1s (so /replacement 404s and races the
// poll) while a failed record persists. That asymmetry makes the workload
// record unambiguous: a failure status => failure; nil-while-running => settled
// (the proton switch lands before nil appears).
// A nil is only "done" after an active replacement was seen (seenActive) —
// otherwise it's the brief gap before the API creates the record, so keep polling.
//
// "Settled" is not the same as "promoted". A rollout whose new version never
// passes readiness is abandoned: the platform stops the new replica, keeps the
// old one serving and clears the record, so the workload looks exactly like a
// completed rollout. opts.ExpectedArtifactID tells the two apart.
func (s *ServiceImpl) WaitForWorkloadReplacement(
	ctx context.Context,
	workloadID string,
	opts *WaitForWorkloadReplacementOptions,
) (*WorkloadReplacement, error) {
	pollInterval := workloadReplacementPollInterval()
	timeout := workloadReplacementPollTimeout()
	if opts != nil {
		if opts.PollInterval > 0 {
			pollInterval = opts.PollInterval
		}
		if opts.Timeout > 0 {
			timeout = opts.Timeout
		}
	}
	expectedArtifactID := ""
	if opts != nil {
		expectedArtifactID = opts.ExpectedArtifactID
	}

	deadline := time.Now().Add(timeout)
	seenActive := false
	var lastReplacement *WorkloadReplacement

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		workload, err := s.GetWorkload(ctx, workloadID)
		if err != nil {
			return lastReplacement, err
		}
		replacement := workload.Replacement

		switch {
		case replacement != nil && IsReplacementFailed(replacement.Status):
			return replacement, &ReplacementFailedError{Message: replacementFailureMessage(replacement)}

		case replacement != nil:
			lastReplacement = replacement
			if IsReplacementTerminal(replacement.Status) {
				// "completed" is rarely observable (cleaned up within ~1s), but
				// accept it when caught.
				return replacement, s.settledReplacementError(ctx, workloadID, workload, expectedArtifactID)
			}
			seenActive = true

		default: // replacement == nil
			if seenActive && workload.Status == ProtonStatusRunning {
				return lastReplacement, s.settledReplacementError(ctx, workloadID, workload, expectedArtifactID)
			}
		}

		if time.Now().After(deadline) {
			return lastReplacement, fmt.Errorf(
				"timeout waiting for workload %s replacement after %s (workload status: %s)",
				workloadID,
				timeout,
				workload.Status,
			)
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

// WorkloadEvent is one entry of the workload's activity log
// (GET /workloads/{id}/events/, newest first). Replacements are the only thing
// recorded there so far, as "Replacement Completed" and "Replacement Errored",
// and the errored one is the platform's only account of a rollout it abandoned:
// the replacement record itself is deleted, so by the time apply can ask, this
// feed is what still remembers why.
type WorkloadEvent struct {
	ID        string               `json:"id"`
	Timestamp time.Time            `json:"timestamp"`
	EventType string               `json:"eventType"`
	Details   WorkloadEventDetails `json:"details"`
}

type WorkloadEventDetails struct {
	ReplacementID string `json:"replacementId"`
	ArtifactID    string `json:"artifactId"`
	Message       string `json:"message"`
	// ProtonStatuses is keyed by candidate proton ID and mirrors Kubernetes pod
	// state. Only the fields worth quoting back are decoded; anything missing
	// leaves the event's own Message to speak for itself.
	ProtonStatuses map[string]WorkloadEventProtonStatus `json:"protonStatuses"`
}

type WorkloadEventProtonStatus struct {
	Replicas []WorkloadEventReplica `json:"replicas"`
}

type WorkloadEventReplica struct {
	Containers []WorkloadEventContainer `json:"containers"`
}

type WorkloadEventContainer struct {
	Name         string                       `json:"name"`
	Ready        bool                         `json:"ready"`
	RestartCount int64                        `json:"restartCount"`
	Reason       string                       `json:"reason"`
	LastState    *WorkloadEventContainerState `json:"lastState"`
}

type WorkloadEventContainerState struct {
	Reason   string `json:"reason"`
	ExitCode *int64 `json:"exitCode"`
}

// workloadEventPageSize is how far back a failure explanation is looked for.
// The event that matters was written seconds ago and the feed is newest first,
// so this only has to outrun the replacements a busy workload logged in between.
const workloadEventPageSize = 20

func (s *ServiceImpl) listWorkloadEvents(ctx context.Context, workloadID string) ([]WorkloadEvent, error) {
	result, err := Get[PaginatedResponse[WorkloadEvent]](
		s.client, ctx,
		fmt.Sprintf("/workloads/%s/events/?limit=%d", workloadID, workloadEventPageSize),
	)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

// settledReplacementError is the verdict once a replacement has settled: nil
// when the workload is on the artifact it was asked to promote, otherwise the
// failure with whatever the platform's event feed says about it appended. The
// feed read is best effort — it only adds detail to an error already decided,
// so its own failure must not replace the diagnosis with a fetch error.
func (s *ServiceImpl) settledReplacementError(
	ctx context.Context,
	workloadID string,
	workload *Workload,
	expectedArtifactID string,
) error {
	failed := replacementLandedOnArtifact(workload, expectedArtifactID)
	if failed == nil {
		return nil
	}

	if events, err := s.listWorkloadEvents(ctx, workloadID); err == nil {
		if detail := replacementFailureFromEvents(events, expectedArtifactID); detail != "" {
			failed.Message += "\n" + detail
			return failed
		}
	}

	// Nothing in the feed to quote, so say where to look instead.
	failed.Message += "\nCheck that version's container logs for the startup failure."
	return failed
}

// replacementFailureFromEvents is the platform's own account of the rollout,
// taken from the newest event that reports a failure for the artifact apply
// asked to promote. Only event types that name a failure are quoted, so a
// "Replacement Completed" message can never be attached to a failure, and an
// event whose artifact does not match is skipped so an older rollout's reason
// is not passed off as this one's.
func replacementFailureFromEvents(events []WorkloadEvent, expectedArtifactID string) string {
	for _, event := range events {
		if !isWorkloadFailureEvent(event.EventType) {
			continue
		}
		if event.Details.ArtifactID != "" && expectedArtifactID != "" &&
			event.Details.ArtifactID != expectedArtifactID {
			continue
		}

		parts := make([]string, 0, 2)
		if event.Details.Message != "" {
			parts = append(parts, event.Details.Message)
		}
		if container := describeUnreadyContainer(event.Details.ProtonStatuses); container != "" {
			parts = append(parts, container)
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

func isWorkloadFailureEvent(eventType string) bool {
	lower := strings.ToLower(eventType)
	return strings.Contains(lower, "error") || strings.Contains(lower, "fail")
}

// describeUnreadyContainer names the first container that kept the candidate
// from becoming ready, which is the line that turns "read the logs" into a
// cause. Kubernetes reports the crash on the previous run, so the exit code
// comes from lastState rather than the current waiting one.
func describeUnreadyContainer(protonStatuses map[string]WorkloadEventProtonStatus) string {
	for _, proton := range protonStatuses {
		for _, replica := range proton.Replicas {
			for _, container := range replica.Containers {
				if container.Ready {
					continue
				}
				detail := fmt.Sprintf("Container %s is not ready", container.Name)
				if container.Reason != "" {
					detail += ": " + container.Reason
				}
				if container.LastState != nil && container.LastState.ExitCode != nil {
					detail += fmt.Sprintf(" (last run exited with code %d", *container.LastState.ExitCode)
					if container.RestartCount == 1 {
						detail += ", 1 restart"
					} else if container.RestartCount > 1 {
						detail += fmt.Sprintf(", %d restarts", container.RestartCount)
					}
					detail += ")"
				}
				return detail + "."
			}
		}
	}
	return ""
}

// replacementFailureMessage names the status the rollout ended on and carries
// the platform's own account of it when there is one. Without the status the
// reader of a bare message cannot tell a failed rollout from a failed request.
func replacementFailureMessage(replacement *WorkloadReplacement) string {
	message := fmt.Sprintf("workload replacement ended with status %q", replacement.Status)
	if replacement.Message != nil && *replacement.Message != "" {
		message += ": " + *replacement.Message
	}
	return message + ". The workload keeps serving its previous artifact"
}

// replacementLandedOnArtifact reports a replacement that settled without the
// workload switching to the artifact it was asked to promote. An empty expected
// ID skips the check.
func replacementLandedOnArtifact(workload *Workload, expectedArtifactID string) *ReplacementFailedError {
	served := ""
	if workload.ArtifactID != nil {
		served = *workload.ArtifactID
	}
	if expectedArtifactID == "" || served == expectedArtifactID {
		return nil
	}
	return &ReplacementFailedError{Message: fmt.Sprintf(
		"workload replacement finished without switching to artifact %s: the workload still serves artifact %s. "+
			"The platform abandons a rollout whose new version never becomes ready.",
		expectedArtifactID, served,
	)}
}

type ArtifactStatus string
type ArtifactType string

const (
	ArtifactStatusDraft  ArtifactStatus = "draft"
	ArtifactStatusLocked ArtifactStatus = "locked"

	ArtifactTypeService ArtifactType = "service"
	ArtifactTypeNim     ArtifactType = "nim"
	ArtifactTypeAgent   ArtifactType = "agent"
	ArtifactTypeMCP     ArtifactType = "mcp"
)

const (
	EnvironmentVariableSourceString     = "string"
	EnvironmentVariableSourceCredential = "dr-credential"
	EnvironmentVariableSourceAPIKey     = "api-key"
)

const (
	RouteAuthRequired = "required"
	RouteAuthOptional = "optional"
	RouteAuthDisabled = "disabled"
)

const (
	// RoutePathMaxLength mirrors workload_api.schemas.containers.WorkloadRoute.path max_length.
	RoutePathMaxLength = 1024
	// ArtifactContainerMaxRoutes mirrors workload_api.schemas.containers.Container.routes max_length.
	ArtifactContainerMaxRoutes = 50
)

// ArtifactContainerRoute is a workload route exposed publicly from a primary
// container, e.g. an MCP server's OAuth discovery document. Mirrors
// workload_api.schemas.containers.WorkloadRoute.
type ArtifactContainerRoute struct {
	Path string `json:"path"`
	Auth string `json:"auth"`
}

type ArtifactEnvironmentVariable struct {
	Source string `json:"source,omitempty"`
	// Name is optional for the api-key source (the platform resolves an
	// omitted name to DATAROBOT_API_TOKEN and stores it as absent).
	Name           string `json:"name,omitempty"`
	Value          string `json:"value,omitempty"`
	DrCredentialID string `json:"drCredentialId,omitempty"`
	Key            string `json:"key,omitempty"`
}

type ArtifactProbeConfig struct {
	Path                string            `json:"path"`
	Port                *int64            `json:"port,omitempty"`
	Scheme              *string           `json:"scheme,omitempty"`
	Host                *string           `json:"host,omitempty"`
	HTTPHeaders         map[string]string `json:"httpHeaders,omitempty"`
	InitialDelaySeconds *int64            `json:"initialDelaySeconds,omitempty"`
	PeriodSeconds       *int64            `json:"periodSeconds,omitempty"`
	TimeoutSeconds      *int64            `json:"timeoutSeconds,omitempty"`
	FailureThreshold    *int64            `json:"failureThreshold,omitempty"`
	SuccessThreshold    *int64            `json:"successThreshold,omitempty"`
}

type ArtifactCapabilities struct {
	Add  []string `json:"add,omitempty"`
	Drop []string `json:"drop,omitempty"`
}

type ArtifactSeccompProfile struct {
	Type             string  `json:"type"`
	LocalhostProfile *string `json:"localhostProfile,omitempty"`
}

type ArtifactSecurityContext struct {
	AllowPrivilegeEscalation *bool                   `json:"allowPrivilegeEscalation,omitempty"`
	Capabilities             *ArtifactCapabilities   `json:"capabilities,omitempty"`
	ReadOnlyRootFilesystem   *bool                   `json:"readOnlyRootFilesystem,omitempty"`
	SeccompProfile           *ArtifactSeccompProfile `json:"seccompProfile,omitempty"`
}

type ArtifactDataRobotCodeRef struct {
	CatalogID        string `json:"catalogId"`
	CatalogVersionID string `json:"catalogVersionId"`
}

type ArtifactCodeRef struct {
	Provider  string                   `json:"provider,omitempty"`
	Type      string                   `json:"type,omitempty"`
	DataRobot ArtifactDataRobotCodeRef `json:"datarobot"`
}

type ArtifactDockerfileConfig struct {
	Source                        string   `json:"source,omitempty"`
	Path                          string   `json:"path,omitempty"`
	Entrypoint                    []string `json:"entrypoint,omitempty"`
	ExecutionEnvironmentID        string   `json:"executionEnvironmentId,omitempty"`
	ExecutionEnvironmentVersionID string   `json:"executionEnvironmentVersionId,omitempty"`
}

type ArtifactImageBuildConfig struct {
	CodeRef    *ArtifactCodeRef          `json:"codeRef,omitempty"`
	Dockerfile *ArtifactDockerfileConfig `json:"dockerfile,omitempty"`
}

type ArtifactContainerBuildInfo struct {
	ArtifactImageBuildID string `json:"artifactImageBuildId"`
	Status               string `json:"status"`
	CreatedAt            string `json:"createdAt"`
}

type ArtifactContainer struct {
	Name             *string                       `json:"name,omitempty"`
	ImageURI         string                        `json:"imageUri,omitempty"`
	Primary          *bool                         `json:"primary,omitempty"`
	Description      string                        `json:"description,omitempty"`
	Port             *int64                        `json:"port,omitempty"`
	Entrypoint       []string                      `json:"entrypoint,omitempty"`
	Routes           []ArtifactContainerRoute      `json:"routes,omitempty"`
	EnvironmentVars  []ArtifactEnvironmentVariable `json:"environmentVars,omitempty"`
	StartupProbe     *ArtifactProbeConfig          `json:"startupProbe,omitempty"`
	ReadinessProbe   *ArtifactProbeConfig          `json:"readinessProbe,omitempty"`
	LivenessProbe    *ArtifactProbeConfig          `json:"livenessProbe,omitempty"`
	ImageBuildConfig *ArtifactImageBuildConfig     `json:"imageBuildConfig,omitempty"`
	Build            *ArtifactContainerBuildInfo   `json:"build,omitempty"`
	SecurityContext  *ArtifactSecurityContext      `json:"securityContext,omitempty"`
}

type ArtifactContainerGroup struct {
	Name       string              `json:"name,omitempty"`
	Containers []ArtifactContainer `json:"containers"`
}

type ArtifactNimStorageConfig struct {
	Mode    string  `json:"mode,omitempty"`
	PvcSize *string `json:"pvcSize,omitempty"`
}

type ArtifactSpec struct {
	Type            string                    `json:"type,omitempty"`
	ContainerGroups []ArtifactContainerGroup  `json:"containerGroups"`
	Storage         *ArtifactNimStorageConfig `json:"storage,omitempty"`
	TemplateID      *string                   `json:"templateId,omitempty"`
	A2AEnabled      *bool                     `json:"a2aEnabled,omitempty"`
}

type ArtifactUser struct {
	ID       string  `json:"id"`
	FullName *string `json:"fullName,omitempty"`
	Email    *string `json:"email,omitempty"`
	Username *string `json:"username,omitempty"`
	Userhash *string `json:"userhash,omitempty"`
}

type ArtifactTag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Artifact struct {
	ID                   string         `json:"id"`
	Name                 string         `json:"name"`
	Description          string         `json:"description"`
	Type                 ArtifactType   `json:"type"`
	Status               ArtifactStatus `json:"status"`
	Version              *int           `json:"version"`
	Spec                 ArtifactSpec   `json:"spec"`
	ArtifactRepositoryID *string        `json:"artifactRepositoryId"`
	CreatedAt            string         `json:"createdAt"`
	UpdatedAt            string         `json:"updatedAt"`
	Creator              *ArtifactUser  `json:"creator,omitempty"`
	Tags                 []ArtifactTag  `json:"tags,omitempty"`
	Permissions          []string       `json:"permissions,omitempty"`
}

type CreateArtifactRequest struct {
	Name                 string         `json:"name"`
	Description          string         `json:"description,omitempty"`
	Type                 ArtifactType   `json:"type,omitempty"`
	Status               ArtifactStatus `json:"status,omitempty"`
	Spec                 ArtifactSpec   `json:"spec"`
	ArtifactRepositoryID *string        `json:"artifactRepositoryId,omitempty"`
}

type PatchArtifactRequest struct {
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	Status      *ArtifactStatus `json:"status,omitempty"`
	Spec        *ArtifactSpec   `json:"spec,omitempty"`
}

func (s *ServiceImpl) CreateArtifact(ctx context.Context, req *CreateArtifactRequest) (*Artifact, error) {
	return Post[Artifact](s.client, ctx, "/artifacts/", req)
}

func (s *ServiceImpl) PatchArtifact(ctx context.Context, id string, req *PatchArtifactRequest) (*Artifact, error) {
	return Patch[Artifact](s.client, ctx, "/artifacts/"+id+"/", req)
}

func (s *ServiceImpl) GetArtifact(ctx context.Context, id string) (*Artifact, error) {
	return Get[Artifact](s.client, ctx, "/artifacts/"+id+"/")
}

type ListArtifactsRequest struct {
	Status string `url:"status,omitempty"`
	Limit  int    `url:"limit,omitempty"`
}

func (s *ServiceImpl) ListArtifacts(ctx context.Context, req *ListArtifactsRequest) ([]Artifact, error) {
	const defaultPageSize = 100

	maxResults := 0
	pageSize := defaultPageSize
	status := ""

	if req != nil {
		status = req.Status
		if req.Limit > 0 {
			maxResults = req.Limit
			pageSize = req.Limit
		}
	}

	queryReq := &ListArtifactsRequest{
		Status: status,
		Limit:  pageSize,
	}
	pathValues, _ := query.Values(queryReq)
	nextURL := "/artifacts/?" + pathValues.Encode()

	var results []Artifact
	for nextURL != "" {
		result, err := Get[PaginatedResponse[Artifact]](s.client, ctx, nextURL)
		if err != nil {
			return nil, err
		}

		results = append(results, result.Data...)

		if maxResults > 0 && len(results) >= maxResults {
			return results[:maxResults], nil
		}

		nextURL = result.Next
		if nextURL != "" {
			if strings.Contains(nextURL, "?") {
				query := strings.Split(nextURL, "?")[1]
				nextURL = "/artifacts/?" + query
			} else {
				nextURL = "/artifacts/"
			}
		}
	}

	return results, nil
}

func (s *ServiceImpl) DeleteArtifactRepository(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/artifactRepositories/"+id+"/")
}
