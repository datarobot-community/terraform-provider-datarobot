package client

import (
	"context"
	"errors"
	"net/http"
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
	ListLLMs(ctx context.Context) (*ListLLMsResponse, error)

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
	ListExecutionEnvironments(ctx context.Context) (*ListExecutionEnvironmentsResponse, error)
	ListGuardTemplates(ctx context.Context) (*ListGuardTemplatesResponse, error)
	GetGuardConfigurationsForCustomModelVersion(ctx context.Context, id string) (*GuardConfigurationResponse, error)
	GetOverallModerationConfigurationForCustomModelVersion(ctx context.Context, id string) (*OverallModerationConfiguration, error)
	CreateCustomModelVersionFromGuardConfigurations(ctx context.Context, id string, req *CreateCustomModelVersionFromGuardsConfigurationRequest) (*CreateCustomModelVersionFromGuardsConfigurationResponse, error)
	CreateDependencyBuild(ctx context.Context, id string, versionID string) (*DependencyBuild, error)
	GetDependencyBuild(ctx context.Context, id string, versionID string) (*DependencyBuild, error)

	// Registered Model
	CreateRegisteredModelFromCustomModelVersion(ctx context.Context, req *CreateRegisteredModelFromCustomModelRequest) (*RegisteredModelVersion, error)
	UpdateRegisteredModelVersion(ctx context.Context, registeredModelId string, versionId string, req *UpdateRegisteredModelVersionRequest) (*RegisteredModelVersion, error)
	ListRegisteredModelVersions(ctx context.Context, id string) (*ListRegisteredModelVersionsResponse, error)
	GetLatestRegisteredModelVersion(ctx context.Context, id string) (*RegisteredModelVersion, error)
	GetRegisteredModelVersion(ctx context.Context, registeredModelId string, versionId string) (*RegisteredModelVersion, error)
	IsRegisteredModelVersionReady(ctx context.Context, registeredModelId string, versionId string) (bool, error)
	ListRegisteredModels(ctx context.Context) (*ListRegisteredModelsResponse, error)
	GetRegisteredModel(ctx context.Context, id string) (*RegisteredModel, error)
	UpdateRegisteredModel(ctx context.Context, id string, req *UpdateRegisteredModelRequest) (*RegisteredModel, error)
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
	ValidateDeploymentModelReplacement(ctx context.Context, id string, req *ValidateDeployemntModelReplacementRequest) (*ValidateDeployemntModelReplacementResponse, error)
	UpdateDeploymentModel(ctx context.Context, id string, req *UpdateDeploymentModelRequest) (*DeploymentRetrieveResponse, string, error)

	// Application Source
	CreateApplicationSource(ctx context.Context) (*ApplicationSource, error)
	GetApplicationSource(ctx context.Context, id string) (*ApplicationSource, error)
	UpdateApplicationSource(ctx context.Context, id string, req *UpdateApplicationSourceRequest) (*ApplicationSource, error)
	ListApplicationSourceVersions(ctx context.Context, id string) (*ListApplicationSourceVersionsResponse, error)
	CreateApplicationSourceVersion(ctx context.Context, id string, req *CreateApplicationSourceVersionRequest) (*ApplicationSourceVersion, error)
	UpdateApplicationSourceVersion(ctx context.Context, id string, versionId string, req *UpdateApplicationSourceVersionRequest) (*ApplicationSourceVersion, error)
	UpdateApplicationSourceVersionFiles(ctx context.Context, id string, versionId string, files []FileInfo) (*ApplicationSourceVersion, error)
	GetApplicationSourceVersion(ctx context.Context, id string, versionId string) (*ApplicationSourceVersion, error)
	DeleteApplicationSource(ctx context.Context, id string) error

	// Application
	CreateApplicationFromSource(ctx context.Context, req *CreateApplicationFromSourceRequest) (*Application, error)
	CreateQAApplication(ctx context.Context, req *CreateQAApplicationRequest) (*Application, error)
	GetApplication(ctx context.Context, id string) (*Application, error)
	IsApplicationReady(ctx context.Context, id string) (bool, error)
	UpdateApplication(ctx context.Context, id string, req *UpdateApplicationRequest) (*Application, error)
	DeleteApplication(ctx context.Context, id string) error

	// Credential
	CreateCredential(ctx context.Context, req *CredentialRequest) (*CredentialResponse, error)
	GetCredential(ctx context.Context, id string) (*CredentialResponse, error)
	UpdateCredential(ctx context.Context, id string, req *CredentialRequest) (*CredentialResponse, error)
	DeleteCredential(ctx context.Context, id string) error

	// Async Tasks
	GetTaskStatus(ctx context.Context, id string) (*TaskStatusResponse, error)

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
	return Get[DatasetResponse](s.client, ctx, "/datasets/"+id+"/")
}

func (s *ServiceImpl) UpdateDataset(ctx context.Context, id string, req *UpdateDatasetRequest) (*DatasetResponse, error) {
	return Patch[DatasetResponse](s.client, ctx, "/datasets/"+id+"/", req)
}

func (s *ServiceImpl) DeleteDataset(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/datasets/"+id+"/")
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

// Use Case Service Implementation.
func (s *ServiceImpl) LinkDatasetToUseCase(ctx context.Context, useCaseID, datasetID string) error {
	_, err := Post[CreateVoidResponse](s.client, ctx, "/useCases/"+useCaseID+"/datasets/"+datasetID+"/", &CreateVoidRequest{})
	return err
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

func (s *ServiceImpl) ListLLMs(ctx context.Context) (*ListLLMsResponse, error) {
	return Get[ListLLMsResponse](s.client, ctx, "/genai/llms/")
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
	return PatchAndExpectStatus[CustomModelVersion](s.client, ctx, "/customModels/"+id+"/versions/fromRepository/", req)
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

func (s *ServiceImpl) CreateDependencyBuild(ctx context.Context, id string, versionID string) (*DependencyBuild, error) {
	return Post[DependencyBuild](s.client, ctx, "/customModels/"+id+"/versions/"+versionID+"/dependencyBuild/", map[string]string{})
}

func (s *ServiceImpl) GetDependencyBuild(ctx context.Context, id string, versionID string) (*DependencyBuild, error) {
	return Get[DependencyBuild](s.client, ctx, "/customModels/"+id+"/versions/"+versionID+"/dependencyBuild/")
}

// Registered Model Service Implementation.
func (s *ServiceImpl) CreateRegisteredModelFromCustomModelVersion(ctx context.Context, req *CreateRegisteredModelFromCustomModelRequest) (*RegisteredModelVersion, error) {
	return Post[RegisteredModelVersion](s.client, ctx, "/modelPackages/fromCustomModelVersion/", req)
}

func (s *ServiceImpl) UpdateRegisteredModelVersion(ctx context.Context, registeredModelId string, versionId string, req *UpdateRegisteredModelVersionRequest) (*RegisteredModelVersion, error) {
	return Patch[RegisteredModelVersion](s.client, ctx, "/registeredModels/"+registeredModelId+"/versions/"+versionId+"/", req)
}

func (s *ServiceImpl) GetRegisteredModelVersion(ctx context.Context, registeredModelId string, versionId string) (*RegisteredModelVersion, error) {
	return Get[RegisteredModelVersion](s.client, ctx, "/registeredModels/"+registeredModelId+"/versions/"+versionId+"/")
}

func (s *ServiceImpl) ListRegisteredModelVersions(ctx context.Context, id string) (*ListRegisteredModelVersionsResponse, error) {
	return Get[ListRegisteredModelVersionsResponse](s.client, ctx, "/registeredModels/"+id+"/versions/")
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

func (s *ServiceImpl) ValidateDeploymentModelReplacement(ctx context.Context, id string, req *ValidateDeployemntModelReplacementRequest) (*ValidateDeployemntModelReplacementResponse, error) {
	return Post[ValidateDeployemntModelReplacementResponse](s.client, ctx, "/deployments/"+id+"/model/validation/", req)
}

func (s *ServiceImpl) UpdateDeploymentModel(ctx context.Context, id string, req *UpdateDeploymentModelRequest) (*DeploymentRetrieveResponse, string, error) {
	return PatchAndExpectStatus[DeploymentRetrieveResponse](s.client, ctx, "/deployments/"+id+"/model/", req)
}

// Application Service Implementation.
func (s *ServiceImpl) CreateApplicationSource(ctx context.Context) (*ApplicationSource, error) {
	return Post[ApplicationSource](s.client, ctx, "/customApplicationSources/", map[string]string{})
}

func (s *ServiceImpl) GetApplicationSource(ctx context.Context, id string) (*ApplicationSource, error) {
	return Get[ApplicationSource](s.client, ctx, "/customApplicationSources/"+id+"/")
}

func (s *ServiceImpl) UpdateApplicationSource(ctx context.Context, id string, req *UpdateApplicationSourceRequest) (*ApplicationSource, error) {
	return Patch[ApplicationSource](s.client, ctx, "/customApplicationSources/"+id+"/", req)
}

func (s *ServiceImpl) ListApplicationSourceVersions(ctx context.Context, id string) (*ListApplicationSourceVersionsResponse, error) {
	return Get[ListApplicationSourceVersionsResponse](s.client, ctx, "/customApplicationSources/"+id+"/versions/")
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

func (s *ServiceImpl) CreateApplicationFromSource(ctx context.Context, req *CreateApplicationFromSourceRequest) (*Application, error) {
	return Post[Application](s.client, ctx, "/customApplications/", req)
}

func (s *ServiceImpl) CreateQAApplication(ctx context.Context, req *CreateQAApplicationRequest) (*Application, error) {
	return Post[Application](s.client, ctx, "/customApplications/qanda/", req)
}

func (s *ServiceImpl) GetApplication(ctx context.Context, id string) (*Application, error) {
	return Get[Application](s.client, ctx, "/customApplications/"+id+"/")
}

func (s *ServiceImpl) IsApplicationReady(ctx context.Context, id string) (bool, error) {
	customApplication, err := s.GetApplication(ctx, id)
	if err != nil {
		return false, err
	}

	return customApplication.Status == "running", nil
}

func (s *ServiceImpl) UpdateApplication(ctx context.Context, id string, req *UpdateApplicationRequest) (*Application, error) {
	return Patch[Application](s.client, ctx, "/customApplications/"+id+"/", req)
}

func (s *ServiceImpl) DeleteApplication(ctx context.Context, id string) error {
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
