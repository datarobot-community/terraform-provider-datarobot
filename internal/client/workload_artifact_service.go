package client

// Workload API: Artifacts
//
// OpenAPI source: terraform-provider-datarobot/openapi.yml

type ArtifactStatus string

const (
	ArtifactStatusDraft      ArtifactStatus = "draft"
	ArtifactStatusRegistered ArtifactStatus = "registered"
)

type ArtifactType string

const (
	ArtifactTypeGeneric ArtifactType = "generic"
)

// InputArtifact represents the request body for creating/updating an artifact.
// NOTE: The Workload API currently creates artifacts as drafts.
type InputArtifact struct {
	ArtifactCollectionID *string           `json:"artifactCollectionId,omitempty"`
	Description          string            `json:"description,omitempty"`
	Name                 string            `json:"name"`
	Spec                 ArtifactSpecInput `json:"spec"`
	Status               ArtifactStatus    `json:"status,omitempty"`
	Type                 ArtifactType      `json:"type,omitempty"`
}

// UpdateArtifactRequest represents the PATCH request body.
type UpdateArtifactRequest struct {
	Description *string         `json:"description,omitempty"`
	Name        *string         `json:"name,omitempty"`
	Status      *ArtifactStatus `json:"status,omitempty"`
}

// ArtifactFormatted represents the API response for an artifact.
type ArtifactFormatted struct {
	ArtifactCollectionID *string            `json:"artifactCollectionId"`
	CreatedAt            string             `json:"createdAt"`
	Creator              *Creator           `json:"creator,omitempty"`
	Description          string             `json:"description"`
	ID                   string             `json:"id"`
	Name                 string             `json:"name"`
	Spec                 ArtifactSpecOutput `json:"spec"`
	Status               ArtifactStatus     `json:"status"`
	Type                 ArtifactType       `json:"type"`
	UpdatedAt            string             `json:"updatedAt"`
	Version              int64              `json:"version"`
}

// Creator matches components/schemas/Creator in the Workload API spec.
type Creator struct {
	ID       string  `json:"id"`
	Username *string `json:"username,omitempty"`
	Email    *string `json:"email,omitempty"`
}

type ArtifactSpecInput struct {
	ContainerGroups []ContainerGroupInput `json:"containerGroups,omitempty"`
}

type ArtifactSpecOutput struct {
	ContainerGroups []ContainerGroupOutput `json:"containerGroups,omitempty"`
}

type ContainerGroupInput struct {
	Containers []Container `json:"containers,omitempty"`
}

type ContainerGroupOutput struct {
	Containers []Container `json:"containers,omitempty"`
}

type EnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ProbeConfig struct {
	FailureThreshold    *int64            `json:"failureThreshold,omitempty"`
	Host                *string           `json:"host,omitempty"`
	HTTPHeaders         map[string]string `json:"httpHeaders,omitempty"`
	InitialDelaySeconds *int64            `json:"initialDelaySeconds,omitempty"`
	Path                string            `json:"path"`
	PeriodSeconds       *int64            `json:"periodSeconds,omitempty"`
	Port                *int64            `json:"port,omitempty"`
	Scheme              string            `json:"scheme,omitempty"`
	TimeoutSeconds      *int64            `json:"timeoutSeconds,omitempty"`
}

type ResourceRequest struct {
	CPU     float64 `json:"cpu"`
	GPU     *int64  `json:"gpu,omitempty"`
	GPUType *string `json:"gpuType,omitempty"`
	Memory  int64   `json:"memory"`
}

type Container struct {
	Description     string                `json:"description,omitempty"`
	Entrypoint      []string              `json:"entrypoint,omitempty"`
	EnvironmentVars []EnvironmentVariable `json:"environmentVars,omitempty"`
	ImageURI        string                `json:"imageUri"`
	LivenessProbe   *ProbeConfig          `json:"livenessProbe,omitempty"`
	Name            *string               `json:"name,omitempty"`
	Port            *int64                `json:"port,omitempty"`
	Primary         *bool                 `json:"primary,omitempty"`
	ReadinessProbe  *ProbeConfig          `json:"readinessProbe,omitempty"`
	ResourceRequest ResourceRequest       `json:"resourceRequest"`
	StartupProbe    *ProbeConfig          `json:"startupProbe,omitempty"`
}
