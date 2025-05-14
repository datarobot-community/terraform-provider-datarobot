package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

type Service interface {
	// Use Case
	CreateUseCase(ctx context.Context, req *UseCaseRequest) (*CreateUseCaseResponse, error)
	GetUseCase(ctx context.Context, id string) (*UseCaseResponse, error)
	UpdateUseCase(ctx context.Context, id string, req *UseCaseRequest) (*UseCaseResponse, error)
	DeleteUseCase(ctx context.Context, id string) error
	AddEntityToUseCase(ctx context.Context, useCaseID, entityType, entityID string) error
	RemoveEntityFromUseCase(ctx context.Context, useCaseID, entityType, entityID string) error

	// Remote Repository
	CreateRemoteRepository(ctx context.Context, req *CreateRemoteRepositoryRequest) (*RemoteRepositoryResponse, error)
	GetRemoteRepository(ctx context.Context, id string) (*RemoteRepositoryResponse, error)
	UpdateRemoteRepository(ctx context.Context, id string, req *UpdateRemoteRepositoryRequest) (*RemoteRepositoryResponse, error)
	DeleteRemoteRepository(ctx context.Context, id string) error

	// Data Set
	CreateDataset(ctx context.Context, req *CreateDatasetRequest) (*CreateDatasetResponse, error)
	CreateDatasetFromFile(ctx context.Context, fileName string, content []byte) (*CreateDatasetVersionResponse, error)
	CreateDatasetFromURL(ctx context.Context, req *CreateDatasetFromURLRequest) (*CreateDatasetVersionResponse, error)
	CreateDatasetFromDataSource(ctx context.Context, req *CreateDatasetFromDatasourceRequest) (*CreateDatasetVersionResponse, error)
	GetDataset(ctx context.Context, id string) (*Dataset, error)
	UpdateDataset(ctx context.Context, id string, req *UpdateDatasetRequest) (*Dataset, error)
	DeleteDataset(ctx context.Context, id string) error

	// Data Store
	CreateDatastore(ctx context.Context, req *CreateDatastoreRequest) (*Datastore, error)
	GetDatastore(ctx context.Context, id string) (*Datastore, error)
	UpdateDatastore(ctx context.Context, id string, req *UpdateDatastoreRequest) (*Datastore, error)
	DeleteDatastore(ctx context.Context, id string) error
	ListDatastoreCredentials(ctx context.Context, id string) ([]Credential, error)
	ListExternalDataDrivers(ctx context.Context, req *ListExternalDataDriversRequest) ([]ExternalDataDriver, error)
	ListExternalConnectors(ctx context.Context) ([]ExternalConnector, error)
	TestDataStoreConnection(ctx context.Context, id string, req *TestDatastoreConnectionRequest) (*TestDatastoreConnectionResponse, error)

	// Data Source
	CreateDatasource(ctx context.Context, req *CreateDatasourceRequest) (*Datasource, error)
	GetDatasource(ctx context.Context, id string) (*Datasource, error)
	ListDatasources(ctx context.Context, req *ListDataSourcesRequest) ([]Datasource, error)
	UpdateDatasource(ctx context.Context, id string, req *UpdateDatasourceRequest) (*Datasource, error)
	DeleteDatasource(ctx context.Context, id string) error

	// Vector Database
	CreateVectorDatabase(ctx context.Context, req *CreateVectorDatabaseRequest) (*VectorDatabase, error)
	GetVectorDatabase(ctx context.Context, id string) (*VectorDatabase, error)
	UpdateVectorDatabase(ctx context.Context, id string, req *UpdateVectorDatabaseRequest) (*VectorDatabase, error)
	DeleteVectorDatabase(ctx context.Context, id string) error
	IsVectorDatabaseReady(ctx context.Context, id string) (bool, error)

	// Playground
	CreatePlayground(ctx context.Context, req *CreatePlaygroundRequest) (*CreatePlaygroundResponse, error)
	GetPlayground(ctx context.Context, id string) (*PlaygroundResponse, error)
	UpdatePlayground(ctx context.Context, id string, req *UpdatePlaygroundRequest) (*PlaygroundResponse, error)
	DeletePlayground(ctx context.Context, id string) error

	// LLM Blueprint
	CreateLLMBlueprint(ctx context.Context, req *CreateLLMBlueprintRequest) (*LLMBlueprint, error)
	GetLLMBlueprint(ctx context.Context, id string) (*LLMBlueprint, error)
	UpdateLLMBlueprint(ctx context.Context, id string, req *UpdateLLMBlueprintRequest) (*LLMBlueprint, error)
	DeleteLLMBlueprint(ctx context.Context, id string) error

	// Custom Job
	CreateCustomJob(ctx context.Context, req *CreateCustomJobRequest) (*CustomJob, error)
	GetCustomJob(ctx context.Context, id string) (*CustomJob, error)
	UpdateCustomJob(ctx context.Context, id string, req *UpdateCustomJobRequest) (*CustomJob, error)
	UpdateCustomJobFiles(ctx context.Context, id string, files []FileInfo) (*CustomJob, error)
	ListCustomJobMetrics(ctx context.Context, id string) ([]CustomJobMetric, error)
	DeleteCustomJob(ctx context.Context, id string) error

	// Custom Job Schedule
	CreateCustomJobSchedule(ctx context.Context, id string, req CreateaCustomJobScheduleRequest) (*CustomJobScheduleResponse, error)
	ListCustomJobSchedules(ctx context.Context, id string) ([]CustomJobScheduleResponse, error)
	UpdateCustomJobSchedule(ctx context.Context, id string, scheduleID string, req CreateaCustomJobScheduleRequest) (*CustomJobScheduleResponse, error)
	DeleteCustomJobSchedule(ctx context.Context, id string, scheduleID string) error

	// Custom Metric Template
	CreateHostedCustomMetricTemplate(ctx context.Context, customJobID string, req *HostedCustomMetricTemplateRequest) (*HostedCustomMetricTemplate, error)
	GetHostedCustomMetricTemplate(ctx context.Context, customJobID string) (*HostedCustomMetricTemplate, error)
	UpdateHostedCustomMetricTemplate(ctx context.Context, customJobID string, req *HostedCustomMetricTemplateRequest) (*HostedCustomMetricTemplate, error)

	// Custom Model
	CreateCustomModel(ctx context.Context, req *CreateCustomModelRequest) (*CustomModel, error)
	CreateCustomModelFromLLMBlueprint(ctx context.Context, req *CreateCustomModelFromLLMBlueprintRequest) (*CreateCustomModelVersionFromLLMBlueprintResponse, error)
	CreateCustomModelVersionCreateFromLatest(ctxc context.Context, id string, req *CreateCustomModelVersionFromLatestRequest) (*CustomModelVersion, error)
	CreateCustomModelVersionFromFiles(ctx context.Context, id string, req *CreateCustomModelVersionFromFilesRequest) (*CustomModelVersion, error)
	CreateCustomModelVersionFromRemoteRepository(ctx context.Context, id string, req *CreateCustomModelVersionFromRemoteRepositoryRequest) (*CustomModelVersion, string, error)
	GetCustomModel(ctx context.Context, id string) (*CustomModel, error)
	IsCustomModelReady(ctx context.Context, id string) (bool, error)
	UpdateCustomModel(ctx context.Context, id string, req *UpdateCustomModelRequest) (*CustomModel, error)
	DeleteCustomModel(ctx context.Context, id string) error
	ListCustomModels(ctx context.Context) ([]CustomModel, error)
	ListCustomModelVersions(ctx context.Context, id string) ([]CustomModelVersion, error)
	ListGuardTemplates(ctx context.Context) ([]GuardTemplate, error)
	GetGuardConfigurationsForCustomModelVersion(ctx context.Context, id string) (*GuardConfigurationResponse, error)
	GetOverallModerationConfigurationForCustomModelVersion(ctx context.Context, id string) (*OverallModerationConfiguration, error)
	CreateCustomModelVersionFromGuardConfigurations(ctx context.Context, id string, req *CreateCustomModelVersionFromGuardsConfigurationRequest) (*CreateCustomModelVersionFromGuardsConfigurationResponse, error)
	CreateDependencyBuild(ctx context.Context, id string, versionID string) (*DependencyBuild, error)
	GetDependencyBuild(ctx context.Context, id string, versionID string) (*DependencyBuild, error)

	// Custom Model LLM Validation
	CreateCustomModelLLMValidation(ctx context.Context, req *CreateCustomModelLLMValidationRequest) (*CustomModelLLMValidation, string, error)
	GetCustomModelLLMValidation(ctx context.Context, id string) (*CustomModelLLMValidation, error)
	UpdateCustomModelLLMValidation(ctx context.Context, id string, req *UpdateCustomModelLLMValidationRequest) (*CustomModelLLMValidation, error)
	DeleteCustomModelLLMValidation(ctx context.Context, id string) error

	// Registered Model
	CreateRegisteredModelFromCustomModelVersion(ctx context.Context, req *CreateRegisteredModelFromCustomModelRequest) (*RegisteredModelVersion, error)
	CreateRegisteredModelFromLeaderboard(ctx context.Context, req *CreateRegisteredModelFromLeaderboardRequest) (*RegisteredModelVersion, error)
	UpdateRegisteredModelVersion(ctx context.Context, registeredModelId string, versionId string, req *UpdateRegisteredModelVersionRequest) (*RegisteredModelVersion, error)
	ListRegisteredModelVersions(ctx context.Context, id string) ([]RegisteredModelVersion, error)
	GetLatestRegisteredModelVersion(ctx context.Context, id string) (*RegisteredModelVersion, error)
	GetRegisteredModelVersion(ctx context.Context, registeredModelId string, versionId string) (*RegisteredModelVersion, error)
	IsRegisteredModelVersionReady(ctx context.Context, registeredModelId string, versionId string) (bool, error)
	ListRegisteredModels(ctx context.Context, req *ListRegisteredModelsRequest) ([]RegisteredModel, error)
	GetRegisteredModel(ctx context.Context, id string) (*RegisteredModel, error)
	UpdateRegisteredModel(ctx context.Context, id string, req *UpdateRegisteredModelRequest) (*RegisteredModel, error)
	DeleteRegisteredModel(ctx context.Context, id string) error

	// Prediction Environment
	CreatePredictionEnvironment(ctx context.Context, req *PredictionEnvironmentRequest) (*PredictionEnvironment, error)
	GetPredictionEnvironment(ctx context.Context, id string) (*PredictionEnvironment, error)
	UpdatePredictionEnvironment(ctx context.Context, id string, req *PredictionEnvironmentRequest) (*PredictionEnvironment, error)
	DeletePredictionEnvironment(ctx context.Context, id string) error

	// Deployment
	CreateDeploymentFromModelPackage(ctx context.Context, req *CreateDeploymentFromModelPackageRequest) (*DeploymentCreateResponse, string, error)
	GetDeployment(ctx context.Context, id string) (*Deployment, error)
	UpdateDeployment(ctx context.Context, id string, req *UpdateDeploymentRequest) (*Deployment, error)
	DeleteDeployment(ctx context.Context, id string) error
	ValidateDeploymentModelReplacement(ctx context.Context, id string, req *ValidateDeployemntModelReplacementRequest) (*ValidateDeployemntModelReplacementResponse, error)
	UpdateDeploymentRuntimeParameters(ctx context.Context, id string, req *UpdateDeploymentRuntimeParametersRequest) (*Deployment, error)
	ListDeploymentRuntimeParameters(ctx context.Context, id string) ([]RuntimeParameter, error)
	DeactivateDeployment(ctx context.Context, id string) (*Deployment, error)
	ActivateDeployment(ctx context.Context, id string) (*Deployment, error)
	GetDeploymentSettings(ctx context.Context, id string) (*DeploymentSettings, error)
	UpdateDeploymentModel(ctx context.Context, id string, req *UpdateDeploymentModelRequest) (*Deployment, string, error)
	// Deployment: Settings
	UpdateDeploymentSettings(ctx context.Context, id string, req *DeploymentSettings) (*DeploymentSettings, error)
	GetDeploymentChallengerReplaySettings(ctx context.Context, id string) (*DeploymentChallengerReplaySettings, error)
	UpdateDeploymentChallengerReplaySettings(ctx context.Context, id string, req *DeploymentChallengerReplaySettings) (*DeploymentChallengerReplaySettings, error)
	GetDeploymentHealthSettings(ctx context.Context, id string) (*DeploymentHealthSettings, error)
	UpdateDeploymentHealthSettings(ctx context.Context, id string, req *DeploymentHealthSettings) (*DeploymentHealthSettings, error)
	GetDeploymentFeatureCacheSettings(ctx context.Context, id string) (*DeploymentFeatureCacheSettings, error)
	UpdateDeploymentFeatureCacheSettings(ctx context.Context, id string, req *DeploymentFeatureCacheSettings) (*DeploymentFeatureCacheSettings, error)
	GetDeploymentRetrainingSettings(ctx context.Context, deploymentID string) (*RetrainingSettingsRetrieve, error)
	UpdateDeploymentRetrainingSettings(ctx context.Context, deploymentID string, req *DeploymentRetrainingSettings) (*RetrainingSettings, error)
	// Deployment: Retraining policy
	CreateRetrainingPolicy(ctx context.Context, deploymentID string, req *RetrainingPolicyRequest) (*RetrainingPolicy, error)
	GetRetrainingPolicy(ctx context.Context, deploymentID, id string) (*RetrainingPolicy, error)
	UpdateRetrainingPolicy(ctx context.Context, deploymentID, id string, req *RetrainingPolicyRequest) (*RetrainingPolicy, error)
	DeleteRetrainingPolicy(ctx context.Context, deploymentID, id string) error

	// Notification Channel
	CreateNotificationChannel(ctx context.Context, req *CreateNotificationChannelRequest) (*NotificationChannel, error)
	GetNotificationChannel(ctx context.Context, relatedEntityType, relatedEntityID, id string) (*NotificationChannel, error)
	UpdateNotificationChannel(ctx context.Context, relatedEntityType, relatedEntityID, id string, req *UpdateNotificationChannelRequest) (*NotificationChannel, error)
	DeleteNotificationChannel(ctx context.Context, relatedEntityType, relatedEntityID, id string) error

	// Notification Policy
	CreateNotificationPolicy(ctx context.Context, req *CreateNotificationPolicyRequest) (*NotificationPolicy, error)
	GetNotificationPolicy(ctx context.Context, relatedEntityType, relatedEntityID, id string) (*NotificationPolicy, error)
	UpdateNotificationPolicy(ctx context.Context, relatedEntityType, relatedEntityID, id string, req *UpdateNotificationPolicyRequest) (*NotificationPolicy, error)
	DeleteNotificationPolicy(ctx context.Context, relatedEntityType, relatedEntityID, id string) error

	// Custom Metric
	CreateCustomMetricFromJob(ctx context.Context, deploymentID string, req *CreateCustomMetricFromJobRequest) (*CustomMetric, error)
	CreateCustomMetric(ctx context.Context, deploymentID string, req *CreateCustomMetricRequest) (*CustomMetric, error)
	GetCustomMetric(ctx context.Context, deploymentID string, id string) (*CustomMetric, error)
	UpdateCustomMetric(ctx context.Context, deploymentID string, id string, req *UpdateCustomMetricRequest) (*CustomMetric, error)
	DeleteCustomMetric(ctx context.Context, deploymentID string, id string) error

	// Batch Prediction Job Definition
	CreateBatchPredictionJobDefinition(ctx context.Context, req *BatchPredictionJobDefinitionRequest) (*BatchPredictionJobDefinition, error)
	GetBatchPredictionJobDefinition(ctx context.Context, id string) (*BatchPredictionJobDefinition, error)
	UpdateBatchPredictionJobDefinition(ctx context.Context, id string, req *BatchPredictionJobDefinitionRequest) (*BatchPredictionJobDefinition, error)
	DeleteBatchPredictionJobDefinition(ctx context.Context, id string) error

	// Application Source
	CreateApplicationSource(ctx context.Context) (*ApplicationSource, error)
	CreateApplicationSourceFromTemplate(ctx context.Context, req *CreateApplicationSourceFromTemplateRequest) (*ApplicationSource, error)
	GetApplicationSource(ctx context.Context, id string) (*ApplicationSource, error)
	UpdateApplicationSource(ctx context.Context, id string, req *UpdateApplicationSourceRequest) (*ApplicationSource, error)
	CreateApplicationSourceVersion(ctx context.Context, id string, req *CreateApplicationSourceVersionRequest) (*ApplicationSourceVersion, error)
	UpdateApplicationSourceVersion(ctx context.Context, id string, versionId string, req *UpdateApplicationSourceVersionRequest) (*ApplicationSourceVersion, error)
	UpdateApplicationSourceVersionFiles(ctx context.Context, id string, versionId string, files []FileInfo) (*ApplicationSourceVersion, error)
	GetApplicationSourceVersion(ctx context.Context, id string, versionId string) (*ApplicationSourceVersion, error)
	DeleteApplicationSource(ctx context.Context, id string) error

	// Custom Template
	GetCustomTemplate(ctx context.Context, id string) (*CustomTemplate, error)
	GetCustomTemplateFile(ctx context.Context, customTemplateID, fileID string) (*CustomTemplateFile, error)

	// Application
	CreateCustomApplication(ctx context.Context, req *CreateCustomApplicationeRequest) (*Application, error)
	CreateQAApplication(ctx context.Context, req *CreateQAApplicationRequest) (*Application, error)
	GetApplication(ctx context.Context, id string) (*Application, error)
	UpdateApplication(ctx context.Context, id string, req *UpdateApplicationRequest) (*Application, error)
	DeleteApplication(ctx context.Context, id string) error

	// Credential
	CreateCredential(ctx context.Context, req *CredentialRequest) (*Credential, error)
	GetCredential(ctx context.Context, id string) (*Credential, error)
	UpdateCredential(ctx context.Context, id string, req *CredentialRequest) (*Credential, error)
	DeleteCredential(ctx context.Context, id string) error
	ListCredentials(ctx context.Context) ([]Credential, error)

	// Execution Environment
	CreateExecutionEnvironment(ctx context.Context, req *CreateExecutionEnvironmentRequest) (*ExecutionEnvironment, error)
	GetExecutionEnvironment(ctx context.Context, id string) (*ExecutionEnvironment, error)
	UpdateExecutionEnvironment(ctx context.Context, id string, req *UpdateExecutionEnvironmentRequest) (*ExecutionEnvironment, error)
	DeleteExecutionEnvironment(ctx context.Context, id string) error
	ListExecutionEnvironments(ctx context.Context) ([]ExecutionEnvironment, error)
	CreateExecutionEnvironmentVersion(ctx context.Context, id string, req *CreateExecutionEnvironmentVersionRequest) (*ExecutionEnvironmentVersion, error)
	GetExecutionEnvironmentVersion(ctx context.Context, id, versionId string) (*ExecutionEnvironmentVersion, error)

	// Async Tasks
	GetTaskStatus(ctx context.Context, id string) (*TaskStatusResponse, error)
	GetGenAITaskStatus(ctx context.Context, id string) (*TaskStatusResponse, error)

	// User Info
	GetUserInfo(ctx context.Context) (*UserInfo, error)

	// API Gateway methods

	// Notebook
	ImportNotebookFromFile(ctx context.Context, fileName string, content []byte, useCaseID string) (*ImportNotebookResponse, error)
	GetNotebook(ctx context.Context, id string) (*Notebook, error)
	UpdateNotebook(ctx context.Context, id string, useCaseID string) (*Notebook, error)
	DeleteNotebook(ctx context.Context, id string) error

	// Add your service methods here
}

// Service for the DataRobot API.
type ServiceImpl struct {
	client      *Client
	apiGWClient *Client
}

// NewService creates a new API service.
func NewService(c *Client) Service {
	if c == nil {
		panic("client is required")
	}
	// Construct the API Gateway client from the client config
	apiGWConfig := *c.cfg
	apiGWConfig.Endpoint = apiGWConfig.BaseURL() + "/api-gw"
	apiGWClient := NewClient(&apiGWConfig)

	return &ServiceImpl{
		client:      c,
		apiGWClient: apiGWClient,
	}
}

// Playground Service Implementation.
func (s *ServiceImpl) CreatePlayground(ctx context.Context, req *CreatePlaygroundRequest) (*CreatePlaygroundResponse, error) {
	return Post[CreatePlaygroundResponse](s.client, ctx, "/genai/playgrounds/", req)
}

func (s *ServiceImpl) GetPlayground(ctx context.Context, id string) (*PlaygroundResponse, error) {
	return Get[PlaygroundResponse](s.client, ctx, "/genai/playgrounds/"+id+"/")
}

func (s *ServiceImpl) UpdatePlayground(ctx context.Context, id string, req *UpdatePlaygroundRequest) (*PlaygroundResponse, error) {
	return Patch[PlaygroundResponse](s.client, ctx, "/genai/playgrounds/"+id+"/", req)
}

func (s *ServiceImpl) DeletePlayground(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/genai/playgrounds/"+id+"/")
}

// Data Set Service Implementation.
func (s *ServiceImpl) CreateDataset(ctx context.Context, req *CreateDatasetRequest) (*CreateDatasetResponse, error) {
	return Post[CreateDatasetResponse](s.client, ctx, "/datasets/", req)
}

func (s *ServiceImpl) CreateDatasetFromFile(ctx context.Context, fileName string, content []byte) (*CreateDatasetVersionResponse, error) {
	return uploadFileFromBinary[CreateDatasetVersionResponse](s.client, ctx, "/datasets/fromFile", http.MethodPost, fileName, content, map[string]string{})
}

func (s *ServiceImpl) CreateDatasetFromURL(ctx context.Context, req *CreateDatasetFromURLRequest) (*CreateDatasetVersionResponse, error) {
	return Post[CreateDatasetVersionResponse](s.client, ctx, "/datasets/fromURL/", req)
}

func (s *ServiceImpl) CreateDatasetFromDataSource(ctx context.Context, req *CreateDatasetFromDatasourceRequest) (*CreateDatasetVersionResponse, error) {
	return Post[CreateDatasetVersionResponse](s.client, ctx, "/datasets/fromDataSource/", req)
}

func (s *ServiceImpl) GetDataset(ctx context.Context, id string) (*Dataset, error) {
	return Get[Dataset](s.client, ctx, "/datasets/"+id+"/")
}

func (s *ServiceImpl) UpdateDataset(ctx context.Context, id string, req *UpdateDatasetRequest) (*Dataset, error) {
	return Patch[Dataset](s.client, ctx, "/datasets/"+id+"/", req)
}

func (s *ServiceImpl) DeleteDataset(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/datasets/"+id+"/")
}

// Data Store Service Implementation.
func (s *ServiceImpl) CreateDatastore(ctx context.Context, req *CreateDatastoreRequest) (*Datastore, error) {
	return Post[Datastore](s.client, ctx, "/externalDataStores/", req)
}

func (s *ServiceImpl) GetDatastore(ctx context.Context, id string) (*Datastore, error) {
	return Get[Datastore](s.client, ctx, "/externalDataStores/"+id+"/")
}

func (s *ServiceImpl) UpdateDatastore(ctx context.Context, id string, req *UpdateDatastoreRequest) (*Datastore, error) {
	return Patch[Datastore](s.client, ctx, "/externalDataStores/"+id+"/", req)
}

func (s *ServiceImpl) DeleteDatastore(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/externalDataStores/"+id+"/")
}

func (s *ServiceImpl) ListDatastoreCredentials(ctx context.Context, id string) ([]Credential, error) {
	return GetAllPages[Credential](s.client, ctx, "/externalDataStores/"+id+"/credentials/", nil)
}

func (s *ServiceImpl) ListExternalDataDrivers(ctx context.Context, req *ListExternalDataDriversRequest) ([]ExternalDataDriver, error) {
	return GetAllPages[ExternalDataDriver](s.client, ctx, "/externalDataDrivers/", req)
}

func (s *ServiceImpl) ListExternalConnectors(ctx context.Context) ([]ExternalConnector, error) {
	return GetAllPages[ExternalConnector](s.client, ctx, "/externalConnectors/", nil)
}

func (s *ServiceImpl) TestDataStoreConnection(ctx context.Context, id string, req *TestDatastoreConnectionRequest) (*TestDatastoreConnectionResponse, error) {
	return Post[TestDatastoreConnectionResponse](s.client, ctx, "/externalDataStores/"+id+"/test/", req)
}

// Data Source Service Implementation.
func (s *ServiceImpl) CreateDatasource(ctx context.Context, req *CreateDatasourceRequest) (*Datasource, error) {
	return Post[Datasource](s.client, ctx, "/externalDataSources/", req)
}

func (s *ServiceImpl) GetDatasource(ctx context.Context, id string) (*Datasource, error) {
	return Get[Datasource](s.client, ctx, "/externalDataSources/"+id+"/")
}

func (s *ServiceImpl) ListDatasources(ctx context.Context, req *ListDataSourcesRequest) ([]Datasource, error) {
	return GetAllPages[Datasource](s.client, ctx, "/externalDataSources/", req)
}

func (s *ServiceImpl) UpdateDatasource(ctx context.Context, id string, req *UpdateDatasourceRequest) (*Datasource, error) {
	return Patch[Datasource](s.client, ctx, "/externalDataSources/"+id+"/", req)
}

func (s *ServiceImpl) DeleteDatasource(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/externalDataSources/"+id+"/")
}

// Use Case Service Implementation.
func (s *ServiceImpl) AddEntityToUseCase(ctx context.Context, useCaseID, entityType, entityID string) error {
	_, err := Post[CreateVoidResponse](s.client, ctx, "/useCases/"+useCaseID+"/"+entityType+"s/"+entityID+"/", &CreateVoidRequest{})
	return err
}

func (s *ServiceImpl) RemoveEntityFromUseCase(ctx context.Context, useCaseID, entityType, entityID string) error {
	return Delete(s.client, ctx, "/useCases/"+useCaseID+"/"+entityType+"s/"+entityID+"/")
}

func (s *ServiceImpl) CreateUseCase(ctx context.Context, req *UseCaseRequest) (resp *CreateUseCaseResponse, err error) {
	return Post[CreateUseCaseResponse](s.client, ctx, "/useCases/", req)
}

func (s *ServiceImpl) GetUseCase(ctx context.Context, id string) (*UseCaseResponse, error) {
	return Get[UseCaseResponse](s.client, ctx, "/useCases/"+id+"/")
}

func (s *ServiceImpl) UpdateUseCase(ctx context.Context, id string, req *UseCaseRequest) (*UseCaseResponse, error) {
	return Patch[UseCaseResponse](s.client, ctx, "/useCases/"+id+"/", req)
}

func (s *ServiceImpl) DeleteUseCase(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/useCases/"+id+"/")
}

// Remote Repository Service Implementation.
func (s *ServiceImpl) CreateRemoteRepository(ctx context.Context, req *CreateRemoteRepositoryRequest) (*RemoteRepositoryResponse, error) {
	return Post[RemoteRepositoryResponse](s.client, ctx, "/remoteRepositories/", req)
}

func (s *ServiceImpl) GetRemoteRepository(ctx context.Context, id string) (*RemoteRepositoryResponse, error) {
	return Get[RemoteRepositoryResponse](s.client, ctx, "/remoteRepositories/"+id+"/")
}

func (s *ServiceImpl) UpdateRemoteRepository(ctx context.Context, id string, req *UpdateRemoteRepositoryRequest) (*RemoteRepositoryResponse, error) {
	return Patch[RemoteRepositoryResponse](s.client, ctx, "/remoteRepositories/"+id+"/", req)
}

func (s *ServiceImpl) DeleteRemoteRepository(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/remoteRepositories/"+id+"/")
}

// Vector Database Service Implementation.
func (s *ServiceImpl) CreateVectorDatabase(ctx context.Context, req *CreateVectorDatabaseRequest) (*VectorDatabase, error) {
	return Post[VectorDatabase](s.client, ctx, "/genai/vectorDatabases/", req)
}

func (s *ServiceImpl) GetVectorDatabase(ctx context.Context, id string) (*VectorDatabase, error) {
	return Get[VectorDatabase](s.client, ctx, "/genai/vectorDatabases/"+id+"/")
}

func (s *ServiceImpl) UpdateVectorDatabase(ctx context.Context, id string, req *UpdateVectorDatabaseRequest) (*VectorDatabase, error) {
	return Patch[VectorDatabase](s.client, ctx, "/genai/vectorDatabases/"+id+"/", req)
}

func (s *ServiceImpl) DeleteVectorDatabase(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/genai/vectorDatabases/"+id+"/")
}

func (s *ServiceImpl) IsVectorDatabaseReady(ctx context.Context, id string) (bool, error) {
	dataset, err := s.GetVectorDatabase(ctx, id)
	if err != nil {
		return false, err
	}
	if dataset.ExecutionStatus == "ERROR" {
		return false, NewGenericError("Vector Database execution failed")
	}
	return dataset.ExecutionStatus == "COMPLETED", nil
}

func (s *ServiceImpl) CreateLLMBlueprint(ctx context.Context, req *CreateLLMBlueprintRequest) (*LLMBlueprint, error) {
	return Post[LLMBlueprint](s.client, ctx, "/genai/llmBlueprints/", req)
}

func (s *ServiceImpl) GetLLMBlueprint(ctx context.Context, id string) (*LLMBlueprint, error) {
	return Get[LLMBlueprint](s.client, ctx, "/genai/llmBlueprints/"+id+"/")
}

func (s *ServiceImpl) UpdateLLMBlueprint(ctx context.Context, id string, req *UpdateLLMBlueprintRequest) (*LLMBlueprint, error) {
	return Patch[LLMBlueprint](s.client, ctx, "/genai/llmBlueprints/"+id+"/", req)
}

func (s *ServiceImpl) DeleteLLMBlueprint(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/genai/llmBlueprints/"+id+"/")
}

func (s *ServiceImpl) CreateCustomJob(ctx context.Context, req *CreateCustomJobRequest) (*CustomJob, error) {
	return Post[CustomJob](s.client, ctx, "/customJobs/", req)
}

func (s *ServiceImpl) GetCustomJob(ctx context.Context, id string) (*CustomJob, error) {
	return Get[CustomJob](s.client, ctx, "/customJobs/"+id+"/")
}

func (s *ServiceImpl) UpdateCustomJob(ctx context.Context, id string, req *UpdateCustomJobRequest) (*CustomJob, error) {
	return Patch[CustomJob](s.client, ctx, "/customJobs/"+id+"/", req)
}

func (s *ServiceImpl) UpdateCustomJobFiles(ctx context.Context, id string, files []FileInfo) (*CustomJob, error) {
	return uploadFilesFromBinaries[CustomJob](s.client, ctx, "/customJobs/"+id+"/", http.MethodPatch, files, map[string]string{})
}

func (s *ServiceImpl) ListCustomJobMetrics(ctx context.Context, id string) ([]CustomJobMetric, error) {
	return GetAllPages[CustomJobMetric](s.client, ctx, "/customJobs/"+id+"/customMetrics/", nil)
}
func (s *ServiceImpl) CreateCustomJobSchedule(ctx context.Context, id string, req CreateaCustomJobScheduleRequest) (*CustomJobScheduleResponse, error) {
	return Post[CustomJobScheduleResponse](s.client, ctx, fmt.Sprintf("/customJobs/%s/schedules/", id), req)
}

func (s *ServiceImpl) ListCustomJobSchedules(ctx context.Context, id string) ([]CustomJobScheduleResponse, error) {
	return GetAllPages[CustomJobScheduleResponse](s.client, ctx, fmt.Sprintf("/customJobs/%s/schedules/", id), nil)
}

func (s *ServiceImpl) DeleteCustomJobSchedule(ctx context.Context, id string, scheduleID string) error {
	return Delete(s.client, ctx, fmt.Sprintf("/customJobs/%s/schedules/%s/", id, scheduleID))
}

func (s *ServiceImpl) UpdateCustomJobSchedule(ctx context.Context, id string, scheduleID string, req CreateaCustomJobScheduleRequest) (*CustomJobScheduleResponse, error) {
	return Patch[CustomJobScheduleResponse](s.client, ctx, fmt.Sprintf("/customJobs/%s/schedules/%s/", id, scheduleID), req)
}

func (s *ServiceImpl) DeleteCustomJob(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/customJobs/"+id+"/")
}

func (s *ServiceImpl) CreateHostedCustomMetricTemplate(ctx context.Context, customJobID string, req *HostedCustomMetricTemplateRequest) (*HostedCustomMetricTemplate, error) {
	return Post[HostedCustomMetricTemplate](s.client, ctx, "/customJobs/"+customJobID+"/hostedCustomMetricTemplate/", req)
}

func (s *ServiceImpl) GetHostedCustomMetricTemplate(ctx context.Context, customJobID string) (*HostedCustomMetricTemplate, error) {
	return Get[HostedCustomMetricTemplate](s.client, ctx, "/customJobs/"+customJobID+"/hostedCustomMetricTemplate/")
}

func (s *ServiceImpl) UpdateHostedCustomMetricTemplate(ctx context.Context, customJobID string, req *HostedCustomMetricTemplateRequest) (*HostedCustomMetricTemplate, error) {
	return Patch[HostedCustomMetricTemplate](s.client, ctx, "/customJobs/"+customJobID+"/hostedCustomMetricTemplate/", req)
}

func (s *ServiceImpl) CreateCustomModel(ctx context.Context, req *CreateCustomModelRequest) (*CustomModel, error) {
	return Post[CustomModel](s.client, ctx, "/customModels/", req)
}

func (s *ServiceImpl) CreateCustomModelFromLLMBlueprint(ctx context.Context, req *CreateCustomModelFromLLMBlueprintRequest) (*CreateCustomModelVersionFromLLMBlueprintResponse, error) {
	return Post[CreateCustomModelVersionFromLLMBlueprintResponse](s.client, ctx, "/genai/customModelVersions/", req)
}

func (s *ServiceImpl) CreateCustomModelVersionCreateFromLatest(ctx context.Context, id string, req *CreateCustomModelVersionFromLatestRequest) (*CustomModelVersion, error) {
	return Patch[CustomModelVersion](s.client, ctx, "/customModels/"+id+"/versions/", req)
}

func (s *ServiceImpl) CreateCustomModelVersionFromFiles(ctx context.Context, id string, req *CreateCustomModelVersionFromFilesRequest) (*CustomModelVersion, error) {
	return uploadFilesFromBinaries[CustomModelVersion](s.client, ctx, "/customModels/"+id+"/versions/", http.MethodPatch, req.Files, map[string]string{"baseEnvironmentId": req.BaseEnvironmentID, "isMajorUpdate": "false"})
}

func (s *ServiceImpl) CreateCustomModelVersionFromRemoteRepository(ctx context.Context, id string, req *CreateCustomModelVersionFromRemoteRepositoryRequest) (*CustomModelVersion, string, error) {
	return ExecuteAndExpectStatus[CustomModelVersion](s.client, ctx, http.MethodPatch, "/customModels/"+id+"/versions/fromRepository/", req)
}

func (s *ServiceImpl) GetCustomModel(ctx context.Context, id string) (*CustomModel, error) {
	return Get[CustomModel](s.client, ctx, "/customModels/"+id+"/")
}

func (s *ServiceImpl) IsCustomModelReady(ctx context.Context, id string) (bool, error) {
	customModel, err := s.GetCustomModel(ctx, id)
	if err != nil {
		return false, err
	}
	return customModel.LatestVersion.ID != "", nil
}

func (s *ServiceImpl) UpdateCustomModel(ctx context.Context, id string, req *UpdateCustomModelRequest) (*CustomModel, error) {
	return Patch[CustomModel](s.client, ctx, "/customModels/"+id+"/", req)
}

func (s *ServiceImpl) DeleteCustomModel(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/customModels/"+id+"/")
}

func (s *ServiceImpl) ListCustomModels(ctx context.Context) ([]CustomModel, error) {
	return GetAllPages[CustomModel](s.client, ctx, "/customModels/", nil)
}

func (s *ServiceImpl) ListCustomModelVersions(ctx context.Context, id string) ([]CustomModelVersion, error) {
	return GetAllPages[CustomModelVersion](s.client, ctx, "/customModels/"+id+"/versions/", nil)
}

func (s *ServiceImpl) ListGuardTemplates(ctx context.Context) ([]GuardTemplate, error) {
	return GetAllPages[GuardTemplate](s.client, ctx, "/guardTemplates/", nil)
}

func (s *ServiceImpl) GetGuardConfigurationsForCustomModelVersion(ctx context.Context, id string) (*GuardConfigurationResponse, error) {
	return Get[GuardConfigurationResponse](s.client, ctx, "/guardConfigurations/?entityId="+id+"&entityType=customModelVersion")
}

func (s *ServiceImpl) GetOverallModerationConfigurationForCustomModelVersion(ctx context.Context, id string) (*OverallModerationConfiguration, error) {
	return Get[OverallModerationConfiguration](s.client, ctx, "/overallModerationConfiguration/?entityId="+id+"&entityType=customModelVersion")
}

func (s *ServiceImpl) CreateCustomModelVersionFromGuardConfigurations(ctx context.Context, id string, req *CreateCustomModelVersionFromGuardsConfigurationRequest) (*CreateCustomModelVersionFromGuardsConfigurationResponse, error) {
	return Post[CreateCustomModelVersionFromGuardsConfigurationResponse](s.client, ctx, "/guardConfigurations/toNewCustomModelVersion/", req)
}

func (s *ServiceImpl) CreateDependencyBuild(ctx context.Context, id string, versionID string) (*DependencyBuild, error) {
	return Post[DependencyBuild](s.client, ctx, "/customModels/"+id+"/versions/"+versionID+"/dependencyBuild/", map[string]string{})
}

func (s *ServiceImpl) GetDependencyBuild(ctx context.Context, id string, versionID string) (*DependencyBuild, error) {
	return Get[DependencyBuild](s.client, ctx, "/customModels/"+id+"/versions/"+versionID+"/dependencyBuild/")
}

func (s *ServiceImpl) CreateCustomModelLLMValidation(ctx context.Context, req *CreateCustomModelLLMValidationRequest) (*CustomModelLLMValidation, string, error) {
	return ExecuteAndExpectStatus[CustomModelLLMValidation](s.client, ctx, http.MethodPost, "/genai/customModelLLMValidations/", req)
}

func (s *ServiceImpl) GetCustomModelLLMValidation(ctx context.Context, id string) (*CustomModelLLMValidation, error) {
	return Get[CustomModelLLMValidation](s.client, ctx, "/genai/customModelLLMValidations/"+id+"/")
}

func (s *ServiceImpl) UpdateCustomModelLLMValidation(ctx context.Context, id string, req *UpdateCustomModelLLMValidationRequest) (*CustomModelLLMValidation, error) {
	return Patch[CustomModelLLMValidation](s.client, ctx, "/genai/customModelLLMValidations/"+id+"/", req)
}

func (s *ServiceImpl) DeleteCustomModelLLMValidation(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/genai/customModelLLMValidations/"+id+"/")
}

// Registered Model Service Implementation.
func (s *ServiceImpl) CreateRegisteredModelFromCustomModelVersion(ctx context.Context, req *CreateRegisteredModelFromCustomModelRequest) (*RegisteredModelVersion, error) {
	return Post[RegisteredModelVersion](s.client, ctx, "/modelPackages/fromCustomModelVersion/", req)
}

func (s *ServiceImpl) CreateRegisteredModelFromLeaderboard(ctx context.Context, req *CreateRegisteredModelFromLeaderboardRequest) (*RegisteredModelVersion, error) {
	return Post[RegisteredModelVersion](s.client, ctx, "/modelPackages/fromLeaderboard/", req)
}

func (s *ServiceImpl) UpdateRegisteredModelVersion(ctx context.Context, registeredModelId string, versionId string, req *UpdateRegisteredModelVersionRequest) (*RegisteredModelVersion, error) {
	return Patch[RegisteredModelVersion](s.client, ctx, "/registeredModels/"+registeredModelId+"/versions/"+versionId+"/", req)
}

func (s *ServiceImpl) GetRegisteredModelVersion(ctx context.Context, registeredModelId string, versionId string) (*RegisteredModelVersion, error) {
	return Get[RegisteredModelVersion](s.client, ctx, "/registeredModels/"+registeredModelId+"/versions/"+versionId+"/")
}

func (s *ServiceImpl) ListRegisteredModelVersions(ctx context.Context, id string) ([]RegisteredModelVersion, error) {
	return GetAllPages[RegisteredModelVersion](s.client, ctx, "/registeredModels/"+id+"/versions/", nil)
}

func (s *ServiceImpl) GetLatestRegisteredModelVersion(ctx context.Context, id string) (*RegisteredModelVersion, error) {
	registeredModel, err := s.GetRegisteredModel(ctx, id)
	if err != nil {
		return nil, err
	}

	registeredModelVersions, err := s.ListRegisteredModelVersions(ctx, id)
	if err != nil {
		return nil, err
	}

	for index := range registeredModelVersions {
		version := registeredModelVersions[index]
		if version.RegisteredModelVersion == registeredModel.LastVersionNum {
			return &version, nil
		}
	}

	return nil, errors.New("latest version not found")
}

func (s *ServiceImpl) IsRegisteredModelVersionReady(ctx context.Context, registeredModelId string, versionId string) (bool, error) {
	modelPackage, err := s.GetRegisteredModelVersion(ctx, registeredModelId, versionId)
	if err != nil {
		return false, err
	}

	return modelPackage.BuildStatus == "complete", nil
}

func (s *ServiceImpl) ListRegisteredModels(ctx context.Context, req *ListRegisteredModelsRequest) ([]RegisteredModel, error) {
	return GetAllPages[RegisteredModel](s.client, ctx, "/registeredModels/", req)
}

func (s *ServiceImpl) GetRegisteredModel(ctx context.Context, id string) (*RegisteredModel, error) {
	return Get[RegisteredModel](s.client, ctx, "/registeredModels/"+id+"/")
}

func (s *ServiceImpl) UpdateRegisteredModel(ctx context.Context, id string, req *UpdateRegisteredModelRequest) (*RegisteredModel, error) {
	return Patch[RegisteredModel](s.client, ctx, "/registeredModels/"+id+"/", req)
}

func (s *ServiceImpl) DeleteRegisteredModel(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/registeredModels/"+id+"/")
}

// Prediction Environment Service Implementation.
func (s *ServiceImpl) CreatePredictionEnvironment(ctx context.Context, req *PredictionEnvironmentRequest) (*PredictionEnvironment, error) {
	return Post[PredictionEnvironment](s.client, ctx, "/predictionEnvironments/", req)
}

func (s *ServiceImpl) GetPredictionEnvironment(ctx context.Context, id string) (*PredictionEnvironment, error) {
	return Get[PredictionEnvironment](s.client, ctx, "/predictionEnvironments/"+id+"/")
}

func (s *ServiceImpl) UpdatePredictionEnvironment(ctx context.Context, id string, req *PredictionEnvironmentRequest) (*PredictionEnvironment, error) {
	return Patch[PredictionEnvironment](s.client, ctx, "/predictionEnvironments/"+id+"/", req)
}

func (s *ServiceImpl) DeletePredictionEnvironment(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/predictionEnvironments/"+id+"/")
}

// Deployment Service Implementation.
func (s *ServiceImpl) CreateDeploymentFromModelPackage(ctx context.Context, req *CreateDeploymentFromModelPackageRequest) (*DeploymentCreateResponse, string, error) {
	return ExecuteAndExpectStatus[DeploymentCreateResponse](s.client, ctx, http.MethodPost, "/deployments/fromModelPackage/", req)
}

func (s *ServiceImpl) GetDeployment(ctx context.Context, id string) (*Deployment, error) {
	return Get[Deployment](s.client, ctx, "/deployments/"+id+"/")
}

func (s *ServiceImpl) UpdateDeployment(ctx context.Context, id string, req *UpdateDeploymentRequest) (*Deployment, error) {
	return Patch[Deployment](s.client, ctx, "/deployments/"+id+"/", req)
}

func (s *ServiceImpl) UpdateDeploymentSettings(ctx context.Context, id string, req *DeploymentSettings) (*DeploymentSettings, error) {
	return Patch[DeploymentSettings](s.client, ctx, "/deployments/"+id+"/settings/", req)
}

func (s *ServiceImpl) GetDeploymentSettings(ctx context.Context, id string) (*DeploymentSettings, error) {
	return Get[DeploymentSettings](s.client, ctx, "/deployments/"+id+"/settings/")
}

func (s *ServiceImpl) GetDeploymentChallengerReplaySettings(ctx context.Context, id string) (*DeploymentChallengerReplaySettings, error) {
	return Get[DeploymentChallengerReplaySettings](s.client, ctx, "/deployments/"+id+"/challengerReplaySettings/")
}

func (s *ServiceImpl) UpdateDeploymentChallengerReplaySettings(ctx context.Context, id string, req *DeploymentChallengerReplaySettings) (*DeploymentChallengerReplaySettings, error) {
	return Patch[DeploymentChallengerReplaySettings](s.client, ctx, "/deployments/"+id+"/challengerReplaySettings/", req)
}

func (s *ServiceImpl) GetDeploymentHealthSettings(ctx context.Context, id string) (*DeploymentHealthSettings, error) {
	return Get[DeploymentHealthSettings](s.client, ctx, "/deployments/"+id+"/healthSettings/")
}

func (s *ServiceImpl) UpdateDeploymentHealthSettings(ctx context.Context, id string, req *DeploymentHealthSettings) (*DeploymentHealthSettings, error) {
	return Patch[DeploymentHealthSettings](s.client, ctx, "/deployments/"+id+"/healthSettings/", req)
}

func (s *ServiceImpl) GetDeploymentFeatureCacheSettings(ctx context.Context, id string) (*DeploymentFeatureCacheSettings, error) {
	return Get[DeploymentFeatureCacheSettings](s.client, ctx, "/deployments/"+id+"/featureCache/")
}

func (s *ServiceImpl) UpdateDeploymentFeatureCacheSettings(ctx context.Context, id string, req *DeploymentFeatureCacheSettings) (*DeploymentFeatureCacheSettings, error) {
	return Patch[DeploymentFeatureCacheSettings](s.client, ctx, "/deployments/"+id+"/featureCache/", req)
}

func (s *ServiceImpl) GetDeploymentRetrainingSettings(ctx context.Context, deploymentID string) (*RetrainingSettingsRetrieve, error) {
	return Get[RetrainingSettingsRetrieve](s.client, ctx, fmt.Sprintf("/deployments/%s/retrainingSettings/", deploymentID))
}

func (s *ServiceImpl) UpdateDeploymentRetrainingSettings(ctx context.Context, deploymentID string, req *DeploymentRetrainingSettings) (*RetrainingSettings, error) {
	return Patch[RetrainingSettings](s.client, ctx, fmt.Sprintf("/deployments/%s/retrainingSettings/", deploymentID), req)
}

func (s *ServiceImpl) CreateRetrainingPolicy(ctx context.Context, deploymentID string, req *RetrainingPolicyRequest) (*RetrainingPolicy, error) {
	return Post[RetrainingPolicy](s.client, ctx, "/deployments/"+deploymentID+"/retrainingPolicies/", req)
}

func (s *ServiceImpl) GetRetrainingPolicy(ctx context.Context, deploymentID, id string) (*RetrainingPolicy, error) {
	return Get[RetrainingPolicy](s.client, ctx, "/deployments/"+deploymentID+"/retrainingPolicies/"+id+"/")
}

func (s *ServiceImpl) UpdateRetrainingPolicy(ctx context.Context, deploymentID, id string, req *RetrainingPolicyRequest) (*RetrainingPolicy, error) {
	return Patch[RetrainingPolicy](s.client, ctx, "/deployments/"+deploymentID+"/retrainingPolicies/"+id+"/", req)
}

func (s *ServiceImpl) DeleteRetrainingPolicy(ctx context.Context, deploymentID, id string) error {
	return Delete(s.client, ctx, "/deployments/"+deploymentID+"/retrainingPolicies/"+id+"/")
}

func (s *ServiceImpl) CreateNotificationChannel(ctx context.Context, req *CreateNotificationChannelRequest) (*NotificationChannel, error) {
	return Post[NotificationChannel](s.client, ctx, "/entityNotificationChannels/", req)
}

func (s *ServiceImpl) GetNotificationChannel(ctx context.Context, relatedEntityType, relatedEntityID, id string) (*NotificationChannel, error) {
	return Get[NotificationChannel](s.client, ctx, "/entityNotificationChannels/"+relatedEntityType+"/"+relatedEntityID+"/"+id+"/")
}

func (s *ServiceImpl) UpdateNotificationChannel(ctx context.Context, relatedEntityType, relatedEntityID, id string, req *UpdateNotificationChannelRequest) (*NotificationChannel, error) {
	return Put[NotificationChannel](s.client, ctx, "/entityNotificationChannels/"+relatedEntityType+"/"+relatedEntityID+"/"+id+"/", req)
}

func (s *ServiceImpl) DeleteNotificationChannel(ctx context.Context, relatedEntityType, relatedEntityID, id string) error {
	return Delete(s.client, ctx, "/entityNotificationChannels/"+relatedEntityType+"/"+relatedEntityID+"/"+id+"/")
}

func (s *ServiceImpl) CreateNotificationPolicy(ctx context.Context, req *CreateNotificationPolicyRequest) (*NotificationPolicy, error) {
	return Post[NotificationPolicy](s.client, ctx, "/entityNotificationPolicies/", req)
}

func (s *ServiceImpl) GetNotificationPolicy(ctx context.Context, relatedEntityType, relatedEntityID, id string) (*NotificationPolicy, error) {
	return Get[NotificationPolicy](s.client, ctx, "/entityNotificationPolicies/"+relatedEntityType+"/"+relatedEntityID+"/"+id+"/")
}

func (s *ServiceImpl) UpdateNotificationPolicy(ctx context.Context, relatedEntityType, relatedEntityID, id string, req *UpdateNotificationPolicyRequest) (*NotificationPolicy, error) {
	return Put[NotificationPolicy](s.client, ctx, "/entityNotificationPolicies/"+relatedEntityType+"/"+relatedEntityID+"/"+id+"/", req)
}

func (s *ServiceImpl) DeleteNotificationPolicy(ctx context.Context, relatedEntityType, relatedEntityID, id string) error {
	return Delete(s.client, ctx, "/entityNotificationPolicies/"+relatedEntityType+"/"+relatedEntityID+"/"+id+"/")
}

func (s *ServiceImpl) CreateCustomMetricFromJob(ctx context.Context, deploymentID string, req *CreateCustomMetricFromJobRequest) (*CustomMetric, error) {
	return Post[CustomMetric](s.client, ctx, "/deployments/"+deploymentID+"/customMetrics/fromCustomJob/", req)
}

func (s *ServiceImpl) CreateCustomMetric(ctx context.Context, deploymentID string, req *CreateCustomMetricRequest) (*CustomMetric, error) {
	return Post[CustomMetric](s.client, ctx, "/deployments/"+deploymentID+"/customMetrics/", req)
}

func (s *ServiceImpl) GetCustomMetric(ctx context.Context, deploymentID string, id string) (*CustomMetric, error) {
	return Get[CustomMetric](s.client, ctx, "/deployments/"+deploymentID+"/customMetrics/"+id+"/")
}

func (s *ServiceImpl) UpdateCustomMetric(ctx context.Context, deploymentID string, id string, req *UpdateCustomMetricRequest) (*CustomMetric, error) {
	return Patch[CustomMetric](s.client, ctx, "/deployments/"+deploymentID+"/customMetrics/"+id+"/", req)
}

func (s *ServiceImpl) DeleteCustomMetric(ctx context.Context, deploymentID string, id string) error {
	return Delete(s.client, ctx, "/deployments/"+deploymentID+"/customMetrics/"+id+"/")
}

func (s *ServiceImpl) DeleteDeployment(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/deployments/"+id+"/")
}

func (s *ServiceImpl) ValidateDeploymentModelReplacement(ctx context.Context, id string, req *ValidateDeployemntModelReplacementRequest) (*ValidateDeployemntModelReplacementResponse, error) {
	return Post[ValidateDeployemntModelReplacementResponse](s.client, ctx, "/deployments/"+id+"/model/validation/", req)
}

func (s *ServiceImpl) UpdateDeploymentRuntimeParameters(ctx context.Context, id string, req *UpdateDeploymentRuntimeParametersRequest) (*Deployment, error) {
	return Put[Deployment](s.client, ctx, "/deployments/"+id+"/runtimeParameters/", req)
}

func (s *ServiceImpl) ListDeploymentRuntimeParameters(ctx context.Context, id string) ([]RuntimeParameter, error) {
	return GetAllPages[RuntimeParameter](s.client, ctx, "/deployments/"+id+"/runtimeParameters/", nil)
}

func (s *ServiceImpl) DeactivateDeployment(ctx context.Context, id string) (*Deployment, error) {
	return Patch[Deployment](s.client, ctx, "/deployments/"+id+"/status/", &UpdateDeploymentStatusRequest{Status: "inactive"})
}

func (s *ServiceImpl) ActivateDeployment(ctx context.Context, id string) (*Deployment, error) {
	return Patch[Deployment](s.client, ctx, "/deployments/"+id+"/status/", &UpdateDeploymentStatusRequest{Status: "active"})
}

func (s *ServiceImpl) UpdateDeploymentModel(ctx context.Context, id string, req *UpdateDeploymentModelRequest) (*Deployment, string, error) {
	return ExecuteAndExpectStatus[Deployment](s.client, ctx, http.MethodPatch, "/deployments/"+id+"/model/", req)
}

// Batch Prediction Job Definition Service Implementation.
func (s *ServiceImpl) CreateBatchPredictionJobDefinition(ctx context.Context, req *BatchPredictionJobDefinitionRequest) (*BatchPredictionJobDefinition, error) {
	return Post[BatchPredictionJobDefinition](s.client, ctx, "/batchPredictionJobDefinitions/", req)
}

func (s *ServiceImpl) GetBatchPredictionJobDefinition(ctx context.Context, id string) (*BatchPredictionJobDefinition, error) {
	return Get[BatchPredictionJobDefinition](s.client, ctx, "/batchPredictionJobDefinitions/"+id+"/")
}

func (s *ServiceImpl) UpdateBatchPredictionJobDefinition(ctx context.Context, id string, req *BatchPredictionJobDefinitionRequest) (*BatchPredictionJobDefinition, error) {
	return Patch[BatchPredictionJobDefinition](s.client, ctx, "/batchPredictionJobDefinitions/"+id+"/", req)
}

func (s *ServiceImpl) DeleteBatchPredictionJobDefinition(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/batchPredictionJobDefinitions/"+id+"/")
}

// Application Service Implementation.
func (s *ServiceImpl) CreateApplicationSource(ctx context.Context) (*ApplicationSource, error) {
	return Post[ApplicationSource](s.client, ctx, "/customApplicationSources/", map[string]string{})
}

func (s *ServiceImpl) CreateApplicationSourceFromTemplate(ctx context.Context, req *CreateApplicationSourceFromTemplateRequest) (*ApplicationSource, error) {
	return Post[ApplicationSource](s.client, ctx, "/customApplicationSources/fromCustomTemplate/", req)
}

func (s *ServiceImpl) GetApplicationSource(ctx context.Context, id string) (*ApplicationSource, error) {
	return Get[ApplicationSource](s.client, ctx, "/customApplicationSources/"+id+"/")
}

func (s *ServiceImpl) UpdateApplicationSource(ctx context.Context, id string, req *UpdateApplicationSourceRequest) (*ApplicationSource, error) {
	return Patch[ApplicationSource](s.client, ctx, "/customApplicationSources/"+id+"/", req)
}

func (s *ServiceImpl) CreateApplicationSourceVersion(ctx context.Context, id string, req *CreateApplicationSourceVersionRequest) (*ApplicationSourceVersion, error) {
	return Post[ApplicationSourceVersion](s.client, ctx, "/customApplicationSources/"+id+"/versions/", req)
}

func (s *ServiceImpl) UpdateApplicationSourceVersion(ctx context.Context, id string, versionId string, req *UpdateApplicationSourceVersionRequest) (*ApplicationSourceVersion, error) {
	return Patch[ApplicationSourceVersion](s.client, ctx, "/customApplicationSources/"+id+"/versions/"+versionId+"/", req)
}

func (s *ServiceImpl) UpdateApplicationSourceVersionFiles(ctx context.Context, id string, versionId string, files []FileInfo) (*ApplicationSourceVersion, error) {
	return uploadFilesFromBinaries[ApplicationSourceVersion](s.client, ctx, "/customApplicationSources/"+id+"/versions/"+versionId+"/", http.MethodPatch, files, map[string]string{})
}

func (s *ServiceImpl) GetApplicationSourceVersion(ctx context.Context, id string, versionId string) (*ApplicationSourceVersion, error) {
	return Get[ApplicationSourceVersion](s.client, ctx, "/customApplicationSources/"+id+"/versions/"+versionId+"/")
}

func (s *ServiceImpl) DeleteApplicationSource(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/customApplicationSources/"+id+"/")
}

func (s *ServiceImpl) GetCustomTemplate(ctx context.Context, id string) (*CustomTemplate, error) {
	return Get[CustomTemplate](s.client, ctx, "/customTemplates/"+id+"/")
}

func (s *ServiceImpl) GetCustomTemplateFile(ctx context.Context, customTemplateID, fileID string) (*CustomTemplateFile, error) {
	return Get[CustomTemplateFile](s.client, ctx, "/customTemplates/"+customTemplateID+"/files/"+fileID+"/")
}

func (s *ServiceImpl) CreateCustomApplication(ctx context.Context, req *CreateCustomApplicationeRequest) (*Application, error) {
	return Post[Application](s.client, ctx, "/customApplications/", req)
}

func (s *ServiceImpl) CreateQAApplication(ctx context.Context, req *CreateQAApplicationRequest) (*Application, error) {
	return Post[Application](s.client, ctx, "/customApplications/qanda/", req)
}

func (s *ServiceImpl) GetApplication(ctx context.Context, id string) (*Application, error) {
	return Get[Application](s.client, ctx, "/customApplications/"+id+"/")
}

func (s *ServiceImpl) UpdateApplication(ctx context.Context, id string, req *UpdateApplicationRequest) (*Application, error) {
	return Patch[Application](s.client, ctx, "/customApplications/"+id+"/", req)
}

func (s *ServiceImpl) DeleteApplication(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/customApplications/"+id+"/")
}

// Credentials Service Implementation.
func (s *ServiceImpl) CreateCredential(ctx context.Context, req *CredentialRequest) (*Credential, error) {
	return Post[Credential](s.client, ctx, "/credentials/", req)
}

func (s *ServiceImpl) GetCredential(ctx context.Context, id string) (*Credential, error) {
	return Get[Credential](s.client, ctx, "/credentials/"+id+"/")
}

func (s *ServiceImpl) UpdateCredential(ctx context.Context, id string, req *CredentialRequest) (*Credential, error) {
	return Patch[Credential](s.client, ctx, "/credentials/"+id+"/", req)
}

func (s *ServiceImpl) DeleteCredential(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/credentials/"+id+"/")
}

func (s *ServiceImpl) ListCredentials(ctx context.Context) ([]Credential, error) {
	return GetAllPages[Credential](s.client, ctx, "/credentials/", nil)
}

// Execution Environment Service Implementation.
func (s *ServiceImpl) CreateExecutionEnvironment(ctx context.Context, req *CreateExecutionEnvironmentRequest) (*ExecutionEnvironment, error) {
	return Post[ExecutionEnvironment](s.client, ctx, "/executionEnvironments/", req)
}

func (s *ServiceImpl) GetExecutionEnvironment(ctx context.Context, id string) (*ExecutionEnvironment, error) {
	return Get[ExecutionEnvironment](s.client, ctx, "/executionEnvironments/"+id+"/")
}

func (s *ServiceImpl) UpdateExecutionEnvironment(ctx context.Context, id string, req *UpdateExecutionEnvironmentRequest) (*ExecutionEnvironment, error) {
	return Patch[ExecutionEnvironment](s.client, ctx, "/executionEnvironments/"+id+"/", req)
}

func (s *ServiceImpl) DeleteExecutionEnvironment(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/executionEnvironments/"+id+"/")
}

func (s *ServiceImpl) ListExecutionEnvironments(ctx context.Context) ([]ExecutionEnvironment, error) {
	return GetAllPages[ExecutionEnvironment](s.client, ctx, "/executionEnvironments/", nil)
}

func (s *ServiceImpl) CreateExecutionEnvironmentVersion(ctx context.Context, id string, req *CreateExecutionEnvironmentVersionRequest) (*ExecutionEnvironmentVersion, error) {
	return uploadFilesFromBinaries[ExecutionEnvironmentVersion](s.client, ctx, "/executionEnvironments/"+id+"/versions/", http.MethodPost, req.Files, map[string]string{"description": req.Description})
}

func (s *ServiceImpl) GetExecutionEnvironmentVersion(ctx context.Context, id, versionId string) (*ExecutionEnvironmentVersion, error) {
	return Get[ExecutionEnvironmentVersion](s.client, ctx, "/executionEnvironments/"+id+"/versions/"+versionId+"/")
}

func (s *ServiceImpl) GetTaskStatus(ctx context.Context, id string) (*TaskStatusResponse, error) {
	return Get[TaskStatusResponse](s.client, ctx, "/status/"+id+"/")
}

func (s *ServiceImpl) GetGenAITaskStatus(ctx context.Context, id string) (*TaskStatusResponse, error) {
	return Get[TaskStatusResponse](s.client, ctx, "/genai/status/"+id+"/")
}

func (s *ServiceImpl) GetUserInfo(ctx context.Context) (*UserInfo, error) {
	return Get[UserInfo](s.client, ctx, "/account/info/")
}

// API Gateway Service Implementations

// Notebook Service Implementation.
func (s *ServiceImpl) ImportNotebookFromFile(ctx context.Context, fileName string, content []byte, useCaseID string) (*ImportNotebookResponse, error) {
	extraFields := map[string]string{}
	if useCaseID != "" {
		extraFields["useCaseId"] = useCaseID
	}
	importNotebookResponse, err := uploadFileFromBinary[ImportNotebookResponse](s.apiGWClient, ctx, "/nbx/notebookImport/fromFile/", http.MethodPost, fileName, content, extraFields)
	if err != nil {
		return nil, err
	}
	importNotebookResponse.URL = URLForNotebook(importNotebookResponse.ID, useCaseID, s.apiGWClient.cfg.BaseURL())

	return importNotebookResponse, nil
}

func (s *ServiceImpl) GetNotebook(ctx context.Context, id string) (*Notebook, error) {
	notebookResponse, err := Get[Notebook](s.apiGWClient, ctx, fmt.Sprintf("/nbx/notebooks/%s/", id))
	if err != nil {
		return nil, err
	}
	notebookResponse.URL = URLForNotebook(notebookResponse.ID, notebookResponse.UseCaseID, s.apiGWClient.cfg.BaseURL())
	return notebookResponse, nil
}

func (s *ServiceImpl) UpdateNotebook(ctx context.Context, id string, useCaseID string) (*Notebook, error) {
	notebookResponse, err := Patch[Notebook](s.apiGWClient, ctx, fmt.Sprintf("/nbx/notebooks/%s/", id), map[string]string{"useCaseId": useCaseID})
	if err != nil {
		return nil, err
	}
	notebookResponse.URL = URLForNotebook(notebookResponse.ID, notebookResponse.UseCaseID, s.apiGWClient.cfg.BaseURL())
	return notebookResponse, nil
}

func (s *ServiceImpl) DeleteNotebook(ctx context.Context, id string) error {
	return Delete(s.apiGWClient, ctx, fmt.Sprintf("/nbx/notebooks/%s/", id))
}
