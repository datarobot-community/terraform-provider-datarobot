package client

import "context"

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
	MinCount      int64   `json:"minCount"`
	MaxCount      int64   `json:"maxCount"`
	Priority      *int64  `json:"priority,omitempty"`
}

type AutoscalingProperties struct {
	Enabled  *bool               `json:"enabled,omitempty"`
	Policies []AutoscalingPolicy `json:"policies"`
}

type ResourceBundleResources struct {
	Type             string `json:"type"`
	ResourceBundleID string `json:"resourceBundleId"`
}

type ProtonRuntime struct {
	ReplicaCount *int64                    `json:"replicaCount,omitempty"`
	Autoscaling  *AutoscalingProperties    `json:"autoscaling,omitempty"`
	Resources    []ResourceBundleResources `json:"resources,omitempty"`
}

type Workload struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Status      ProtonStatus       `json:"status"`
	Importance  WorkloadImportance `json:"importance"`
	ArtifactID  *string            `json:"artifactId"`
	Endpoint    *string            `json:"endpoint"`
	Runtime     ProtonRuntime      `json:"runtime"`
}

type CreateWorkloadRequest struct {
	Name        string             `json:"name"`
	Runtime     ProtonRuntime      `json:"runtime"`
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

func (s *ServiceImpl) UpdateWorkload(ctx context.Context, id string, req *UpdateWorkloadRequest) (*Workload, error) {
	return Patch[Workload](s.client, ctx, "/workloads/"+id+"/", req)
}

func (s *ServiceImpl) DeleteWorkload(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/workloads/"+id+"/")
}

type ArtifactStatus string
type ArtifactType string

const (
	ArtifactStatusDraft  ArtifactStatus = "draft"
	ArtifactStatusLocked ArtifactStatus = "locked"

	ArtifactTypeService ArtifactType = "service"
	ArtifactTypeNim     ArtifactType = "nim"
)

type ArtifactEnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
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
}

type ArtifactResourceRequest struct {
	CPU     int64   `json:"cpu"`
	Memory  int64   `json:"memory"`
	GPU     *int64  `json:"gpu,omitempty"`
	GPUType *string `json:"gpuType,omitempty"`
}

type ArtifactContainer struct {
	Name            *string                       `json:"name,omitempty"`
	ImageURI        string                        `json:"imageUri"`
	Primary         *bool                         `json:"primary,omitempty"`
	Description     string                        `json:"description,omitempty"`
	Port            *int64                        `json:"port,omitempty"`
	Entrypoint      []string                      `json:"entrypoint,omitempty"`
	EnvironmentVars []ArtifactEnvironmentVariable `json:"environmentVars,omitempty"`
	ResourceRequest ArtifactResourceRequest       `json:"resourceRequest"`
	StartupProbe    *ArtifactProbeConfig          `json:"startupProbe,omitempty"`
	ReadinessProbe  *ArtifactProbeConfig          `json:"readinessProbe,omitempty"`
	LivenessProbe   *ArtifactProbeConfig          `json:"livenessProbe,omitempty"`
}

type ArtifactContainerGroup struct {
	Containers []ArtifactContainer `json:"containers"`
}

type ArtifactSpec struct {
	Type            string                   `json:"type,omitempty"`
	ContainerGroups []ArtifactContainerGroup `json:"containerGroups"`
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
}

type CreateArtifactRequest struct {
	Name                 string         `json:"name"`
	Description          string         `json:"description,omitempty"`
	Type                 ArtifactType   `json:"type,omitempty"`
	Status               ArtifactStatus `json:"status,omitempty"`
	Spec                 ArtifactSpec   `json:"spec"`
	ArtifactRepositoryID *string        `json:"artifactRepositoryId,omitempty"`
}

func (s *ServiceImpl) CreateArtifact(ctx context.Context, req *CreateArtifactRequest) (*Artifact, error) {
	return Post[Artifact](s.client, ctx, "/artifacts/", req)
}

func (s *ServiceImpl) GetArtifact(ctx context.Context, id string) (*Artifact, error) {
	return Get[Artifact](s.client, ctx, "/artifacts/"+id+"/")
}
