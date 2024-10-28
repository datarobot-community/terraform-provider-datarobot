package client

const (
	RuntimeParameterTypeBoolean    string = "boolean"
	RuntimeParameterTypeCredential string = "credential"
	RuntimeParameterTypeNumeric    string = "numeric"
	RuntimeParameterTypeString     string = "string"
)

type CreateCustomModelRequest struct {
	Name                string   `json:"name"`
	TargetType          string   `json:"targetType"`
	CustomModelType     string   `json:"customModelType"`
	TargetName          string   `json:"targetName,omitempty"`
	NegativeClassLabel  string   `json:"negativeClassLabel,omitempty"`
	PositiveClassLabel  string   `json:"positiveClassLabel,omitempty"`
	PredictionThreshold float64  `json:"predictionThreshold,omitempty"`
	Description         string   `json:"description,omitempty"`
	IsProxyModel        bool     `json:"isProxyModel,omitempty"`
	Language            string   `json:"language,omitempty"`
	ClassLabels         []string `json:"classLabels,omitempty"`
}

type CreateCustomModelFromLLMBlueprintRequest struct {
	LLMBlueprintID string `json:"llmBlueprintId"`
}

type CreateCustomModelVersionFromLLMBlueprintResponse struct {
	CustomModelID string `json:"customModelId"`
}

type CustomModel struct {
	ID                                          string             `json:"id"`
	Name                                        string             `json:"name"`
	Description                                 string             `json:"description"`
	LatestVersion                               CustomModelVersion `json:"latestVersion"`
	TargetType                                  string             `json:"targetType"`
	TargetName                                  string             `json:"targetName"`
	CustomModelType                             string             `json:"customModelType"`
	Language                                    string             `json:"language"`
	PositiveClassLabel                          string             `json:"positiveClassLabel"`
	NegativeClassLabel                          string             `json:"negativeClassLabel"`
	PredictionThreshold                         float64            `json:"predictionThreshold"`
	ClassLabels                                 []string           `json:"classLabels,omitempty"`
	IsTrainingDataForVersionsPermanentlyEnabled bool               `json:"isTrainingDataForVersionsPermanentlyEnabled"`
	IsProxyModel                                bool               `json:"isProxyModel"`
	DeploymentsCount                            int64              `json:"deploymentsCount"`
}

type CustomModelVersion struct {
	ID                       string                   `json:"id"`
	Description              string                   `json:"description"`
	CustomModelID            string                   `json:"customModelId"`
	BaseEnvironmentID        string                   `json:"baseEnvironmentId"`
	BaseEnvironmentVersionID string                   `json:"baseEnvironmentVersionId"`
	Dependencies             []Dependency             `json:"dependencies"`
	RuntimeParameters        []RuntimeParameter       `json:"runtimeParameters"`
	Items                    []FileItem               `json:"items"`
	MaximumMemory            *int64                   `json:"maximumMemory"`
	NetworkEgressPolicy      *string                  `json:"networkEgressPolicy"`
	Replicas                 *int64                   `json:"replicas"`
	ResourceBundleID         *string                  `json:"resourceBundleId"`
	TrainingData             *CustomModelTrainingData `json:"trainingData"`
	HoldoutData              *CustomModelHoldoutData  `json:"holdoutData"`
	IsFrozen                 bool                     `json:"isFrozen"`
}

type Dependency struct {
	PackageName string `json:"packageName"`
}

type FileItem struct {
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

type UpdateCustomModelRequest struct {
	Name                                        string   `json:"name,omitempty"`
	Description                                 string   `json:"description,omitempty"`
	TargetName                                  string   `json:"targetName,omitempty"`
	PositiveClassLabel                          string   `json:"positiveClassLabel,omitempty"`
	NegativeClassLabel                          string   `json:"negativeClassLabel,omitempty"`
	PredictionThreshold                         float64  `json:"predictionThreshold,omitempty"`
	Language                                    string   `json:"language,omitempty"`
	ClassLabels                                 []string `json:"classLabels,omitempty"`
	IsTrainingDataForVersionsPermanentlyEnabled bool     `json:"isTrainingDataForVersionsPermanentlyEnabled,omitempty"`
}

type RuntimeParameter struct {
	FieldName      string  `json:"fieldName"`
	Type           string  `json:"type,omitempty"`
	DefaultValue   any     `json:"defaultValue,omitempty"`
	CredentialType *string `json:"credentialType,omitempty"`
	AllowEmpty     bool    `json:"allowEmpty,omitempty"`
	Description    string  `json:"description,omitempty"`
	OverrideValue  any     `json:"overrideValue,omitempty"`
	CurrentValue   any     `json:"currentValue,omitempty"`
}

type CreateCustomModelVersionFromLatestRequest struct {
	IsMajorUpdate            string   `json:"isMajorUpdate"`
	BaseEnvironmentID        string   `json:"baseEnvironmentId,omitempty"`
	BaseEnvironmentVersionID string   `json:"baseEnvironmentVersionId,omitempty"`
	RuntimeParameterValues   string   `json:"runtimeParameterValues,omitempty"`
	FilesToDelete            []string `json:"filesToDelete,omitempty"`
	Replicas                 int64    `json:"replicas,omitempty"`
	MaximumMemory            int64    `json:"maximumMemory,omitempty"`
	NetworkEgressPolicy      string   `json:"networkEgressPolicy,omitempty"`
	ResourceBundleID         *string  `json:"resourceBundleId"`
	KeepTrainingHoldoutData  *bool    `json:"keepTrainingHoldoutData,omitempty"`
	TrainingData             string   `json:"trainingData,omitempty"`
	HoldoutData              string   `json:"holdoutData,omitempty"`
}

type CustomModelTrainingData struct {
	DatasetID            string                       `json:"datasetId,omitempty"`
	DatasetVersionID     string                       `json:"datasetVersionId,omitempty"`
	DatasetName          string                       `json:"datasetName,omitempty"`
	AssignmentInProgress bool                         `json:"assignmentInProgress,omitempty"`
	AssignmentError      *TrainingDataAssignmentError `json:"assignmentError,omitempty"`
}

type TrainingDataAssignmentError struct {
	Message string `json:"message"`
}

type CustomModelHoldoutData struct {
	PartitionColumn *string `json:"partitionColumn,omitempty"`
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
	Value     *any   `json:"value"`
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
	ModelInfo     GuardModelInfo    `json:"modelInfo,omitempty"`
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
	Comparator string `json:"comparator"`
	Comparand  any    `json:"comparand"`
}

type GuardConfigurationResponse struct {
	Data []GuardConfiguration `json:"data"`
}

type GuardConfiguration struct {
	ID                 string            `json:"id,omitempty"`
	Name               string            `json:"name"`
	Description        string            `json:"description"`
	Stages             []string          `json:"stages"`
	Type               string            `json:"type"`
	OOTBType           string            `json:"ootbType,omitempty"`
	Intervention       GuardIntervention `json:"intervention"`
	ErrorMessage       string            `json:"errorMessage,omitempty"`
	IsValid            bool              `json:"isValid,omitempty"`
	LlmType            string            `json:"llmType,omitempty"`
	DeploymentID       string            `json:"deploymentId,omitempty"`
	NemoInfo           NemoInfo          `json:"nemoInfo,omitempty"`
	ModelInfo          GuardModelInfo    `json:"modelInfo,omitempty"`
	OpenAICredential   string            `json:"openaiCredential,omitempty"`
	OpenAIApiBase      string            `json:"openaiApiBase,omitempty"`
	OpenAIDeploymentID string            `json:"openaiDeploymentId,omitempty"`
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

type DependencyBuild struct {
	BuildStatus string `json:"buildStatus"`
}
