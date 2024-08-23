package client

import (
	"context"
	_ "embed"
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed datarobot_english_documentation_docsassist.zip
var dataRobotEnglishDocumentationDocAssistsZipFileContent []byte

const dataRobotEnglishDocumentationDocAssistsZipFileName = "datarobot_english_documentation_docsassist.zip"

func TestCredential(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	assert := assert.New(t)
	require := require.New(t)

	s := initializeTest(t)

	// Create a credential
	credential, err := s.CreateCredential(ctx, &client.CredentialRequest{
		Name:           "Integration Test" + uuid.New().String(),
		Description:    "This is a test credential.",
		CredentialType: client.CredentialTypeBasic,
		User:           "test",
		Password:       "test",
	})
	require.NoError(err)
	require.NotNil(credential)
	assert.NotEmpty(credential.ID)

	defer func() {
		err = s.DeleteCredential(ctx, credential.ID)
		require.NoError(err)
	}()

	getCredential, err := s.GetCredential(ctx, credential.ID)
	require.NoError(err)
	require.NotNil(getCredential)
	require.Equal(credential.ID, getCredential.ID)
	require.Equal(credential.Name, getCredential.Name)
	require.Equal(credential.Description, getCredential.Description)
	require.Equal(credential.CredentialType, getCredential.CredentialType)

	// Update the credential
	updateName := "Updated Integration Test" + uuid.New().String()
	updateDescription := "This is an updated test credential."
	_, err = s.UpdateCredential(ctx, credential.ID, &client.CredentialRequest{
		Name:        updateName,
		Description: updateDescription,
	})
	require.NoError(err)

	updatedCredential, err := s.GetCredential(ctx, credential.ID)
	require.NoError(err)
	require.NotNil(updatedCredential)
	require.Equal(credential.ID, updatedCredential.ID)
	require.Equal(updateName, updatedCredential.Name)
	require.Equal(updateDescription, updatedCredential.Description)

	credentialApiToken, err := s.CreateCredential(ctx, &client.CredentialRequest{
		Name:           "Integration Test" + uuid.New().String(),
		Description:    "This is a test credential.",
		CredentialType: client.CredentialTypeApiToken,
		ApiToken:       "token",
	})
	require.NoError(err)
	require.NotNil(credentialApiToken)
	assert.NotEmpty(credentialApiToken.ID)

	defer func() {
		err = s.DeleteCredential(ctx, credentialApiToken.ID)
		require.NoError(err)
	}()
}

func TestCustomModel(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	assert := assert.New(t)
	require := require.New(t)

	s := initializeTest(t)

	// Create a use case request
	req := &client.UseCaseRequest{
		Name:        "Integration Test" + uuid.New().String(),
		Description: "This is a test use case.",
	}

	// Create a use case
	useCase, err := s.CreateUseCase(ctx, req)
	require.NoError(err)
	require.NotNil(useCase)
	require.NotEmpty(useCase.ID)

	// Delete the use case
	defer func() {
		err = s.DeleteUseCase(ctx, useCase.ID)
		require.NoError(err)
	}()

	name := "Integration Test" + uuid.New().String()
	description := "This is a test playground."
	playground, err := s.CreatePlayground(ctx, &client.CreatePlaygroundRequest{
		Name:        name,
		Description: description,
		UseCaseID:   useCase.ID,
	})
	require.NoError(err)
	require.NotNil(playground)
	assert.NotEmpty(playground.ID)

	getPlayground, err := s.GetPlayground(ctx, playground.ID)
	require.NoError(err)
	require.NotNil(getPlayground)
	require.Equal(playground.ID, getPlayground.ID)
	require.Equal(name, getPlayground.Name)
	require.Equal(description, getPlayground.Description)

	llmBlueprintName := "Integration Test" + uuid.New().String()
	llmBlueprint, err := s.CreateLLMBlueprint(ctx, &client.CreateLLMBlueprintRequest{
		Name:         llmBlueprintName,
		Description:  "This is a test LLM blueprint.",
		PlaygroundID: playground.ID,
		LLMID:        "azure-openai-gpt-3.5-turbo",
	})
	require.NoError(err)
	require.NotEmpty(llmBlueprint.ID)
	require.Equal(llmBlueprintName, llmBlueprint.Name)

	resp, err := s.CreateCustomModelFromLLMBlueprint(ctx, &client.CreateCustomModelFromLLMBlueprintRequest{
		LLMBlueprintID: llmBlueprint.ID,
	})
	require.NoError(err)
	require.NotEmpty(resp.CustomModelID)

	defer func() {
		err = s.DeleteCustomModel(ctx, resp.CustomModelID)
		require.NoError(err)
	}()

	timeout := 1 * time.Minute
	start := time.Now()
	for {
		status, err := s.IsCustomModelReady(ctx, resp.CustomModelID)
		require.NoError(err)
		if status {
			break
		}
		if time.Since(start) > timeout {
			require.FailNow("timeout reached while waiting for custom model to be ready")
		}
		time.Sleep(1 * time.Second)
	}

	customModel, err := s.GetCustomModel(ctx, resp.CustomModelID)
	require.NoError(err)
	assert.Equal(resp.CustomModelID, customModel.ID)
	assert.Equal(llmBlueprintName, customModel.Name)
	assert.NotEmpty(customModel.LatestVersion.BaseEnvironmentID)
	assert.NotEmpty(customModel.LatestVersion.BaseEnvironmentVersionID)
	assert.NotEmpty(customModel.LatestVersion.RuntimeParameters)

	name = "Updated Integration Test" + uuid.New().String()
	description = "This is an updated test custom model."
	_, err = s.UpdateCustomModel(ctx, resp.CustomModelID, &client.CustomModelUpdate{
		Name:        name,
		Description: description,
	})
	require.NoError(err)

	params := []client.RuntimeParameterValueRequest{
		{
			FieldName: "OPENAI_API_BASE",
			Type:      client.RuntimeParameterTypeString,
			Value:     "https://api.openai.com",
		},
	}

	jsonParams, err := json.Marshal(params)
	if err != nil {
		log.Fatalf("Error encoding JSON: %v", err)
	}

	newVersion, err := s.CreateCustomModelVersionCreateFromLatest(ctx, resp.CustomModelID, &client.CreateCustomModelVersionCreateFromLatestRequest{
		BaseEnvironmentID:        customModel.LatestVersion.BaseEnvironmentID,
		BaseEnvironmentVersionID: customModel.LatestVersion.BaseEnvironmentVersionID,
		IsMajorUpdate:            "false",
		RuntimeParameterValues:   string(jsonParams),
	})
	require.NoError(err)
	require.NotEmpty(newVersion.ID)
	require.NotEmpty(newVersion.CustomModelID)
	assert.Equal(resp.CustomModelID, newVersion.CustomModelID)

	start = time.Now()
	for {
		status, err := s.IsCustomModelReady(ctx, resp.CustomModelID)
		require.NoError(err)
		if status {
			break
		}
		if time.Since(start) > timeout {
			require.FailNow("timeout reached while waiting for custom model to be ready")
		}
		time.Sleep(1 * time.Second)
	}

	customModel, err = s.GetCustomModel(ctx, resp.CustomModelID)
	require.NoError(err)
	assert.Equal(resp.CustomModelID, customModel.ID)
	assert.Equal(newVersion.ID, customModel.LatestVersion.ID)
	assert.Equal(name, customModel.Name)
	assert.Equal(description, customModel.Description)
}

func TestCustomModelFromGitHub(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	assert := assert.New(t)
	require := require.New(t)

	s := initializeTest(t)

	// Create a remote repository
	name := "Integration Test" + uuid.New().String()
	description := "This is a test remote repository."
	location := "https://github.com/datarobot-community/custom-models"
	sourceType := "github"

	resp, err := s.CreateRemoteRepository(ctx, &client.CreateRemoteRepositoryRequest{
		Name:        name,
		Description: description,
		Location:    location,
		SourceType:  sourceType,
	})
	require.NoError(err)
	require.Equal(name, resp.Name)
	require.Equal(description, resp.Description)
	require.Equal(location, resp.Location)
	require.Equal(sourceType, resp.SourceType)

	resp, err = s.GetRemoteRepository(ctx, resp.ID)
	require.NoError(err)
	require.Equal(name, resp.Name)
	require.Equal(description, resp.Description)
	require.Equal(location, resp.Location)
	require.Equal(sourceType, resp.SourceType)

	newName := "Updated Integration Test" + uuid.New().String()
	newDescription := "This is an updated test remote repository."
	remoteRepository, err := s.UpdateRemoteRepository(ctx, resp.ID, &client.UpdateRemoteRepositoryRequest{
		Name:        newName,
		Description: newDescription,
	})
	require.NoError(err)
	require.Equal(newName, remoteRepository.Name)
	require.Equal(newDescription, remoteRepository.Description)

	listResp, err := s.ListExecutionEnvironments(ctx)
	require.NoError(err)

	var environmentID string
	var environmentVersionID string
	for _, executionEnvironment := range listResp.Data {
		if executionEnvironment.Name == "[GenAI] Python 3.11 with Moderations" {
			environmentID = executionEnvironment.ID
			environmentVersionID = executionEnvironment.LatestVersion.ID
			break
		}
	}
	require.NotEmpty(environmentID)
	require.NotEmpty(environmentVersionID)

	name = "Integration Test" + uuid.New().String()
	description = "This is a test custom model."
	customModel, err := s.CreateCustomModel(ctx, &client.CreateCustomModelRequest{
		Name:            name,
		Description:     description,
		TargetType:      "TextGeneration",
		TargetName:      "promptText",
		CustomModelType: "inference",
	})
	require.NoError(err)
	require.NotEmpty(customModel.ID)

	_, err = s.CreateCustomModelVersionFromRemoteRepository(ctx, customModel.ID, &client.CreateCustomModelVersionFromRemoteRepositoryRequest{
		BaseEnvironmentID: environmentID,
		IsMajorUpdate:     true,
		RepositoryID:      remoteRepository.ID,
		Ref:               "master",
		SourcePath: []string{
			"custom_inference/python/gan_mnist/custom.py",
			"custom_inference/python/gan_mnist/gan_weights.h5",
		},
	})
	require.NoError(err)

	timeout := 1 * time.Minute
	start := time.Now()
	for {
		status, err := s.IsCustomModelReady(ctx, customModel.ID)
		require.NoError(err)
		if status {
			break
		}
		if time.Since(start) > timeout {
			require.FailNow("timeout reached while waiting for custom model to be ready")
		}
		time.Sleep(1 * time.Second)
	}

	customModel, err = s.GetCustomModel(ctx, customModel.ID)
	require.NoError(err)

	customModel, err = s.GetCustomModel(ctx, customModel.ID)
	require.NoError(err)
	assert.Equal(customModel.ID, customModel.ID)
	assert.Equal(name, customModel.Name)
	assert.Equal(description, customModel.Description)

	err = s.DeleteCustomModel(ctx, customModel.ID)
	require.NoError(err)

	err = s.DeleteRemoteRepository(ctx, resp.ID)
	require.NoError(err)
}

func TestApplicationFromCustomModel(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	assert := assert.New(t)
	require := require.New(t)

	s := initializeTest(t)

	// Create a use case request
	req := &client.UseCaseRequest{
		Name:        "Integration Test" + uuid.New().String(),
		Description: "This is a test use case.",
	}

	// Create a use case
	useCase, err := s.CreateUseCase(ctx, req)
	require.NoError(err)
	require.NotNil(useCase)
	require.NotEmpty(useCase.ID)

	// Delete the use case
	defer func() {
		err = s.DeleteUseCase(ctx, useCase.ID)
		require.NoError(err)
	}()

	name := "Integration Test" + uuid.New().String()
	description := "This is a test playground."
	playground, err := s.CreatePlayground(ctx, &client.CreatePlaygroundRequest{
		Name:        name,
		Description: description,
		UseCaseID:   useCase.ID,
	})
	require.NoError(err)
	require.NotNil(playground)
	assert.NotEmpty(playground.ID)

	getPlayground, err := s.GetPlayground(ctx, playground.ID)
	require.NoError(err)
	require.NotNil(getPlayground)
	require.Equal(playground.ID, getPlayground.ID)
	require.Equal(name, getPlayground.Name)
	require.Equal(description, getPlayground.Description)

	llmBlueprintName := "Integration Test" + uuid.New().String()
	llmBlueprint, err := s.CreateLLMBlueprint(ctx, &client.CreateLLMBlueprintRequest{
		Name:         llmBlueprintName,
		Description:  "This is a test LLM blueprint.",
		PlaygroundID: playground.ID,
		LLMID:        "azure-openai-gpt-3.5-turbo",
	})
	require.NoError(err)
	require.NotEmpty(llmBlueprint.ID)
	require.Equal(llmBlueprintName, llmBlueprint.Name)

	updatedLLMBlueprintName := "Updated Integration Test" + uuid.New().String()
	updatedLLmBlueprint, err := s.UpdateLLMBlueprint(ctx, llmBlueprint.ID, &client.UpdateLLMBlueprintRequest{
		Name: updatedLLMBlueprintName,
	})
	require.NoError(err)
	require.Equal(llmBlueprint.ID, updatedLLmBlueprint.ID)
	require.Equal(updatedLLMBlueprintName, updatedLLmBlueprint.Name)

	resp, err := s.CreateCustomModelFromLLMBlueprint(ctx, &client.CreateCustomModelFromLLMBlueprintRequest{
		LLMBlueprintID: llmBlueprint.ID,
	})
	require.NoError(err)
	require.NotEmpty(resp.CustomModelID)

	defer func() {
		err = s.DeleteCustomModel(ctx, resp.CustomModelID)
		require.NoError(err)
	}()

	timeout := 5 * time.Minute
	start := time.Now()
	for {
		status, err := s.IsCustomModelReady(ctx, resp.CustomModelID)
		require.NoError(err)
		if status {
			break
		}
		if time.Since(start) > timeout {
			require.FailNow("timeout reached while waiting for custom model to be ready")
		}
		time.Sleep(1 * time.Second)
	}

	customModel, err := s.GetCustomModel(ctx, resp.CustomModelID)
	require.NoError(err)

	guardConfigurations, err := s.GetGuardConfigurationsForCustomModelVersion(ctx, customModel.LatestVersion.ID)
	require.NoError(err)
	require.Len(guardConfigurations.Data, 3)
	require.Equal("Rouge 1", guardConfigurations.Data[0].Name)
	require.Equal("Token Count", guardConfigurations.Data[1].Name)
	require.Equal("Token Count", guardConfigurations.Data[2].Name)

	overallModerationConfiguration, err := s.GetOverallModerationConfigurationForCustomModelVersion(ctx, customModel.LatestVersion.ID)
	require.NoError(err)
	require.Equal("score", overallModerationConfiguration.TimeoutAction)
	require.Equal(60, overallModerationConfiguration.TimeoutSec)

	overallModerationConfiguration.TimeoutSec = 120

	listGuardTemplatesResp, err := s.ListGuardTemplates(ctx)
	require.NoError(err)
	require.NotEmpty(listGuardTemplatesResp.Data)

	var newGuardData client.GuardConfiguration

	for _, guard := range guardConfigurations.Data {
		if guard.Name == "Rouge 1" {
			newGuardData = client.GuardConfiguration{
				Name:         guard.Name + " response",
				Description:  guard.Description,
				Type:         guard.Type,
				Stages:       []string{"response"},
				Intervention: guard.Intervention,
				OOTBType:     guard.OOTBType,
			}
			// message is required, this is the default
			newGuardData.Intervention.Message = "This message has triggered moderation criteria and therefore been blocked by the DataRobot moderation system."
			break
		}
	}
	require.NotEmpty(newGuardData.Name)

	newGuardConfigurationData := []client.GuardConfiguration{newGuardData}

	for _, guard := range guardConfigurations.Data {
		intervention := guard.Intervention
		intervention.Message = "This message has triggered moderation criteria and therefore been blocked by the DataRobot moderation system."
		newGuardConfigurationData = append(newGuardConfigurationData, client.GuardConfiguration{
			Name:         guard.Name,
			Description:  guard.Description,
			Type:         guard.Type,
			Stages:       guard.Stages,
			Intervention: intervention,
			OOTBType:     guard.OOTBType,
		})
	}

	createCustomModelVersionFromGuardsResp, err := s.CreateCustomModelVersionFromGuardConfigurations(ctx, customModel.LatestVersion.ID, &client.CreateCustomModelVersionFromGuardsConfigurationRequest{
		CustomModelID: customModel.ID,
		Data:          newGuardConfigurationData,
		OverallConfig: *overallModerationConfiguration,
	})
	require.NoError(err)
	latestVersion := createCustomModelVersionFromGuardsResp.CustomModelVersionID
	require.NotEmpty(latestVersion)
	require.NotEqual(customModel.LatestVersion.ID, latestVersion)

	overallModerationConfiguration, err = s.GetOverallModerationConfigurationForCustomModelVersion(ctx, latestVersion)
	require.NoError(err)
	require.Equal("score", overallModerationConfiguration.TimeoutAction)
	require.Equal(120, overallModerationConfiguration.TimeoutSec)

	guardConfigurations, err = s.GetGuardConfigurationsForCustomModelVersion(ctx, latestVersion)
	require.NoError(err)
	require.Len(guardConfigurations.Data, 4)

	registeredModelVersion, err := s.CreateRegisteredModelFromCustomModelVersion(ctx, &client.CreateRegisteredModelFromCustomModelRequest{
		CustomModelVersionID: latestVersion,
		Name:                 "Integration Test" + uuid.New().String(),
	})
	require.NoError(err)
	require.NotEmpty(registeredModelVersion.ID)

	start = time.Now()
	for {
		status, err := s.IsRegisteredModelVersionReady(ctx, registeredModelVersion.RegisteredModelID, registeredModelVersion.ID)
		require.NoError(err)
		if status {
			break
		}
		if time.Since(start) > timeout {
			require.FailNow("timeout reached while waiting for registered model version to be ready")
		}
		time.Sleep(1 * time.Second)
	}

	registeredModelName := "Updated Integration Test" + uuid.New().String()
	registeredModelDescription := "This is an updated test registered model."
	registeredModelResp, err := s.UpdateRegisteredModel(ctx, registeredModelVersion.RegisteredModelID, &client.RegisteredModelUpdate{
		Name:        registeredModelName,
		Description: registeredModelDescription,
	})
	require.NoError(err)
	require.Equal(registeredModelVersion.RegisteredModelID, registeredModelResp.ID)

	predictionEnvironment, err := s.CreatePredictionEnvironment(ctx, &client.CreatePredictionEnvironmentRequest{
		Name:     "Integration Test" + uuid.New().String(),
		Platform: "aws",
	})
	require.NoError(err)
	require.NotEmpty(predictionEnvironment.ID)
	require.NotEmpty(predictionEnvironment.Name)

	createDeploymentResp, err := s.CreateDeploymentFromModelPackage(ctx, &client.CreateDeploymentFromModelPackageRequest{
		ModelPackageID:          registeredModelVersion.ID,
		PredictionEnvironmentID: predictionEnvironment.ID,
		Label:                   registeredModelVersion.Name,
	})
	require.NoError(err)
	require.NotEmpty(createDeploymentResp.ID)

	deployment, err := s.GetDeployment(ctx, createDeploymentResp.ID)
	require.NoError(err)
	require.NotEmpty(deployment.ID)
	require.Equal(createDeploymentResp.ID, deployment.ID)
	require.NotEmpty(deployment.Label)
	require.NotEmpty(deployment.Status)

	newLabel := "Updated Integration Test" + uuid.New().String()
	_, err = s.UpdateDeployment(ctx, deployment.ID, &client.UpdateDeploymentRequest{
		Label: newLabel,
	})
	require.NoError(err)

	_, err = s.UpdateDeploymentSettings(ctx, deployment.ID, &client.DeploymentSettings{
		AssociationID: &client.AssociationIDSetting{
			AutoGenerateID:               true,
			RequiredInPredictionRequests: false,
			ColumnNames:                  []string{"column1"},
		},
		PredictionsDataCollection: &client.BasicSetting{Enabled: true},
	})
	require.NoError(err)

	settings, err := s.GetDeploymentSettings(ctx, deployment.ID)
	require.NoError(err)
	require.NotEmpty(settings.AssociationID)
	require.True(settings.AssociationID.AutoGenerateID)
	require.False(settings.AssociationID.RequiredInPredictionRequests)
	require.Equal([]string{"column1"}, settings.AssociationID.ColumnNames)

	deployment, err = s.GetDeployment(ctx, createDeploymentResp.ID)
	require.NoError(err)
	require.NotEmpty(deployment.ID)
	require.Equal(newLabel, deployment.Label)

	application, err := s.CreateChatApplication(ctx, &client.CreateChatApplicationRequest{
		DeploymentID: deployment.ID,
	})
	require.NoError(err)
	require.NotEmpty(application.ID)

	start = time.Now()
	for {
		status, err := s.IsChatApplicationReady(ctx, application.ID)
		require.NoError(err)
		if status {
			break
		}
		if time.Since(start) > timeout {
			require.FailNow("timeout reached while waiting for custom application to be ready")
		}
		time.Sleep(1 * time.Second)
	}

	newName := "Updated Integration Test" + uuid.New().String()
	_, err = s.UpdateChatApplication(ctx, application.ID, &client.UpdateChatApplicationRequest{
		Name: newName,
	})
	require.NoError(err)

	updatedApplication, err := s.GetChatApplication(ctx, application.ID)
	require.NoError(err)
	require.NotEmpty(updatedApplication.ID)
	require.Equal(newName, updatedApplication.Name)
	require.NotEmpty(updatedApplication.ApplicationUrl)

	applicationSource, err := s.GetChatApplicationSource(ctx, updatedApplication.CustomApplicationSourceID)
	require.NoError(err)
	require.NotEmpty(applicationSource.LatestVersion.Label)

	// Delete the entities
	err = s.DeleteChatApplication(ctx, application.ID)
	require.NoError(err)

	err = s.DeleteChatApplicationSource(ctx, updatedApplication.CustomApplicationSourceID)
	require.NoError(err)

	err = s.DeleteDeployment(ctx, deployment.ID)
	require.NoError(err)

	err = s.DeletePredictionEnvironment(ctx, predictionEnvironment.ID)
	require.NoError(err)

	err = s.DeleteRegisteredModel(ctx, registeredModelVersion.RegisteredModelID)
	require.NoError(err)
}

func TestLLMBlueprint(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	assert := assert.New(t)
	require := require.New(t)

	s := initializeTest(t)

	// Create a use case request
	req := &client.UseCaseRequest{
		Name:        "Integration Test" + uuid.New().String(),
		Description: "This is a test use case.",
	}

	// Create a use case
	useCase, err := s.CreateUseCase(ctx, req)
	require.NoError(err)
	require.NotNil(useCase)
	require.NotEmpty(useCase.ID)

	// Delete the use case
	defer func() {
		err = s.DeleteUseCase(ctx, useCase.ID)
		require.NoError(err)
	}()

	name := "Integration Test" + uuid.New().String()
	description := "This is a test playground."
	playground, err := s.CreatePlayground(ctx, &client.CreatePlaygroundRequest{
		Name:        name,
		Description: description,
		UseCaseID:   useCase.ID,
	})
	require.NoError(err)
	require.NotNil(playground)
	assert.NotEmpty(playground.ID)

	getPlayground, err := s.GetPlayground(ctx, playground.ID)
	require.NoError(err)
	require.NotNil(getPlayground)
	require.Equal(playground.ID, getPlayground.ID)
	require.Equal(name, getPlayground.Name)
	require.Equal(description, getPlayground.Description)

	llmBlueprintName := "Integration Test" + uuid.New().String()
	llmBlueprint, err := s.CreateLLMBlueprint(ctx, &client.CreateLLMBlueprintRequest{
		Name:         llmBlueprintName,
		Description:  "This is a test LLM blueprint.",
		PlaygroundID: playground.ID,
		LLMID:        "azure-openai-gpt-3.5-turbo",
	})
	require.NoError(err)
	require.NotEmpty(llmBlueprint.ID)
	require.Equal(llmBlueprintName, llmBlueprint.Name)

	defer func() {
		err = s.DeleteLLMBlueprint(ctx, llmBlueprint.ID)
		require.NoError(err)
	}()

	updatedLLMBlueprintName := "Updated Integration Test" + uuid.New().String()
	updatedLLmBlueprint, err := s.UpdateLLMBlueprint(ctx, llmBlueprint.ID, &client.UpdateLLMBlueprintRequest{
		Name: updatedLLMBlueprintName,
	})
	require.NoError(err)
	require.Equal(llmBlueprint.ID, updatedLLmBlueprint.ID)
	require.Equal(updatedLLMBlueprintName, updatedLLmBlueprint.Name)
}

func TestVectorDatabase(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	assert := assert.New(t)
	require := require.New(t)

	s := initializeTest(t)

	// Create a use case
	useCase, err := s.CreateUseCase(ctx, &client.UseCaseRequest{
		Name:        "Integration Test" + uuid.New().String(),
		Description: "This is a test use case.",
	})
	require.NoError(err)
	require.NotNil(useCase)
	require.NotEmpty(useCase.ID)

	// Delete the use case
	defer func() {
		err = s.DeleteUseCase(ctx, useCase.ID)
		require.NoError(err)
	}()

	// Upload file to the dataset
	dataset, err := s.CreateDatasetFromFile(ctx,
		dataRobotEnglishDocumentationDocAssistsZipFileName,
		dataRobotEnglishDocumentationDocAssistsZipFileContent,
	)
	require.NoError(err)
	require.NotNil(dataset)
	assert.NotEmpty(dataset.ID)
	assert.NotEmpty(dataset.VersionID)
	assert.NotEmpty(dataset.StatusID)

	// Link the dataset to the use case
	err = s.LinkDatasetToUseCase(ctx, useCase.ID, dataset.ID)
	require.NoError(err)

	// wait for data source to be ready
	timeout := 5 * time.Minute
	start := time.Now()
	for {
		status, err := s.IsDatasetReadyForVectorDatabase(ctx, dataset.ID)
		require.NoError(err)
		if status {
			break
		}
		if time.Since(start) > timeout {
			require.FailNow("timeout reached while waiting for dataset to be ready")
		}
		time.Sleep(1 * time.Second)
	}

	name := "Integration Test" + uuid.New().String()
	vdb, err := s.CreateVectorDatabase(ctx, &client.CreateVectorDatabaseRequest{
		Name:      name,
		UseCaseID: useCase.ID,
		DatasetID: dataset.ID,
		ChunkingParameters: client.ChunkingParameters{
			ChunkOverlapPercentage: 0,
			ChunkSize:              256,
			ChunkingMethod:         "recursive",
			EmbeddingModel:         "jinaai/jina-embedding-t-en-v1",
			IsSeparatorRegex:       false,
			Separators:             []string{"↵↵", "↵", " ", ""},
		},
	})
	require.NoError(err)
	require.NotNil(vdb)
	assert.NotEmpty(vdb.ID)

	defer func() {
		err = s.DeleteVectorDatabase(ctx, vdb.ID)
		require.NoError(err)
	}()

	getVdb, err := s.GetVectorDatabase(ctx, vdb.ID)
	require.NoError(err)
	require.NotNil(getVdb)
	assert.Equal(vdb.ID, getVdb.ID)
	assert.Equal(name, getVdb.Name)
	assert.Equal(useCase.ID, getVdb.UseCaseID)
	assert.Equal(dataset.ID, getVdb.DatasetID)
	assert.NotEmpty(getVdb.ExecutionStatus)

	// Update the vector database request
	updateName := "Updated Integration Test" + uuid.New().String()
	updateVdb, err := s.UpdateVectorDatabase(ctx, vdb.ID, &client.UpdateVectorDatabaseRequest{
		Name: updateName,
	})
	require.NoError(err)
	require.NotNil(updateVdb)
	assert.Equal(vdb.ID, updateVdb.ID)
	assert.Equal(updateName, updateVdb.Name)

	// Verify the updated vector database
	updatedVdb, err := s.GetVectorDatabase(ctx, vdb.ID)
	require.NoError(err)
	assert.NotNil(updatedVdb)
	assert.Equal(vdb.ID, updatedVdb.ID)
	assert.Equal(updateName, updatedVdb.Name)

	// wait for the dataset to be processed and timeout after 1 minute
	checkStatus := func() bool {
		status, err := s.IsVectorDatabaseReady(ctx, getVdb.ID)
		require.NoError(err)
		return status
	}
	require.Eventually(checkStatus, 1*time.Minute, 1*time.Second, "vector database processing failed")

	err = s.DeleteDataset(ctx, dataset.ID)
	require.NoError(err)
}

func TestPlayground(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	assert := assert.New(t)
	require := require.New(t)

	s := initializeTest(t)

	// Create a use case request
	req := &client.UseCaseRequest{
		Name:        "Integration Test" + uuid.New().String(),
		Description: "This is a test use case.",
	}

	// Create a use case
	useCase, err := s.CreateUseCase(ctx, req)
	require.NoError(err)
	require.NotNil(useCase)
	require.NotEmpty(useCase.ID)

	// Delete the use case
	defer func() {
		err = s.DeleteUseCase(ctx, useCase.ID)
		require.NoError(err)
	}()

	name := "Integration Test" + uuid.New().String()
	description := "This is a test playground."
	playground, err := s.CreatePlayground(ctx, &client.CreatePlaygroundRequest{
		Name:        name,
		Description: description,
		UseCaseID:   useCase.ID,
	})
	require.NoError(err)
	require.NotNil(playground)
	assert.NotEmpty(playground.ID)

	getPlayground, err := s.GetPlayground(ctx, playground.ID)
	require.NoError(err)
	require.NotNil(getPlayground)
	require.Equal(playground.ID, getPlayground.ID)
	require.Equal(name, getPlayground.Name)
	require.Equal(description, getPlayground.Description)

	// Update the playground request
	updateName := "Updated Integration Test" + uuid.New().String()
	updateDescription := "This is an updated test playground."
	updatePlayground, err := s.UpdatePlayground(ctx, playground.ID, &client.UpdatePlaygroundRequest{
		Name:        updateName,
		Description: updateDescription,
	})
	require.NoError(err)
	require.NotNil(updatePlayground)
	assert.Equal(playground.ID, updatePlayground.ID)
	assert.Equal(updateName, updatePlayground.Name)
	assert.Equal(updateDescription, updatePlayground.Description)

	// Verify the updated playground
	updatedPlayground, err := s.GetPlayground(ctx, playground.ID)
	require.NoError(err)
	assert.NotNil(updatedPlayground)
	assert.Equal(playground.ID, updatedPlayground.ID)
	assert.Equal(updateName, updatedPlayground.Name)
	assert.Equal(updateDescription, updatedPlayground.Description)

	err = s.DeletePlayground(ctx, playground.ID)
	require.NoError(err)
}

func TestLinkDatasetToUseCase(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	assert := assert.New(t)
	require := require.New(t)

	s := initializeTest(t)

	// Create a use case request
	req := &client.UseCaseRequest{
		Name:        "Integration Test" + uuid.New().String(),
		Description: "This is a test use case.",
	}

	// Create a use case
	useCase, err := s.CreateUseCase(ctx, req)
	require.NoError(err)
	require.NotNil(useCase)
	require.NotEmpty(useCase.ID)

	// Delete the use case
	defer func() {
		err = s.DeleteUseCase(ctx, useCase.ID)
		require.NoError(err)
	}()

	// Upload file to the dataset
	fileName := "test-linked-to-use.csv"
	content := []byte("value1,value2\n1,2\n")

	dataset, err := s.CreateDatasetFromFile(ctx, fileName, content)
	require.NoError(err)
	require.NotNil(dataset)
	assert.NotEmpty(dataset.ID)
	assert.NotEmpty(dataset.VersionID)
	assert.NotEmpty(dataset.StatusID)

	// Link the dataset to the use case
	err = s.LinkDatasetToUseCase(ctx, useCase.ID, dataset.ID)
	require.NoError(err)

	err = s.DeleteDataset(ctx, dataset.ID)
	require.NoError(err)
}

func TestUseCase(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	assert := assert.New(t)
	require := require.New(t)

	s := initializeTest(t)

	// Create a use case request
	req := &client.UseCaseRequest{
		Name:        "Integration Test" + uuid.New().String(),
		Description: "This is a test use case.",
	}

	// Create a use case
	resp, err := s.CreateUseCase(ctx, req)
	require.NoError(err)
	require.NotNil(resp)
	require.NotEmpty(resp.ID)

	// Delete the use case
	defer func() {
		err = s.DeleteUseCase(ctx, resp.ID)
		require.NoError(err)
	}()

	// Verify the updated use case
	createUseCase, err := s.GetUseCase(ctx, resp.ID)
	require.NoError(err)
	assert.NotNil(createUseCase)
	assert.Equal(resp.ID, createUseCase.ID)
	assert.Equal(req.Name, createUseCase.Name)
	assert.Equal(req.Description, createUseCase.Description)

	// Update the use case request
	updateReq := &client.UseCaseRequest{
		Name:        "Updated Integration Test" + uuid.New().String(),
		Description: "This is an updated test use case.",
	}
	updateResp, err := s.UpdateUseCase(ctx, resp.ID, updateReq)
	require.NoError(err)
	assert.NotNil(updateResp)
	assert.Equal(resp.ID, updateResp.ID)
	assert.Equal(updateReq.Name, updateResp.Name)
	assert.Equal(updateReq.Description, updateResp.Description)

	// Verify the updated use case
	updatedUseCase, err := s.GetUseCase(ctx, resp.ID)
	require.NoError(err)
	assert.NotNil(updatedUseCase)
	assert.Equal(resp.ID, updatedUseCase.ID)
	assert.Equal(updateReq.Name, updatedUseCase.Name)
	assert.Equal(updateReq.Description, updatedUseCase.Description)
}

func TestDatasetFromFile(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	assert := assert.New(t)
	require := require.New(t)

	s := initializeTest(t)

	// Upload file to the dataset
	fileName := "test-from-file.csv"
	content := []byte("value1,value2\n1,2\n")

	resp, err := s.CreateDatasetFromFile(ctx, fileName, content)
	require.NoError(err)
	require.NotNil(resp)
	assert.NotEmpty(resp.ID)
	assert.NotEmpty(resp.VersionID)
	assert.NotEmpty(resp.StatusID)

	// Delete the dataset
	defer func() {
		err = s.DeleteUseCase(ctx, resp.ID)
		require.NoError(err)
	}()

	// Verify the updated dataset
	getDataset, err := s.GetDataset(ctx, resp.ID)
	require.NoError(err)
	require.NotNil(getDataset)
	require.Equal(resp.ID, getDataset.ID)
	require.Equal(resp.VersionID, getDataset.VersionID)
	assert.Equal(fileName, getDataset.Name)

	// wait for the dataset to be processed and timeout after 1 minute
	checkStatus := func() bool {
		status, err := s.IsDatasetReady(ctx, resp.ID)
		require.NoError(err)
		return status
	}
	require.Eventually(checkStatus, 1*time.Minute, 1*time.Second, "data source processing failed")
}

func TestDatasetCreatingVersion(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	assert := assert.New(t)
	require := require.New(t)

	s := initializeTest(t)

	// Create a dataset
	resp, err := s.CreateDataset(ctx, &client.CreateDatasetRequest{
		DoSnapshot: true,
	})
	require.NoError(err)
	require.NotNil(resp)
	require.NotEmpty(resp.ID)

	// Delete the dataset
	defer func() {
		err = s.DeleteDataset(ctx, resp.ID)
		require.NoError(err)
	}()

	// Update the name of the dataset
	updateReq := &client.UpdateDatasetRequest{
		Name: "Integration Test" + uuid.New().String(),
	}

	updateResp, err := s.UpdateDataset(ctx, resp.ID, updateReq)
	require.NoError(err)
	require.NotNil(updateResp)
	assert.NotEmpty(updateResp.ID)
	assert.NotEmpty(updateResp.Name)

	// Verify the updated dataset
	getUpdatedDataset, err := s.GetDataset(ctx, resp.ID)
	require.NoError(err)
	require.NotNil(getUpdatedDataset)
	require.Equal(resp.ID, getUpdatedDataset.ID)
	require.NotEmpty(getUpdatedDataset.VersionID)
	assert.Equal(updateReq.Name, getUpdatedDataset.Name)

	// Upload file to the dataset
	fileName := "test-from-file-version.csv"
	content := []byte("value1,value2\n1,2\n")

	versionResp, err := s.CreateDatasetVersionFromFile(ctx, resp.ID, fileName, content)
	require.NoError(err)
	require.NotNil(versionResp)
	assert.NotEmpty(versionResp.ID)
	assert.NotEmpty(versionResp.VersionID)
	assert.NotEmpty(versionResp.StatusID)

	// wait for the dataset to be processed and timeout after 1 minute
	checkStatus := func() bool {
		status, err := s.IsDatasetReady(ctx, resp.ID)
		require.NoError(err)
		return status
	}
	require.Eventually(checkStatus, 1*time.Minute, 1*time.Second, "data source processing failed")
}

func initializeTest(t *testing.T) client.Service {
	if os.Getenv("ENABLE_INTEGRATION_TESTS") != "true" {
		t.Skip("Integration tests are disabled")
	}
	if os.Getenv("DATAROBOT_API_KEY") == "" {
		t.Fatal("DATAROBOT_API_KEY not set")
	}

	cfg := client.NewConfiguration(os.Getenv("DATAROBOT_API_KEY"))
	cfg.Debug = true
	c := client.NewClient(cfg)

	s := client.NewService(c)

	return s
}
