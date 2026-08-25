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
}

type ReplacementFailedError struct {
	Message string
}

func (e *ReplacementFailedError) Error() string {
	return e.Message
}

func IsReplacementTerminal(status ReplacementStatus) bool {
	return status == ReplacementStatusCompleted || status == ReplacementStatusErrored
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
// poll) while an "errored" record persists. That asymmetry makes the workload
// record unambiguous: errored => failure; nil-while-running => completed (nil
// can't be a masked failure, and the proton switch lands before nil appears).
// A nil is only "done" after an active replacement was seen (seenActive) —
// otherwise it's the brief gap before the API creates the record, so keep polling.
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
		case replacement != nil && replacement.Status == ReplacementStatusErrored:
			message := "workload replacement failed"
			if replacement.Message != nil && *replacement.Message != "" {
				message = *replacement.Message
			}
			return replacement, &ReplacementFailedError{Message: message}

		case replacement != nil:
			lastReplacement = replacement
			if replacement.Status == ReplacementStatusCompleted {
				// Rarely observable (cleaned up within ~1s), but accept it when caught.
				return replacement, nil
			}
			seenActive = true

		default: // replacement == nil
			if seenActive && workload.Status == ProtonStatusRunning {
				return lastReplacement, nil
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

type ArtifactStatus string
type ArtifactType string

const (
	ArtifactStatusDraft  ArtifactStatus = "draft"
	ArtifactStatusLocked ArtifactStatus = "locked"

	ArtifactTypeService ArtifactType = "service"
	ArtifactTypeNim     ArtifactType = "nim"
	ArtifactTypeAgent   ArtifactType = "agent"
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
