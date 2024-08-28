package client

type CreateDeploymentFromModelPackageRequest struct {
	ModelPackageID          string `json:"modelPackageId"`
	PredictionEnvironmentID string `json:"predictionEnvironmentId"`
	Label                   string `json:"label"`
	Importance              string `json:"importance,omitempty"`
}

type DeploymentCreateResponse struct {
	ID string `json:"id"`
}

type DeploymentRetrieveResponse struct {
	ID                    string                `json:"id"`
	Label                 string                `json:"label"`
	Status                string                `json:"status"`
	Model                 Model                 `json:"model"`
	ModelPackage          ModelPackage          `json:"modelPackage"`
	PredictionEnvironment PredictionEnvironment `json:"predictionEnvironment"`
}

type Model struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	TargetName string `json:"targetName"`
	TargetType string `json:"targetType"`
}

type ModelPackage struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	RegisteredModelID string `json:"registeredModelId"`
}

type UpdateDeploymentRequest struct {
	Label string `json:"label"`
}

type DeploymentSettings struct {
	AssociationID             *AssociationIDSetting `json:"associationId,omitempty"`
	BatchMonitoring           *BasicSetting         `json:"batchMonitoring,omitempty"`
	ChallengerModels          *BasicSetting         `json:"challengerModels,omitempty"`
	FeatureDrift              *BasicSetting         `json:"featureDrift,omitempty"`
	Humility                  *BasicSetting         `json:"humility,omitempty"`
	PredictionsSettings       *PredictionsSetting   `json:"predictionsSettings,omitempty"`
	PredictionsDataCollection *BasicSetting         `json:"predictionsDataCollection,omitempty"`
	SegmentAnalysis           *BasicSetting         `json:"segmentAnalysis,omitempty"`
	TargetDrift               *BasicSetting         `json:"targetDrift,omitempty"`
}

type AssociationIDSetting struct {
	AutoGenerateID               bool     `json:"autoGenerateId"`
	RequiredInPredictionRequests bool     `json:"requiredInPredictionRequests"`
	ColumnNames                  []string `json:"columnNames,omitempty"`
}

type BasicSetting struct {
	Enabled bool `json:"enabled"`
}

type PredictionsSetting struct {
	MinComputes int  `json:"minComputes"`
	MaxComputes int  `json:"maxComputes"`
	RealTime    bool `json:"realTime"`
}

type SegmentAnalysisSetting struct {
	BasicSetting
	CustomAttributes []string `json:"customAttributes,omitempty"`
}

type ValidateDeployemntModelReplacementRequest struct {
	ModelPackageID string `json:"modelPackageId"`
}

type ValidateDeployemntModelReplacementResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type UpdateDeploymentModelRequest struct {
	ModelPackageID string `json:"modelPackageId"`
	Reason         string `json:"reason"`
}

type TaskStatusResponse struct {
	StatusID    string `json:"statusId"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	Code        int    `json:"code"`
	Description string `json:"description"`
	StatusType  string `json:"statusType"`
}
