package client

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
)

type Service interface {
	// Use Case
	CreateUseCase(ctx context.Context, req *UseCaseRequest) (*CreateUseCaseResponse, error)
	GetUseCase(ctx context.Context, id string) (*UseCaseResponse, error)
	UpdateUseCase(ctx context.Context, id string, req *UseCaseRequest) (*UseCaseResponse, error)
	DeleteUseCase(ctx context.Context, id string) error

	// Remote Repository
	CreateRemoteRepository(ctx context.Context, req *CreateRemoteRepositoryRequest) (*RemoteRepositoryResponse, error)
	GetRemoteRepository(ctx context.Context, id string) (*RemoteRepositoryResponse, error)
	UpdateRemoteRepository(ctx context.Context, id string, req *UpdateRemoteRepositoryRequest) (*RemoteRepositoryResponse, error)
	DeleteRemoteRepository(ctx context.Context, id string) error

	// Data Set
	CreateDataset(ctx context.Context, req *CreateDatasetRequest) (*CreateDatasetResponse, error)
	CreateDatasetFromFile(ctx context.Context, fileName string, content []byte) (*CreateDatasetVersionResponse, error)
	CreateDatasetVersionFromFile(ctx context.Context, id string, fileName string, content []byte) (*CreateDatasetVersionResponse, error)
	GetDataset(ctx context.Context, id string) (*DatasetResponse, error)
	IsDatasetReady(ctx context.Context, id string) (bool, error)
	UpdateDataset(ctx context.Context, id string, req *UpdateDatasetRequest) (*DatasetResponse, error)
	DeleteDataset(ctx context.Context, id string) error
	LinkDatasetToUseCase(ctx context.Context, useCaseID, datasetID string) error

	// Vector Database
	// try out vscode commit
	CreateVectorDatabase(ctx context.Context, req *CreateVectorDatabaseRequest) (*CreateVectorDatabaseResponse, error)
	GetVectorDatabase(ctx context.Context, id string) (*VectorDatabaseResponse, error)
	UpdateVectorDatabase(ctx context.Context, id string, req *UpdateVectorDatabaseRequest) (*VectorDatabaseResponse, error)
	DeleteVectorDatabase(ctx context.Context, id string) error
	IsVectorDatabaseReady(ctx context.Context, id string) (bool, error)
	IsDatasetReadyForVectorDatabase(ctx context.Context, id string) (bool, error)

	// Playground
	CreatePlayground(ctx context.Context, req *CreatePlaygroundRequest) (*CreatePlaygroundResponse, error)
	GetPlayground(ctx context.Context, id string) (*PlaygroundResponse, error)
	UpdatePlayground(ctx context.Context, id string, req *UpdatePlaygroundRequest) (*PlaygroundResponse, error)
	DeletePlayground(ctx context.Context, id string) error

	// LLM Blueprint
	CreateLLMBlueprint(ctx context.Context, req *CreateLLMBlueprintRequest) (*LLMBlueprintResponse, error)
	GetLLMBlueprint(ctx context.Context, id string) (*LLMBlueprintResponse, error)
	UpdateLLMBlueprint(ctx context.Context, id string, req *UpdateLLMBlueprintRequest) (*LLMBlueprintResponse, error)
	DeleteLLMBlueprint(ctx context.Context, id string) error
	ListLLMs(ctx context.Context) (*ListLLMsResponse, error)

	// Custom Model
	CreateCustomModel(ctx context.Context, req *CreateCustomModelRequest) (*CustomModelResponse, error)
	CreateCustomModelFromLLMBlueprint(ctx context.Context, req *CreateCustomModelFromLLMBlueprintRequest) (*CreateCustomModelVersionFromLLMBlueprintResponse, error)
	CreateCustomModelVersionCreateFromLatest(ctxc context.Context, id string, req *CreateCustomModelVersionCreateFromLatestRequest) (*CreateCustomModelVersionResponse, error)
	CreateCustomModelVersionFromFiles(ctx context.Context, id string, req *CreateCustomModelVersionFromFilesRequest) (*CreateCustomModelVersionResponse, error)
	CreateCustomModelVersionFromRemoteRepository(ctx context.Context, id string, req *CreateCustomModelVersionFromRemoteRepositoryRequest) (*CreateCustomModelVersionResponse, error)
	GetCustomModel(ctx context.Context, id string) (*CustomModelResponse, error)
	IsCustomModelReady(ctx context.Context, id string) (bool, error)
	UpdateCustomModel(ctx context.Context, id string, req *CustomModelUpdate) (*CustomModelResponse, error)
	DeleteCustomModel(ctx context.Context, id string) error
	ListExecutionEnvironments(ctx context.Context) (*ListExecutionEnvironmentsResponse, error)
	ListGuardTemplates(ctx context.Context) (*ListGuardTemplatesResponse, error)
	GetGuardConfigurationsForCustomModelVersion(ctx context.Context, id string) (*GuardConfigurationResponse, error)
	GetOverallModerationConfigurationForCustomModelVersion(ctx context.Context, id string) (*OverallModerationConfiguration, error)
	CreateCustomModelVersionFromGuardConfigurations(ctx context.Context, id string, req *CreateCustomModelVersionFromGuardsConfigurationRequest) (*CreateCustomModelVersionFromGuardsConfigurationResponse, error)

	// Registered Model
	CreateRegisteredModelFromCustomModelVersion(ctx context.Context, req *CreateRegisteredModelFromCustomModelRequest) (*RegisteredModelVersionResponse, error)
	ListRegisteredModelVersions(ctx context.Context, id string) (*ListRegisteredModelVersionsResponse, error)
	GetLatestRegisteredModelVersion(ctx context.Context, id string) (*RegisteredModelVersionResponse, error)
	GetRegisteredModelVersion(ctx context.Context, registeredModelId string, versionId string) (*RegisteredModelVersionResponse, error)
	IsRegisteredModelVersionReady(ctx context.Context, registeredModelId string, versionId string) (bool, error)
	ListRegisteredModels(ctx context.Context) (*ListRegisteredModelsResponse, error)
	GetRegisteredModel(ctx context.Context, id string) (*RegisteredModelResponse, error)
	UpdateRegisteredModel(ctx context.Context, id string, req *RegisteredModelUpdate) (*RegisteredModelResponse, error)
	DeleteRegisteredModel(ctx context.Context, id string) error

	// Prediction Environment
	CreatePredictionEnvironment(ctx context.Context, req *CreatePredictionEnvironmentRequest) (*PredictionEnvironment, error)
	GetPredictionEnvironment(ctx context.Context, id string) (*PredictionEnvironment, error)
	UpdatePredictionEnvironment(ctx context.Context, id string, req *UpdatePredictionEnvironmentRequest) (*PredictionEnvironment, error)
	DeletePredictionEnvironment(ctx context.Context, id string) error

	// Deployment
	CreateDeploymentFromModelPackage(ctx context.Context, req *CreateDeploymentFromModelPackageRequest) (*DeploymentCreateResponse, error)
	GetDeployment(ctx context.Context, id string) (*DeploymentRetrieveResponse, error)
	UpdateDeployment(ctx context.Context, id string, req *UpdateDeploymentRequest) (*DeploymentRetrieveResponse, error)
	UpdateDeploymentSettings(ctx context.Context, id string, req *DeploymentSettings) (*DeploymentSettings, error)
	GetDeploymentSettings(ctx context.Context, id string) (*DeploymentSettings, error)
	DeleteDeployment(ctx context.Context, id string) error
	IsDeploymentReady(ctx context.Context, id string) (bool, error)
	ValidateDeploymentModelReplacement(ctx context.Context, id string, req *ValidateDeployemntModelReplacementRequest) (*ValidateDeployemntModelReplacementResponse, error)
	UpdateDeploymentModel(ctx context.Context, id string, req *UpdateDeploymentModelRequest) (*DeploymentRetrieveResponse, error)

	// Application Source
	GetChatApplicationSource(ctx context.Context, id string) (*ChatApplicationSourceResponse, error)
	DeleteChatApplicationSource(ctx context.Context, id string) error

	// Application
	CreateChatApplication(ctx context.Context, req *CreateChatApplicationRequest) (*ChatApplicationResponse, error)
	GetChatApplication(ctx context.Context, id string) (*ChatApplicationResponse, error)
	IsChatApplicationReady(ctx context.Context, id string) (bool, error)
	UpdateChatApplication(ctx context.Context, id string, req *UpdateChatApplicationRequest) (*ChatApplicationResponse, error)
	DeleteChatApplication(ctx context.Context, id string) error

	// Credential
	CreateCredential(ctx context.Context, req *CredentialRequest) (*CredentialResponse, error)
	GetCredential(ctx context.Context, id string) (*CredentialResponse, error)
	UpdateCredential(ctx context.Context, id string, req *CredentialRequest) (*CredentialResponse, error)
	DeleteCredential(ctx context.Context, id string) error

	// Async Tasks
	GetTaskStatus(ctx context.Context, id string) (*TaskStatusResponse, error)
	WaitForTaskStatusToComplete(ctx context.Context, id string) error

	// Add your service methods here
}

// Service for the DataRobot API.
type ServiceImpl struct {
	client *Client
}

// NewService creates a new API service.
func NewService(c *Client) Service {
	if c == nil {
		panic("client is required")
	}
	return &ServiceImpl{client: c}
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

func (s *ServiceImpl) CreateDatasetVersionFromFile(ctx context.Context, id string, fileName string, content []byte) (*CreateDatasetVersionResponse, error) {
	return uploadFileFromBinary[CreateDatasetVersionResponse](s.client, ctx, "/datasets/"+id+"/versions/fromFile/", http.MethodPost, fileName, content, map[string]string{})
}

func (s *ServiceImpl) GetDataset(ctx context.Context, id string) (*DatasetResponse, error) {
	return Get[DatasetResponse](s.client, ctx, "/datasets/"+id)
}

func (s *ServiceImpl) UpdateDataset(ctx context.Context, id string, req *UpdateDatasetRequest) (*DatasetResponse, error) {
	return Patch[DatasetResponse](s.client, ctx, "/datasets/"+id, req)
}

func (s *ServiceImpl) DeleteDataset(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/datasets/"+id)
}

func (s *ServiceImpl) IsDatasetReady(ctx context.Context, id string) (bool, error) {
	dataset, err := s.GetDataset(ctx, id)
	if err != nil {
		return false, err
	}
	if dataset.Status == "ERROR" {
		return false, NewGenericError("Dataset execution failed")
	}
	return dataset.Status == "COMPLETED", nil
}

func (s *ServiceImpl) IsDatasetReadyForVectorDatabase(ctx context.Context, id string) (bool, error) {
	dataset, err := s.GetDataset(ctx, id)
	if err != nil {
		return false, err
	}
	if dataset.Status == "ERROR" {
		return false, NewGenericError("Dataset execution failed")
	}
	if dataset.Status == "COMPLETED" {
		return dataset.IsVectorDatabaseEligible, nil
	}
	return false, nil
}

// Use Case Service Implementation.
func (s *ServiceImpl) LinkDatasetToUseCase(ctx context.Context, useCaseID, datasetID string) error {
	_, err := Post[CreateVoidResponse](s.client, ctx, "/useCases/"+useCaseID+"/datasets/"+datasetID, &CreateVoidRequest{})
	return err
}

func (s *ServiceImpl) CreateUseCase(ctx context.Context, req *UseCaseRequest) (resp *CreateUseCaseResponse, err error) {
	return Post[CreateUseCaseResponse](s.client, ctx, "/useCases/", req)
}

func (s *ServiceImpl) GetUseCase(ctx context.Context, id string) (*UseCaseResponse, error) {
	return Get[UseCaseResponse](s.client, ctx, "/useCases/"+id)
}

func (s *ServiceImpl) UpdateUseCase(ctx context.Context, id string, req *UseCaseRequest) (*UseCaseResponse, error) {
	return Patch[UseCaseResponse](s.client, ctx, "/useCases/"+id, req)
}

func (s *ServiceImpl) DeleteUseCase(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/useCases/"+id)
}

// Remote Repository Service Implementation.
func (s *ServiceImpl) CreateRemoteRepository(ctx context.Context, req *CreateRemoteRepositoryRequest) (*RemoteRepositoryResponse, error) {
	return Post[RemoteRepositoryResponse](s.client, ctx, "/remoteRepositories/", req)
}

func (s *ServiceImpl) GetRemoteRepository(ctx context.Context, id string) (*RemoteRepositoryResponse, error) {
	return Get[RemoteRepositoryResponse](s.client, ctx, "/remoteRepositories/"+id)
}

func (s *ServiceImpl) UpdateRemoteRepository(ctx context.Context, id string, req *UpdateRemoteRepositoryRequest) (*RemoteRepositoryResponse, error) {
	return Patch[RemoteRepositoryResponse](s.client, ctx, "/remoteRepositories/"+id, req)
}

func (s *ServiceImpl) DeleteRemoteRepository(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/remoteRepositories/"+id)
}

// Vector Database Service Implementation.
func (s *ServiceImpl) CreateVectorDatabase(ctx context.Context, req *CreateVectorDatabaseRequest) (*CreateVectorDatabaseResponse, error) {
	return Post[CreateVectorDatabaseResponse](s.client, ctx, "/genai/vectorDatabases/", req)
}

func (s *ServiceImpl) GetVectorDatabase(ctx context.Context, id string) (*VectorDatabaseResponse, error) {
	return Get[VectorDatabaseResponse](s.client, ctx, "/genai/vectorDatabases/"+id+"/")
}

func (s *ServiceImpl) UpdateVectorDatabase(ctx context.Context, id string, req *UpdateVectorDatabaseRequest) (*VectorDatabaseResponse, error) {
	return Patch[VectorDatabaseResponse](s.client, ctx, "/genai/vectorDatabases/"+id+"/", req)
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

func (s *ServiceImpl) CreateLLMBlueprint(ctx context.Context, req *CreateLLMBlueprintRequest) (*LLMBlueprintResponse, error) {
	return Post[LLMBlueprintResponse](s.client, ctx, "/genai/llmBlueprints/", req)
}

func (s *ServiceImpl) GetLLMBlueprint(ctx context.Context, id string) (*LLMBlueprintResponse, error) {
	return Get[LLMBlueprintResponse](s.client, ctx, "/genai/llmBlueprints/"+id+"/")
}

func (s *ServiceImpl) UpdateLLMBlueprint(ctx context.Context, id string, req *UpdateLLMBlueprintRequest) (*LLMBlueprintResponse, error) {
	return Patch[LLMBlueprintResponse](s.client, ctx, "/genai/llmBlueprints/"+id+"/", req)
}

func (s *ServiceImpl) DeleteLLMBlueprint(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/genai/llmBlueprints/"+id+"/")
}

func (s *ServiceImpl) ListLLMs(ctx context.Context) (*ListLLMsResponse, error) {
	return Get[ListLLMsResponse](s.client, ctx, "/genai/llms/")
}

func (s *ServiceImpl) CreateCustomModel(ctx context.Context, req *CreateCustomModelRequest) (*CustomModelResponse, error) {
	return Post[CustomModelResponse](s.client, ctx, "/customModels/", req)
}

func (s *ServiceImpl) CreateCustomModelFromLLMBlueprint(ctx context.Context, req *CreateCustomModelFromLLMBlueprintRequest) (*CreateCustomModelVersionFromLLMBlueprintResponse, error) {
	return Post[CreateCustomModelVersionFromLLMBlueprintResponse](s.client, ctx, "/genai/customModelVersions/", req)
}

func (s *ServiceImpl) CreateCustomModelVersionCreateFromLatest(ctx context.Context, id string, req *CreateCustomModelVersionCreateFromLatestRequest) (*CreateCustomModelVersionResponse, error) {
	return Patch[CreateCustomModelVersionResponse](s.client, ctx, "/customModels/"+id+"/versions/", req)
}

func (s *ServiceImpl) CreateCustomModelVersionFromFiles(ctx context.Context, id string, req *CreateCustomModelVersionFromFilesRequest) (*CreateCustomModelVersionResponse, error) {
	return uploadFilesFromBinaries[CreateCustomModelVersionResponse](s.client, ctx, "/customModels/"+id+"/versions/", http.MethodPatch, req.Files, map[string]string{"baseEnvironmentId": req.BaseEnvironmentID, "isMajorUpdate": "false"})
}

func (s *ServiceImpl) CreateCustomModelVersionFromRemoteRepository(ctx context.Context, id string, req *CreateCustomModelVersionFromRemoteRepositoryRequest) (*CreateCustomModelVersionResponse, error) {
	return Patch[CreateCustomModelVersionResponse](s.client, ctx, "/customModels/"+id+"/versions/fromRepository/", req)
}

func (s *ServiceImpl) GetCustomModel(ctx context.Context, id string) (*CustomModelResponse, error) {
	return Get[CustomModelResponse](s.client, ctx, "/customModels/"+id+"/")
}

func (s *ServiceImpl) IsCustomModelReady(ctx context.Context, id string) (bool, error) {
	customModel, err := s.GetCustomModel(ctx, id)
	if err != nil {
		return false, err
	}
	return customModel.LatestVersion.ID != "", nil
}

func (s *ServiceImpl) UpdateCustomModel(ctx context.Context, id string, req *CustomModelUpdate) (*CustomModelResponse, error) {
	return Patch[CustomModelResponse](s.client, ctx, "/customModels/"+id+"/", req)
}

func (s *ServiceImpl) DeleteCustomModel(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/customModels/"+id+"/")
}

func (s *ServiceImpl) ListExecutionEnvironments(ctx context.Context) (*ListExecutionEnvironmentsResponse, error) {
	return Get[ListExecutionEnvironmentsResponse](s.client, ctx, "/executionEnvironments/")
}

func (s *ServiceImpl) ListGuardTemplates(ctx context.Context) (*ListGuardTemplatesResponse, error) {
	return Get[ListGuardTemplatesResponse](s.client, ctx, "/guardTemplates/")
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

// Registered Model Service Implementation.
func (s *ServiceImpl) CreateRegisteredModelFromCustomModelVersion(ctx context.Context, req *CreateRegisteredModelFromCustomModelRequest) (*RegisteredModelVersionResponse, error) {
	return Post[RegisteredModelVersionResponse](s.client, ctx, "/modelPackages/fromCustomModelVersion/", req)
}

func (s *ServiceImpl) GetRegisteredModelVersion(ctx context.Context, registeredModelId string, versionId string) (*RegisteredModelVersionResponse, error) {
	return Get[RegisteredModelVersionResponse](s.client, ctx, "/registeredModels/"+registeredModelId+"/versions/"+versionId+"/")
}

func (s *ServiceImpl) ListRegisteredModelVersions(ctx context.Context, id string) (*ListRegisteredModelVersionsResponse, error) {
	return Get[ListRegisteredModelVersionsResponse](s.client, ctx, "/registeredModels/"+id+"/versions/")
}

func (s *ServiceImpl) GetLatestRegisteredModelVersion(ctx context.Context, id string) (*RegisteredModelVersionResponse, error) {
	registeredModel, err := s.GetRegisteredModel(ctx, id)
	if err != nil {
		return nil, err
	}

	registeredModelVersions, err := s.ListRegisteredModelVersions(ctx, id)
	if err != nil {
		return nil, err
	}

	for index := range registeredModelVersions.Data {
		version := registeredModelVersions.Data[index]
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

func (s *ServiceImpl) ListRegisteredModels(ctx context.Context) (*ListRegisteredModelsResponse, error) {
	return Get[ListRegisteredModelsResponse](s.client, ctx, "/registeredModels/")
}

func (s *ServiceImpl) GetRegisteredModel(ctx context.Context, id string) (*RegisteredModelResponse, error) {
	return Get[RegisteredModelResponse](s.client, ctx, "/registeredModels/"+id+"/")
}

func (s *ServiceImpl) UpdateRegisteredModel(ctx context.Context, id string, req *RegisteredModelUpdate) (*RegisteredModelResponse, error) {
	return Patch[RegisteredModelResponse](s.client, ctx, "/registeredModels/"+id+"/", req)
}

func (s *ServiceImpl) DeleteRegisteredModel(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/registeredModels/"+id+"/")
}

// Prediction Environment Service Implementation.
func (s *ServiceImpl) CreatePredictionEnvironment(ctx context.Context, req *CreatePredictionEnvironmentRequest) (*PredictionEnvironment, error) {
	return Post[PredictionEnvironment](s.client, ctx, "/predictionEnvironments/", req)
}

func (s *ServiceImpl) GetPredictionEnvironment(ctx context.Context, id string) (*PredictionEnvironment, error) {
	return Get[PredictionEnvironment](s.client, ctx, "/predictionEnvironments/"+id+"/")
}

func (s *ServiceImpl) UpdatePredictionEnvironment(ctx context.Context, id string, req *UpdatePredictionEnvironmentRequest) (*PredictionEnvironment, error) {
	return Patch[PredictionEnvironment](s.client, ctx, "/predictionEnvironments/"+id+"/", req)
}

func (s *ServiceImpl) DeletePredictionEnvironment(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/predictionEnvironments/"+id+"/")
}

// Deployment Service Implementation.
func (s *ServiceImpl) CreateDeploymentFromModelPackage(ctx context.Context, req *CreateDeploymentFromModelPackageRequest) (*DeploymentCreateResponse, error) {
	return Post[DeploymentCreateResponse](s.client, ctx, "/deployments/fromModelPackage/", req)
}

func (s *ServiceImpl) GetDeployment(ctx context.Context, id string) (*DeploymentRetrieveResponse, error) {
	return Get[DeploymentRetrieveResponse](s.client, ctx, "/deployments/"+id+"/")
}

func (s *ServiceImpl) UpdateDeployment(ctx context.Context, id string, req *UpdateDeploymentRequest) (*DeploymentRetrieveResponse, error) {
	return Patch[DeploymentRetrieveResponse](s.client, ctx, "/deployments/"+id+"/", req)
}

func (s *ServiceImpl) UpdateDeploymentSettings(ctx context.Context, id string, req *DeploymentSettings) (*DeploymentSettings, error) {
	return Patch[DeploymentSettings](s.client, ctx, "/deployments/"+id+"/settings/", req)
}

func (s *ServiceImpl) GetDeploymentSettings(ctx context.Context, id string) (*DeploymentSettings, error) {
	return Get[DeploymentSettings](s.client, ctx, "/deployments/"+id+"/settings/")
}

func (s *ServiceImpl) DeleteDeployment(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/deployments/"+id+"/")
}

func (s *ServiceImpl) CreateChatApplication(ctx context.Context, req *CreateChatApplicationRequest) (*ChatApplicationResponse, error) {
	return Post[ChatApplicationResponse](s.client, ctx, "/customApplications/qanda", req)
}

func (s *ServiceImpl) IsDeploymentReady(ctx context.Context, id string) (bool, error) {
	deployment, err := s.GetDeployment(ctx, id)
	if err != nil {
		return false, err
	}
	return deployment.Status == "active", nil
}

func (s *ServiceImpl) ValidateDeploymentModelReplacement(ctx context.Context, id string, req *ValidateDeployemntModelReplacementRequest) (*ValidateDeployemntModelReplacementResponse, error) {
	return Post[ValidateDeployemntModelReplacementResponse](s.client, ctx, "/deployments/"+id+"/model/validation", req)
}

func (s *ServiceImpl) UpdateDeploymentModel(ctx context.Context, id string, req *UpdateDeploymentModelRequest) (*DeploymentRetrieveResponse, error) {
	return Patch[DeploymentRetrieveResponse](s.client, ctx, "/deployments/"+id+"/model", req)
}

// Application Source Service Implementation.
func (s *ServiceImpl) GetChatApplicationSource(ctx context.Context, id string) (*ChatApplicationSourceResponse, error) {
	return Get[ChatApplicationSourceResponse](s.client, ctx, "/customApplicationSources/"+id+"/")
}

func (s *ServiceImpl) DeleteChatApplicationSource(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/customApplicationSources/"+id+"/")
}

// Application Service Implementation.
func (s *ServiceImpl) GetChatApplication(ctx context.Context, id string) (*ChatApplicationResponse, error) {
	return Get[ChatApplicationResponse](s.client, ctx, "/customApplications/"+id+"/")
}

func (s *ServiceImpl) IsChatApplicationReady(ctx context.Context, id string) (bool, error) {
	customApplication, err := s.GetChatApplication(ctx, id)
	if err != nil {
		return false, err
	}

	return customApplication.Status == "running", nil
}

func (s *ServiceImpl) UpdateChatApplication(ctx context.Context, id string, req *UpdateChatApplicationRequest) (*ChatApplicationResponse, error) {
	return Patch[ChatApplicationResponse](s.client, ctx, "/customApplications/"+id+"/", req)
}

func (s *ServiceImpl) DeleteChatApplication(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/customApplications/"+id+"/")
}

// Credentials Service Implementation.
func (s *ServiceImpl) CreateCredential(ctx context.Context, req *CredentialRequest) (*CredentialResponse, error) {
	return Post[CredentialResponse](s.client, ctx, "/credentials/", req)
}

func (s *ServiceImpl) GetCredential(ctx context.Context, id string) (*CredentialResponse, error) {
	return Get[CredentialResponse](s.client, ctx, "/credentials/"+id+"/")
}

func (s *ServiceImpl) UpdateCredential(ctx context.Context, id string, req *CredentialRequest) (*CredentialResponse, error) {
	return Patch[CredentialResponse](s.client, ctx, "/credentials/"+id+"/", req)
}

func (s *ServiceImpl) DeleteCredential(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/credentials/"+id+"/")
}

func (s *ServiceImpl) GetTaskStatus(ctx context.Context, id string) (*TaskStatusResponse, error) {
	return Get[TaskStatusResponse](s.client, ctx, "/status/"+id+"/")
}

func (s *ServiceImpl) WaitForTaskStatusToComplete(ctx context.Context, id string) error {
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = 1 * time.Second
	expBackoff.MaxInterval = 30 * time.Second
	expBackoff.MaxElapsedTime = 20 * time.Minute

	operation := func() error {
		task, err := s.GetTaskStatus(ctx, id)
		if err != nil {
			return backoff.Permanent(err)
		}
		if task.Status != "COMPLETED" {
			return errors.New("task is not completed")
		}
		return nil
	}

	// Retry the operation using the backoff strategy
	return backoff.Retry(operation, expBackoff)
}
