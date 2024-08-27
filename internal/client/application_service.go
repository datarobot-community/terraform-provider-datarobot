package client

type CreateChatApplicationRequest struct {
	DeploymentID string `json:"deploymentId"`
}

type CreateApplicationFromSourceRequest struct {
	ApplicationSourceVersionID string `json:"applicationSourceVersionId"`
}

type ApplicationResponse struct {
	ID                               string `json:"id"`
	Name                             string `json:"name"`
	Status                           string `json:"status"`
	CustomApplicationSourceID        string `json:"customApplicationSourceId"`
	CustomApplicationSourceVersionID string `json:"customApplicationSourceVersionId"`
	ApplicationUrl                   string `json:"applicationUrl"`
}

type UpdateApplicationRequest struct {
	Name string `json:"name"`
}

type UpdateApplicationSourceRequest struct {
	Name string `json:"name"`
}

type ApplicationSource struct {
	ID            string                   `json:"id"`
	Name          string                   `json:"name"`
	LatestVersion ApplicationSourceVersion `json:"latestVersion"`
}

type CreateApplicationSourceVersionRequest struct {
	Label             string `json:"label"`
	BaseVersion       string `json:"baseVersion,omitempty"`
	BaseEnvironmentID string `json:"baseEnvironmentId,omitempty"`
}

type ListApplicationSourceVersionsResponse struct {
	Data []ApplicationSourceVersion `json:"data"`
}

type ApplicationSourceVersion struct {
	ID                       string               `json:"id"`
	Label                    string               `json:"label"`
	BaseEnvironmentID        string               `json:"baseEnvironmentId,omitempty"`
	BaseEnvironmentVersionID string               `json:"baseEnvironmentVersionId"`
	IsFrozen                 bool                 `json:"isFrozen"`
	RuntimeParameters        []RuntimeParameter   `json:"runtimeParameters,omitempty"`
	Items                    []FileItem           `json:"items,omitempty"`
	Resources                ApplicationResources `json:"resources,omitempty"`
}

type UpdateApplicationSourceVersionRequest struct {
	BaseEnvironmentID string               `json:"baseEnvironmentId,omitempty"`
	Resources         ApplicationResources `json:"resources,omitempty"`
	FilesToDelete     []string             `json:"filesToDelete,omitempty"`
}

type ApplicationResources struct {
	Replicas      int     `json:"replicas,omitempty"`
	ResourceLabel string  `json:"resourceLabel,omitempty"`
	MemoryRequest int     `json:"memoryRequest,omitempty"`
	MemoryLimit   int     `json:"memoryLimit,omitempty"`
	CpuRequest    float64 `json:"cpuRequest,omitempty"`
	CpuLimit      float64 `json:"cpuLimit,omitempty"`
}
