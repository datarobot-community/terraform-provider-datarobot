package client

type CreateCustomJobRequest struct {
	Name                   string             `json:"name"`
	Description            *string            `json:"description,omitempty"`
	JobType                string             `json:"jobType,omitempty"`
	EnvironmentID          *string            `json:"environmentId,omitempty"`
	EnvironmentVersionID   *string            `json:"environmentVersionId,omitempty"`
	RuntimeParameterValues string             `json:"runtimeParameterValues,omitempty"`
	Resources              CustomJobResources `json:"resources"`
}

type CustomJobResources struct {
	EgressNetworkPolicy string  `json:"egressNetworkPolicy"`
	ResourceBundleID    *string `json:"resourceBundleId,omitempty"`
}

type CustomJob struct {
	ID                   string             `json:"id"`
	Name                 string             `json:"name"`
	Description          string             `json:"description"`
	JobType              string             `json:"jobType"`
	EnvironmentID        string             `json:"environmentId"`
	EnvironmentVersionID string             `json:"environmentVersionId"`
	Items                []FileItem         `json:"items"`
	RuntimeParameters    []RuntimeParameter `json:"runtimeParameters"`
	Resources            CustomJobResources `json:"resources"`
}

type UpdateCustomJobRequest struct {
	Name                   string              `json:"name"`
	Description            *string             `json:"description,omitempty"`
	EnvironmentID          *string             `json:"environmentId,omitempty"`
	EnvironmentVersionID   *string             `json:"environmentVersionId,omitempty"`
	RuntimeParameterValues string              `json:"runtimeParameterValues,omitempty"`
	Resources              *CustomJobResources `json:"resources,omitempty"`
}

type HostedCustomMetricTemplateRequest struct {
	Directionality  string `json:"directionality"`
	Type            string `json:"type"`
	Units           string `json:"units"`
	TimeStep        string `json:"timeStep"`
	IsModelSpecific bool   `json:"isModelSpecific"`
}

type HostedCustomMetricTemplate struct {
	ID              string `json:"id"`
	Directionality  string `json:"directionality"`
	Type            string `json:"type"`
	Units           string `json:"units"`
	TimeStep        string `json:"timeStep"`
	IsModelSpecific bool   `json:"isModelSpecific"`
}

type CustomJobMetric struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
