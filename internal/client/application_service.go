package client

type CreateQAApplicationRequest struct {
	DeploymentID string `json:"deploymentId"`
}

type CreateCustomApplicationeRequest struct {
	ApplicationSourceVersionID string `json:"applicationSourceVersionId,omitempty"`
	EnvironmentID              string `json:"environmentId,omitempty"`
}

type Application struct {
	ID                               string   `json:"id"`
	Name                             string   `json:"name"`
	Status                           string   `json:"status"`
	CustomApplicationSourceID        string   `json:"customApplicationSourceId"`
	CustomApplicationSourceVersionID string   `json:"customApplicationSourceVersionId"`
	EnvVersionID                     string   `json:"envVersionId"`
	ApplicationUrl                   string   `json:"applicationUrl"`
	ExternalAccessEnabled            bool     `json:"externalAccessEnabled"`
	ExternalAccessRecipients         []string `json:"externalAccessRecipients"`
}

type UpdateApplicationRequest struct {
	Name                             string   `json:"name,omitempty"`
	CustomApplicationSourceVersionID string   `json:"customApplicationSourceVersionId,omitempty"`
	ExternalAccessEnabled            bool     `json:"externalAccessEnabled"`
	ExternalAccessRecipients         []string `json:"externalAccessRecipients"`
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
	Label                    string                `json:"label"`
	BaseVersion              string                `json:"baseVersion,omitempty"`
	BaseEnvironmentID        string                `json:"baseEnvironmentId,omitempty"`
	BaseEnvironmentVersionID string                `json:"baseEnvironmentVersionId,omitempty"`
	Resources                *ApplicationResources `json:"resources,omitempty"`
	RuntimeParameterValues   string                `json:"runtimeParameterValues,omitempty"`
}

type ApplicationSourceVersion struct {
	ID                       string               `json:"id"`
	Label                    string               `json:"label"`
	BaseEnvironmentID        string               `json:"baseEnvironmentId,omitempty"`
	BaseEnvironmentVersionID string               `json:"baseEnvironmentVersionId,omitempty"`
	IsFrozen                 bool                 `json:"isFrozen"`
	RuntimeParameters        []RuntimeParameter   `json:"runtimeParameters,omitempty"`
	Items                    []FileItem           `json:"items,omitempty"`
	Resources                ApplicationResources `json:"resources,omitempty"`
}

type UpdateApplicationSourceVersionRequest struct {
	BaseEnvironmentID        string                `json:"baseEnvironmentId,omitempty"`
	BaseEnvironmentVersionID string                `json:"baseEnvironmentVersionId,omitempty"`
	Resources                *ApplicationResources `json:"resources,omitempty"`
	FilesToDelete            []string              `json:"filesToDelete,omitempty"`
	RuntimeParameterValues   string                `json:"runtimeParameterValues,omitempty"`
}

type ApplicationResources struct {
	Replicas        int64  `json:"replicas"`
	ResourceLabel   string `json:"resourceLabel"`
	SessionAffinity bool   `json:"sessionAffinity"`
}
