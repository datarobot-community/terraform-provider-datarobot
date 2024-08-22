package client

const (
	RuntimeParameterTypeBoolean    string = "boolean"
	RuntimeParameterTypeCredential string = "credential"
	RuntimeParameterTypeNumeric    string = "numeric"
	RuntimeParameterTypeString     string = "string"
)

type CreateCustomModelRequest struct {
	Name                                        string `json:"name"`
	TargetType                                  string `json:"targetType"`
	TargetName                                  string `json:"targetName"`
	CustomModelType                             string `json:"customModelType"`
	Description                                 string `json:"description,omitempty"`
	IsProxyModel                                bool   `json:"isProxyModel,omitempty"`
	IsTrainingDataForVersionsPermanentlyEnabled bool   `json:"isTrainingDataForVersionsPermanentlyEnabled,omitempty"`
	Language                                    string `json:"language,omitempty"`
}

type CreateCustomModelFromLLMBlueprintRequest struct {
	LLMBlueprintID string `json:"llmBlueprintId"`
}

type CreateCustomModelVersionFromLLMBlueprintResponse struct {
	CustomModelID string `json:"customModelId"`
}

type CreateCustomModelVersionResponse struct {
	CustomModelID string `json:"customModelId"`
	ID            string `json:"id"`
}

type CustomModelResponse struct {
	ID              string                     `json:"id"`
	Name            string                     `json:"name"`
	Description     string                     `json:"description"`
	LatestVersion   CustomModelVersionResponse `json:"latestVersion"`
	TargetType      string                     `json:"targetType"`
	TargetName      string                     `json:"targetName"`
	CustomModelType string                     `json:"customModelType"`
	Language        string                     `json:"language"`
}

type CustomModelVersionResponse struct {
	ID                       string                     `json:"id"`
	Description              string                     `json:"description"`
	CustomModelID            string                     `json:"customModelId"`
	BaseEnvironmentID        string                     `json:"baseEnvironmentId"`
	BaseEnvironmentVersionID string                     `json:"baseEnvironmentVersionId"`
	RuntimeParameters        []RuntimeParameterResponse `json:"runtimeParameters"`
	Items                    []CustomModelVersionItem   `json:"items"`
}

type CustomModelVersionItem struct {
	ID                 string `json:"id"`
	FileName           string `json:"fileName"`
	FileSource         string `json:"fileSource"`
	FilePath           string `json:"filePath"`
	RepositoryLocation string `json:"repositoryLocation"`
	RepositoryName     string `json:"repositoryName"`
	RepositoryFilePath string `json:"repositoryFilePath"`
	Ref                string `json:"ref"`
	CommitSha          string `json:"commitSha"`
	StoragePath        string `json:"storagePath"`
	WorkspaceID        string `json:"workspaceId"`
}

type CustomModelUpdate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type RuntimeParameterResponse struct {
	FieldName      string  `json:"fieldName"`
	Type           string  `json:"type,omitempty"`
	DefaultValue   any     `json:"defaultValue,omitempty"`
	CredentialType *string `json:"credentialType,omitempty"`
	AllowEmpty     bool    `json:"allowEmpty,omitempty"`
	Description    string  `json:"description,omitempty"`
	OverrideValue  any     `json:"overrideValue,omitempty"`
	CurrentValue   any     `json:"currentValue,omitempty"`
}

type CreateCustomModelVersionCreateFromLatestRequest struct {
	IsMajorUpdate            string   `json:"isMajorUpdate"`
	BaseEnvironmentID        string   `json:"baseEnvironmentId,omitempty"`
	BaseEnvironmentVersionID string   `json:"baseEnvironmentVersionId,omitempty"`
	RuntimeParameterValues   string   `json:"runtimeParameterValues,omitempty"`
	FilesToDelete            []string `json:"filesToDelete,omitempty"`
}

type CreateCustomModelVersionFromFilesRequest struct {
	BaseEnvironmentID string     `json:"baseEnvironmentId"`
	Files             []FileInfo `json:"files"`
}

type CreateCustomModelVersionFromRemoteRepositoryRequest struct {
	IsMajorUpdate     bool     `json:"isMajorUpdate"`
	BaseEnvironmentID string   `json:"baseEnvironmentId,omitempty"`
	RepositoryID      string   `json:"repositoryId,omitempty"`
	Ref               string   `json:"ref,omitempty"`
	SourcePath        []string `json:"sourcePath,omitempty"`
}

type RuntimeParameterValueRequest struct {
	FieldName string `json:"fieldName"`
	Type      string `json:"type,omitempty"`
	Value     string `json:"value"`
}

type ListExecutionEnvironmentsResponse struct {
	Data []ExecutionEnvironment `json:"data"`
}

type ExecutionEnvironment struct {
	ID            string                      `json:"id"`
	Name          string                      `json:"name"`
	Description   string                      `json:"description"`
	LatestVersion ExecutionEnvironmentVersion `json:"latestVersion"`
}

type ExecutionEnvironmentVersion struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type ListGuardTemplatesResponse struct {
	Data []GuardTemplate `json:"data"`
}

type GuardTemplate struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Type          string            `json:"type"`
	AllowedStages []string          `json:"allowedStages"`
	Intervention  GuardIntervention `json:"intervention"`
	OOTBType      string            `json:"ootbType,omitempty"`
	LlmType       string            `json:"llmType,omitempty"`
	ErrorMessage  string            `json:"errorMessage,omitempty"`
	IsValid       bool              `json:"isValid,omitempty"`
	NemoInfo      NemoInfo          `json:"nemoInfo,omitempty"`
}

type NemoInfo struct {
	Actions      string `json:"actions,omitempty"`
	BlockedTerms string `json:"blockedTerms,omitempty"`
	LlmPrompts   string `json:"llmPrompts,omitempty"`
	MainConfig   string `json:"mainConfig,omitempty"`
	RailsConfig  string `json:"railsConfig,omitempty"`
}

type GuardIntervention struct {
	Action           string           `json:"action"`
	AllowedActions   []string         `json:"allowedActions"`
	Conditions       []GuardCondition `json:"conditions"`
	ConditionLogic   string           `json:"conditionLogic,omitempty"`
	ModifyMessage    string           `json:"modifyMessage,omitempty"`
	Message          string           `json:"message,omitempty"`
	SendNotification bool             `json:"sendNotification,omitempty"`
}

type GuardCondition struct {
	Comparator string  `json:"comparator"`
	Comparand  float64 `json:"comparand"`
}

type GuardConfigurationResponse struct {
	Data []GuardConfiguration `json:"data"`
}

type GuardConfiguration struct {
	ID           string            `json:"id,omitempty"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Stages       []string          `json:"stages"`
	Type         string            `json:"type"`
	OOTBType     string            `json:"ootbType,omitempty"`
	Intervention GuardIntervention `json:"intervention"`
	ErrorMessage string            `json:"errorMessage,omitempty"`
	IsValid      bool              `json:"isValid,omitempty"`
	LlmType      string            `json:"llmType,omitempty"`
	DeploymentID string            `json:"deploymentId,omitempty"`
	NemoInfo     NemoInfo          `json:"nemoInfo,omitempty"`
	ModelInfo    GuardModelInfo    `json:"modelInfo,omitempty"`
}

type GuardModelInfo struct {
	InputColumnName           string   `json:"inputColumnName,omitempty"`
	OutputColumnName          string   `json:"outputColumnName,omitempty"`
	TargetType                string   `json:"targetType,omitempty"`
	ClassNames                []string `json:"classNames,omitempty"`
	ModelID                   string   `json:"modelId,omitempty"`
	ModelName                 string   `json:"modelName,omitempty"`
	ReplacementTextColumnName string   `json:"replacementTextColumnName,omitempty"`
}

type CreateCustomModelVersionFromGuardsConfigurationRequest struct {
	CustomModelID string                         `json:"customModelId"`
	Data          []GuardConfiguration           `json:"data"`
	OverallConfig OverallModerationConfiguration `json:"overallConfig"`
}

type OverallModerationConfiguration struct {
	TimeoutAction string `json:"timeoutAction"`
	TimeoutSec    int    `json:"timeoutSec"`
}

type CreateCustomModelVersionFromGuardsConfigurationResponse struct {
	CustomModelVersionID string `json:"customModelVersionId"`
}
