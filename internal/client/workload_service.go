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

