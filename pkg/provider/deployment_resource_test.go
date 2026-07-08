package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccDeploymentResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_deployment.test"
	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	useCaseResourceName := "test_deployment"
	useCaseResourceName2 := "test_new_deployment"

	folderPath, err := prepareTestFolder("deployment")
	if err != nil {
		t.Fatalf("Failed to create test folder: %v", err)
	}
	defer os.RemoveAll(folderPath)

	modelContentsTemplate := `from typing import Any, Dict
import pandas as pd

def load_model(code_dir: str) -> Any:
	return "%s"

def score(data: pd.DataFrame, model: Any, **kwargs: Dict[str, Any]) -> pd.DataFrame:
	positive_label = kwargs["positive_class_label"]
	negative_label = kwargs["negative_class_label"]
	preds = pd.DataFrame([[0.75, 0.25]] * data.shape[0], columns=[positive_label, negative_label])
	return preds
`

	if err := os.WriteFile(folderPath+"/custom.py", []byte(fmt.Sprintf(modelContentsTemplate, "dummy")), 0644); err != nil {
		t.Fatal(err)
	}

	fileContents := `name: runtime-params

type: inference
targetType: binary
inferenceModel:
  targetName: target
  positiveClassLabel: 1
  negativeClassLabel: 0
runtimeParameterDefinitions:
  - fieldName: STRING_PARAMETER
    type: string
    description: An example of a string parameter
    defaultValue: null
  - fieldName: BOOLEAN_PARAMETER
    type: boolean
    description: An example of a boolean parameter
    defaultValue: null`

	if err := os.WriteFile(folderPath+"/"+"model-metadata.yaml", []byte(fileContents), 0644); err != nil {
		t.Fatal(err)
	}

	resourceBundleID := "cpu.micro"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: deploymentResourceConfig(
					"example_label",
					"MODERATE",
					"target",
					&useCaseResourceName,
					false,
					false,
					false,
					false,
					false,
					false,
					nil,
					false,
					"value", nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("use_case_ids"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDeploymentResourceExists(),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttr(resourceName, "label", "example_label"),
					resource.TestCheckResourceAttr(resourceName, "importance", "MODERATE"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttr(resourceName, "runtime_parameter_values.0.value", "value"),
					resource.TestCheckNoResourceAttr(resourceName, "predictions_by_forecast_date_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "challenger_models_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "segment_analysis_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "bias_and_fairness_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "challenger_replay_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "drift_tracking_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "association_id_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "predictions_data_collection_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "prediction_warning_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "prediction_intervals_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "health_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "retraining_settings"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update label, importance, settings, runtime param values and use case id
			{
				Config: deploymentResourceConfig(
					"new_example_label",
					"LOW",
					"target",
					&useCaseResourceName2,
					true,
					true,
					true,
					true,
					true,
					true,
					&resourceBundleID,
					true,
					"newValue",
					&RetrainingSettings{
						PredictionEnvironmentID: types.StringValue("${datarobot_prediction_environment.test_deployment.id}"),
					},
				),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("use_case_ids"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDeploymentResourceExists(),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttr(resourceName, "label", "new_example_label"),
					resource.TestCheckResourceAttr(resourceName, "importance", "LOW"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttr(resourceName, "runtime_parameter_values.0.value", "newValue"),
					resource.TestCheckResourceAttr(resourceName, "predictions_by_forecast_date_settings.enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "segment_analysis_settings.enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "segment_analysis_settings.enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "challenger_replay_settings.enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "predictions_data_collection_settings.enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "predictions_settings.resource_bundle_id", resourceBundleID),
					resource.TestCheckResourceAttr(resourceName, "health_settings.service.batch_count", "5"),
					resource.TestCheckResourceAttr(resourceName, "health_settings.data_drift.batch_count", "5"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					checkRetrainingSettingsUpdate(),
				),
			},
			// Remove settings and use case id
			{
				Config: deploymentResourceConfig(
					"new_example_label",
					"LOW",
					"target",
					nil,
					false,
					false,
					false,
					false,
					false,
					false,
					nil,
					false,
					"",
					nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("registered_model_version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDeploymentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "label", "new_example_label"),
					resource.TestCheckResourceAttr(resourceName, "importance", "LOW"),
					resource.TestCheckNoResourceAttr(resourceName, "use_case_ids.0"),
					resource.TestCheckNoResourceAttr(resourceName, "predictions_by_forecast_date_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "challenger_models_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "segment_analysis_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "bias_and_fairness_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "challenger_replay_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "drift_tracking_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "association_id_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "predictions_data_collection_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "prediction_warning_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "prediction_intervals_settings"),
					resource.TestCheckNoResourceAttr(resourceName, "health_settings"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Try to update target_name of Custom Model (should fail)
			{
				Config: deploymentResourceConfig(
					"new_example_label",
					"LOW",
					"new_target",
					nil,
					false,
					false,
					false,
					false,
					false,
					false,
					nil,
					false,
					"", nil),
				ExpectError: regexp.MustCompile(`target_name cannot be changed if the model was deployed.`),
			},
			// Update custom model version (by updating the file contents) updates registered model version of deployment
			// which triggers a model replacement for the Deployment
			{
				PreConfig: func() {
					if err := os.WriteFile(folderPath+"/custom.py", []byte(fmt.Sprintf(modelContentsTemplate, "dummy2")), 0644); err != nil {
						t.Fatal(err)
					}
				},
				Config: deploymentResourceConfig(
					"new_example_label",
					"LOW",
					"target",
					nil,
					false,
					false,
					false,
					false,
					false,
					false,
					nil,
					false,
					"value", nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("registered_model_version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDeploymentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "label", "new_example_label"),
					resource.TestCheckResourceAttr(resourceName, "importance", "LOW"),
					resource.TestCheckResourceAttr(resourceName, "runtime_parameter_values.0.value", "value"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestDeploymentResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewDeploymentResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func deploymentResourceConfig(
	label,
	importance string,
	customModelTargetName string,
	useCaseResourceName *string,
	isPredictionsByForecastDateEnabled,
	isSegmentAnalysisEnabled,
	isChallengerReplayEnabled,
	isAssociationIDEnabled,
	isPredictionsDataCollectionEnabled,
	isPredictionsSettingsEnabled bool,
	resourceBundleID *string,
	isHealthSettingsEnabled bool,
	runtimeParameterValue string,
	retrainingSettings *RetrainingSettings,
) string {
	useCaseIDsStr := ""
	if useCaseResourceName != nil {
		useCaseIDsStr = fmt.Sprintf(`use_case_ids = ["${datarobot_use_case.%s.id}"]`, *useCaseResourceName)
	}

	deploymentSettings := ""

	if isPredictionsByForecastDateEnabled {
		deploymentSettings = `
	predictions_by_forecast_date_settings = {
		enabled = true
		column_name = "column_name"
		datetime_format = "%H:%M"
	}`
	}

	if isSegmentAnalysisEnabled {
		deploymentSettings += `
	segment_analysis_settings = {
		enabled = true
	}`
	}

	if isChallengerReplayEnabled {
		deploymentSettings += `
	challenger_replay_settings = {
		enabled = true
	}`
	}

	if isAssociationIDEnabled {
		deploymentSettings += `
	association_id_settings = {
		auto_generate_id = true
		column_names = ["example_column"]
		required_in_prediction_requests = true
	}`
	}

	if isPredictionsDataCollectionEnabled {
		deploymentSettings += `
	predictions_data_collection_settings = {
		enabled = true
	}`
	}

	if isPredictionsSettingsEnabled {
		if resourceBundleID != nil {
			deploymentSettings += fmt.Sprintf(`
		predictions_settings = {
			resource_bundle_id = "%s"
		}`, *resourceBundleID)
		} else {
			deploymentSettings += `
		predictions_settings = {
			min_computes = 0
			max_computes = 2
		}`
		}
	}

	if isHealthSettingsEnabled {
		deploymentSettings += `
	health_settings = {
		service = {
			batch_count = 5
		}
		data_drift = {
			time_interval = "P7D"
			batch_count = 5
			drift_threshold = 0.2
			importance_threshold = 0.3
			low_importance_warning_count = 2
			low_importance_failing_count = 5
			high_importance_warning_count = 2
			high_importance_failing_count = 5
		}
		accuracy = {
			batch_count = 1000
		}
		prediction_timeliness = {
			enabled = true
			expected_frequency = "P7D"
		}
		actuals_timeliness = {
			enabled = true
			expected_frequency = "P30D"
		}
	}`
	}
	if retrainingSettings != nil {
		retrainingSettingsStr := `
		retraining_settings = {`

		if retrainingSettings.PredictionEnvironmentID.ValueString() != "" {
			retrainingSettingsStr += fmt.Sprintf(`
			prediction_environment_id = "%s"`, retrainingSettings.PredictionEnvironmentID.ValueString())
		}

		if retrainingSettings.RetrainingUserID.ValueString() != "" {
			retrainingSettingsStr += fmt.Sprintf(`
			retraining_user_id = "%s"`, retrainingSettings.RetrainingUserID.ValueString())
		}

		if retrainingSettings.DatasetID.ValueString() != "" {
			retrainingSettingsStr += fmt.Sprintf(`
			dataset_id = "%s"`, retrainingSettings.DatasetID.ValueString())
		}

		if retrainingSettings.CredentialID.ValueString() != "" {
			retrainingSettingsStr += fmt.Sprintf(`
			credential_id = "%s"`, retrainingSettings.CredentialID.ValueString())
		}

		retrainingSettingsStr += `
		}`

		deploymentSettings += retrainingSettingsStr
	}

	runtimeParameterValuesStr := ""
	if runtimeParameterValue != "" {
		runtimeParameterValuesStr = fmt.Sprintf(`
	runtime_parameter_values = [
		{
			key="STRING_PARAMETER",
			type="string",
			value="%s"
		},
	]`, runtimeParameterValue)
	}

	return fmt.Sprintf(`
resource "datarobot_use_case" "test_deployment" {
	name = "test deployment %s"
}
resource "datarobot_use_case" "test_new_deployment" {
	name = "test new deployment %s"
}
resource "datarobot_custom_model" "test_deployment" {
	name = "test deployment %s"
	description = "test"
	target_type = "Binary"
	target_name = "%s"
	base_environment_id = "`+testGenAIBaseEnvID+`"
	folder_path = "deployment"
}
resource "datarobot_registered_model" "test_deployment" {
	name = "test deployment %s"
	description = "test"
	custom_model_version_id = "${datarobot_custom_model.test_deployment.version_id}"
}
resource "datarobot_prediction_environment" "test_deployment" {
	name = "test deployment %s"
	description = "test"
	platform = "datarobotServerless"
}
resource "datarobot_deployment" "test" {
	label = "%s"
	importance = "%s"
	prediction_environment_id = datarobot_prediction_environment.test_deployment.id
	registered_model_version_id = datarobot_registered_model.test_deployment.version_id
	%s
	%s
	%s
}
`, nameSalt, nameSalt, nameSalt, customModelTargetName, nameSalt, nameSalt, label, importance, useCaseIDsStr, deploymentSettings, runtimeParameterValuesStr)
}

func checkDeploymentResourceExists() resource.TestCheckFunc {
	resourceName := "datarobot_deployment.test"
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("Provider not found")
		}
		p.service = client.NewService(cl)

		traceAPICall("GetDeployment")
		deployment, err := p.service.GetDeployment(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if deployment.Label == rs.Primary.Attributes["label"] &&
			deployment.ModelPackage.ID == rs.Primary.Attributes["registered_model_version_id"] &&
			deployment.PredictionEnvironment.ID == rs.Primary.Attributes["prediction_environment_id"] {
			return nil
		}

		return fmt.Errorf("Deployment not found")
	}
}

func checkRetrainingSettingsUpdate() resource.TestCheckFunc {
	resourceName := "datarobot_deployment.test"

	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("Provider not found")
		}
		p.service = client.NewService(cl)

		traceAPICall("GetDeploymentRetrainingSettings")
		retrainingSettings, err := p.service.GetDeploymentRetrainingSettings(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if retrainingSettings == nil {
			return fmt.Errorf("Retraining settings not found")
		}

		expectedPredictionEnvironmentID := rs.Primary.Attributes["retraining_settings.prediction_environment_id"]

		if retrainingSettings.PredictionEnvironment.ID != expectedPredictionEnvironmentID {

			return fmt.Errorf("Expected prediction environment ID %s, got %s", expectedPredictionEnvironmentID, retrainingSettings.PredictionEnvironment.ID)
		}

		return nil
	}
}

// TestUnitDeploymentModelReplacementSuccess verifies that model replacement succeeds when the
// backend confirms the deployment is serving the expected model package after replacement.
func TestUnitDeploymentModelReplacementSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		globalTestCfg.ApiKey = "fake"
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	depID := uuid.NewString()
	predEnvID := uuid.NewString()
	oldPkgID := uuid.NewString()
	newPkgID := uuid.NewString()
	createTaskID := uuid.NewString()
	updateTaskID := uuid.NewString()

	oldDeployment := &client.Deployment{
		ID:     depID,
		Label:  "test-label",
		Status: "active",
		ModelPackage: client.ModelPackage{
			ID: oldPkgID,
		},
		PredictionEnvironment: client.PredictionEnvironment{
			ID: predEnvID,
		},
		Importance: "MODERATE",
	}
	newDeployment := &client.Deployment{
		ID:     depID,
		Label:  "test-label",
		Status: "active",
		ModelPackage: client.ModelPackage{
			ID: newPkgID,
		},
		PredictionEnvironment: client.PredictionEnvironment{
			ID: predEnvID,
		},
		Importance: "MODERATE",
	}

	// GetDeployment returns oldDeployment until model replacement completes, then newDeployment.
	replacementApplied := false
	mockService.EXPECT().
		GetDeployment(gomock.Any(), depID).
		DoAndReturn(func(_ context.Context, _ string) (*client.Deployment, error) {
			if replacementApplied {
				return newDeployment, nil
			}
			return oldDeployment, nil
		}).
		AnyTimes()

	// Create
	mockService.EXPECT().
		CreateDeploymentFromModelPackage(gomock.Any(), gomock.Any()).
		Return(&client.DeploymentCreateResponse{ID: depID}, createTaskID, nil)
	mockService.EXPECT().
		GetTaskStatus(gomock.Any(), createTaskID).
		Return(&client.TaskStatusResponse{Status: "COMPLETED"}, nil)
	mockService.EXPECT().
		UpdateDeploymentSettings(gomock.Any(), depID, gomock.Any()).
		Return(&client.DeploymentSettings{}, nil)

	// Update: model replacement succeeds and backend confirms new package
	mockService.EXPECT().
		UpdateDeployment(gomock.Any(), depID, gomock.Any()).
		Return(oldDeployment, nil)
	mockService.EXPECT().
		ValidateDeploymentModelReplacement(gomock.Any(), depID, gomock.Any()).
		Return(&client.ValidateDeployemntModelReplacementResponse{Status: "passing"}, nil)
	mockService.EXPECT().
		UpdateDeploymentModel(gomock.Any(), depID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ *client.UpdateDeploymentModelRequest) (*client.Deployment, string, error) {
			replacementApplied = true
			return nil, updateTaskID, nil
		})
	mockService.EXPECT().
		GetTaskStatus(gomock.Any(), updateTaskID).
		Return(&client.TaskStatusResponse{Status: "COMPLETED"}, nil)
	mockService.EXPECT().
		UpdateDeploymentSettings(gomock.Any(), depID, gomock.Any()).
		Return(&client.DeploymentSettings{}, nil)

	// Destroy
	mockService.EXPECT().DeleteDeployment(gomock.Any(), depID).Return(nil)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: deploymentModelReplacementConfig(predEnvID, oldPkgID),
			},
			{
				Config: deploymentModelReplacementConfig(predEnvID, newPkgID),
				Check: resource.TestCheckResourceAttr(
					"datarobot_deployment.test_replacement",
					"registered_model_version_id",
					newPkgID,
				),
			},
		},
	})
}

// TestUnitDeploymentModelReplacementMismatch verifies that model replacement fails with a clear
// error when the deployment is active but still serving the old model package after replacement.
func TestUnitDeploymentModelReplacementMismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	if globalTestCfg.ApiKey == "" {
		globalTestCfg.ApiKey = "fake"
		t.Setenv(DataRobotApiKeyEnvVar, "fake")
	}

	depID := uuid.NewString()
	predEnvID := uuid.NewString()
	oldPkgID := uuid.NewString()
	newPkgID := uuid.NewString()
	createTaskID := uuid.NewString()
	updateTaskID := uuid.NewString()

	// The backend stays on oldDeployment even after replacement (simulates silent rollback).
	oldDeployment := &client.Deployment{
		ID:     depID,
		Label:  "test-label",
		Status: "active",
		ModelPackage: client.ModelPackage{
			ID: oldPkgID,
		},
		PredictionEnvironment: client.PredictionEnvironment{
			ID: predEnvID,
		},
		Importance: "MODERATE",
	}

	mockService.EXPECT().
		GetDeployment(gomock.Any(), depID).
		Return(oldDeployment, nil).
		AnyTimes()

	// Create
	mockService.EXPECT().
		CreateDeploymentFromModelPackage(gomock.Any(), gomock.Any()).
		Return(&client.DeploymentCreateResponse{ID: depID}, createTaskID, nil)
	mockService.EXPECT().
		GetTaskStatus(gomock.Any(), createTaskID).
		Return(&client.TaskStatusResponse{Status: "COMPLETED"}, nil)
	mockService.EXPECT().
		UpdateDeploymentSettings(gomock.Any(), depID, gomock.Any()).
		Return(&client.DeploymentSettings{}, nil)

	// Update: task completes but backend still serves old package → provider returns error
	mockService.EXPECT().
		UpdateDeployment(gomock.Any(), depID, gomock.Any()).
		Return(oldDeployment, nil)
	mockService.EXPECT().
		ValidateDeploymentModelReplacement(gomock.Any(), depID, gomock.Any()).
		Return(&client.ValidateDeployemntModelReplacementResponse{Status: "passing"}, nil)
	mockService.EXPECT().
		UpdateDeploymentModel(gomock.Any(), depID, gomock.Any()).
		Return(nil, updateTaskID, nil)
	mockService.EXPECT().
		GetTaskStatus(gomock.Any(), updateTaskID).
		Return(&client.TaskStatusResponse{Status: "COMPLETED"}, nil)
	// No UpdateDeploymentSettings: update returns error before reaching that point

	// Destroy (cleanup after failed update step)
	mockService.EXPECT().DeleteDeployment(gomock.Any(), depID).Return(nil)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: deploymentModelReplacementConfig(predEnvID, oldPkgID),
			},
			{
				Config:      deploymentModelReplacementConfig(predEnvID, newPkgID),
				ExpectError: regexp.MustCompile("Deployment model replacement did not apply"),
			},
		},
	})
}

func deploymentModelReplacementConfig(predEnvID, pkgID string) string {
	return fmt.Sprintf(`
resource "datarobot_deployment" "test_replacement" {
	label                       = "test-label"
	importance                  = "MODERATE"
	registered_model_version_id = %q
	prediction_environment_id   = %q
}
`, pkgID, predEnvID)
}
