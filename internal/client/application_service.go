package client

import (
	"encoding/json"
	"fmt"
)

type ScopeLevel string

const (
	// App requires API Key with permission only for GET requests
	ViewerLevel ScopeLevel = "viewer"

	// App requires API Key with permission only for
	// GET, POST, PUT, PATCH requests
	UserLevel ScopeLevel = "user"

	// App requires API Key without permissions restriction
	AdminLevel ScopeLevel = "admin"

	// App does not need user's API Key
	NoRequirements ScopeLevel = ""
)

var levelsChoice = []ScopeLevel{
	ViewerLevel,
	UserLevel,
	AdminLevel,
	NoRequirements,
}

func (level ScopeLevel) MarshalJSON() ([]byte, error) {
	if level == NoRequirements {
		return json.Marshal(nil)
	}
	return json.Marshal(string(level))
}

func (level *ScopeLevel) UnmarshalJSON(data []byte) (err error) {
	var providedString string
	if err := json.Unmarshal(data, &providedString); err != nil {
		return err
	}

	var providedLevel = ScopeLevel(providedString)

	for _, levelValue := range levelsChoice {
		if providedLevel == levelValue {
			*level = providedLevel
			return
		}
	}

	return fmt.Errorf("invalid level: %v, choices: %v", providedString, levelsChoice)
}

type CreateQAApplicationRequest struct {
	DeploymentID string `json:"deploymentId"`
}

type CreateCustomApplicationRequest struct {
	ApplicationSourceVersionID string                `json:"applicationSourceVersionId,omitempty"`
	EnvironmentID              string                `json:"environmentId,omitempty"`
	Resources                  *ApplicationResources `json:"resources,omitempty"`
	RequiredKeyScopeLevel      ScopeLevel            `json:"requiredKeyScopeLevel,omitempty"`
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
	RequiredKeyScopeLevel            ScopeLevel            `json:"requiredKeyScopeLevel"`
}

type UpdateApplicationRequest struct {
	Name                             string   `json:"name,omitempty"`
	CustomApplicationSourceVersionID string   `json:"customApplicationSourceVersionId,omitempty"`
	ExternalAccessEnabled            bool     `json:"externalAccessEnabled"`
	ExternalAccessRecipients         []string `json:"externalAccessRecipients"`
	AllowAutoStopping                bool     `json:"allowAutoStopping"`
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
	RequiredKeyScopeLevel    ScopeLevel            `json:"requiredKeyScopeLevel,omitempty"`
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
	RequiredKeyScopeLevel    ScopeLevel           `json:"requiredKeyScopeLevel"`
}

type UpdateApplicationSourceVersionRequest struct {
	BaseEnvironmentID        string                `json:"baseEnvironmentId,omitempty"`
	BaseEnvironmentVersionID string                `json:"baseEnvironmentVersionId,omitempty"`
	Resources                *ApplicationResources `json:"resources,omitempty"`
	FilesToDelete            []string              `json:"filesToDelete,omitempty"`
	RuntimeParameterValues   string                `json:"runtimeParameterValues,omitempty"`
	RequiredKeyScopeLevel    ScopeLevel            `json:"requiredKeyScopeLevel"`
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
