package client

// Workload API: Workloads
//
// OpenAPI source: terraform-provider-datarobot/openapi.yml

type WorkloadStatus string

const (
	WorkloadStatusUnknown      WorkloadStatus = "unknown"
	WorkloadStatusSubmitted    WorkloadStatus = "submitted"
	WorkloadStatusInitializing WorkloadStatus = "initializing"
	WorkloadStatusRunning      WorkloadStatus = "running"
	WorkloadStatusStopping     WorkloadStatus = "stopping"
	WorkloadStatusStopped      WorkloadStatus = "stopped"
	WorkloadStatusErrored      WorkloadStatus = "errored"
)

type ScalingMetricType string

const (
	ScalingMetricTypeCPUAverageUtilization  ScalingMetricType = "cpuAverageUtilization"
	ScalingMetricTypeHTTPRequestsConcurrency ScalingMetricType = "httpRequestsConcurrency"
)

type AutoscalingPolicy struct {
	ScalingMetric ScalingMetricType `json:"scalingMetric"`
	Target        float64           `json:"target"`
	MinCount      int64             `json:"minCount"`
	MaxCount      int64             `json:"maxCount"`
	Priority      *int64            `json:"priority,omitempty"`
}

type AutoscalingPropertiesInput struct {
	Enabled  bool              `json:"enabled,omitempty"`
	Policies []AutoscalingPolicy `json:"policies"`
}

type AutoscalingPropertiesOutput struct {
	Enabled  bool              `json:"enabled,omitempty"`
	Policies []AutoscalingPolicy `json:"policies"`
}

// ResourceBundleResources is currently the only supported resource type in the schema.
type ResourceBundleResources struct {
	ResourceBundleID string `json:"resourceBundleId"`
	Type             string `json:"type,omitempty"` // const: resource_bundle
}

type ResourceBundleResourcesOut struct {
	ResourceBundleID string  `json:"resourceBundleId"`
	Type             string  `json:"type,omitempty"` // const: resource_bundle
	GPUMaker         *string `json:"gpuMaker,omitempty"`
	GPUTypeLabel     *string `json:"gpuTypeLabel,omitempty"`
}

type WorkloadRuntime struct {
	Autoscaling   *AutoscalingPropertiesInput `json:"autoscaling,omitempty"`
	ReplicaCount  *int64                      `json:"replicaCount,omitempty"`
	Resources     []ResourceBundleResources    `json:"resources,omitempty"`
}

type WorkloadRuntimeFormatted struct {
	Autoscaling  *AutoscalingPropertiesOutput `json:"autoscaling,omitempty"`
	ReplicaCount *int64                       `json:"replicaCount,omitempty"`
	Resources    []ResourceBundleResourcesOut  `json:"resources"`
}

type WorkloadCondition struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type WorkloadStatusDetails struct {
	Conditions []WorkloadCondition `json:"conditions,omitempty"`
	LogTail    []string            `json:"logTail,omitempty"`
}

type WorkloadFormatted struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	CreatedAt     string                 `json:"createdAt"`
	UpdatedAt     string                 `json:"updatedAt"`
	Status        WorkloadStatus          `json:"status"`
	ArtifactID    string                 `json:"artifactId"`
	InternalURL   string                 `json:"internalUrl"`
	Endpoint      *string                `json:"endpoint,omitempty"`
	Runtime       WorkloadRuntimeFormatted `json:"runtime"`
	Creator       *Creator               `json:"creator,omitempty"`
	RunningSince  *string                `json:"runningSince,omitempty"`
	StatusDetails *WorkloadStatusDetails  `json:"statusDetails,omitempty"`
}

type CreateWorkloadRequest struct {
	Artifact   *InputArtifact  `json:"artifact,omitempty"`
	ArtifactID *string         `json:"artifactId,omitempty"`
	Name       *string         `json:"name,omitempty"`
	Runtime    WorkloadRuntime `json:"runtime"`
}

