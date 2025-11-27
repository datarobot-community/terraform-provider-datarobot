package client

type CreateQAApplicationRequest struct {
	DeploymentID string `json:"deploymentId"`
}

type CreateCustomApplicationRequest struct {
	ApplicationSourceVersionID string                `json:"applicationSourceVersionId,omitempty"`
	EnvironmentID              string                `json:"environmentId,omitempty"`
	Resources                  *ApplicationResources `json:"resources,omitempty"`
}

type Application struct {
	ID                               string                `json:"id"`
	Name                             string                `json:"name"`
	Status                           string                `json:"status"`
	CustomApplicationSourceID        string                `json:"customApplicationSourceId"`
	CustomApplicationSourceVersionID string                `json:"customApplicationSourceVersionId"`
	EnvVersionID                     string                `json:"envVersionId"`
	ApplicationUrl                   string                `json:"applicationUrl"`
	ExternalAccessEnabled            bool                  `json:"externalAccessEnabled"`
	ExternalAccessRecipients         []string              `json:"externalAccessRecipients"`
	AllowAutoStopping                bool                  `json:"allowAutoStopping"`
	Resources                        *ApplicationResources `json:"resources,omitempty"`
	RequiredKeyScopeLevel            *string               `json:"requiredKeyScopeLevel,omitempty"`
}

type UpdateApplicationRequest struct {
	Name                             string   `json:"name,omitempty"`
	CustomApplicationSourceVersionID string   `json:"customApplicationSourceVersionId,omitempty"`
	ExternalAccessEnabled            bool     `json:"externalAccessEnabled"`
	ExternalAccessRecipients         []string `json:"externalAccessRecipients"`
	AllowAutoStopping                bool     `json:"allowAutoStopping"`
	RequiredKeyScopeLevel            *string  `json:"requiredKeyScopeLevel,omitempty"`
}

type CreateApplicationSourceFromTemplateRequest struct {
	CustomTemplateID string `json:"customTemplateId"`
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
	Replicas                     *int64  `json:"replicas,omitempty"`
	ResourceLabel                *string `json:"resourceLabel,omitempty"`
	SessionAffinity              *bool   `json:"sessionAffinity,omitempty"`
	ServiceWebRequestsOnRootPath *bool   `json:"serviceWebRequestsOnRootPath,omitempty"`
}

type CustomTemplate struct {
	ID    string               `json:"id"`
	Name  string               `json:"name"`
	Items []CustomTemplateItem `json:"items"`
}

type CustomTemplateItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CustomTemplateFile struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Content  string `json:"content"`
}
