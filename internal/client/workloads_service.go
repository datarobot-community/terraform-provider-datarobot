package client

import "context"

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
	CPU     float64 `json:"cpu"`
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
