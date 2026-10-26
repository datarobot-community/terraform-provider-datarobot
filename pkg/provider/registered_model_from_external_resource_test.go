package provider

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

const registeredModelFromExternalResourceName = "datarobot_registered_model_from_external.test"

func TestAccRegisteredModelFromExternalResource(t *testing.T) {
	t.Parallel()

	resourceName := registeredModelFromExternalResourceName
	idSame := statecheck.CompareValue(compare.ValuesSame())
	idDiffer := statecheck.CompareValue(compare.ValuesDiffer())
	versionIDSame := statecheck.CompareValue(compare.ValuesSame())
	versionIDDiffer := statecheck.CompareValue(compare.ValuesDiffer())
	versionIDKept := statecheck.CompareValue(compare.ValuesSame())

	name := "registered_model_from_external " + nameSalt
	newName := "new_registered_model_from_external " + nameSalt
	useCaseResourceName := "test_registered_model_from_external"
	useCaseResourceName2 := "test_new_registered_model_from_external"

	initial := registeredModelFromExternalTestConfig{
		Name:                    name,
		Description:             "example_description",
		TargetName:              "readmitted",
		PredictionThreshold:     "0.5",
		ModelName:               "external model",
		ModelVersionDescription: "example_version_description",
		ModelLocation:           "s3://bucket/model-v1",
		Tags:                    true,
		UseCaseResourceName:     useCaseResourceName,
	}

	updatedMetadata := initial
	updatedMetadata.Name = newName
	updatedMetadata.Description = "new_example_description"
	updatedMetadata.VersionName = "new_version_name"
	updatedMetadata.ModelVersionDescription = "new_example_version_description"
	updatedMetadata.UseCaseResourceName = useCaseResourceName2

	newVersion := updatedMetadata
	newVersion.PredictionThreshold = "0.6"
	newVersion.ModelLocation = "s3://bucket/model-v2"

	removedDescription := newVersion
	removedDescription.Description = ""

	replaced := newVersion
	replaced.TargetName = "readmitted_new"

	withoutUseCase := replaced
	withoutUseCase.UseCaseResourceName = ""

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: registeredModelFromExternalResourceConfig(initial, true),
				ConfigStateChecks: []statecheck.StateCheck{
					idSame.AddStateValue(resourceName, tfjsonpath.New("id")),
					versionIDSame.AddStateValue(resourceName, tfjsonpath.New("version_id")),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelFromExternalResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttr(resourceName, "version_name", name+" (v1)"),
					resource.TestCheckResourceAttr(resourceName, "target.prediction_threshold", "0.5"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update name, description, version name, version description and use case in place
			{
				Config: registeredModelFromExternalResourceConfig(updatedMetadata, true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					idSame.AddStateValue(resourceName, tfjsonpath.New("id")),
					versionIDSame.AddStateValue(resourceName, tfjsonpath.New("version_id")),
					versionIDDiffer.AddStateValue(resourceName, tfjsonpath.New("version_id")),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelFromExternalResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "version_name", "new_version_name"),
					resource.TestCheckResourceAttr(resourceName, "model_description.description", "new_example_version_description"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
				),
			},
			// Update prediction threshold and model location creates a new version
			{
				Config: registeredModelFromExternalResourceConfig(newVersion, true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					idSame.AddStateValue(resourceName, tfjsonpath.New("id")),
					idDiffer.AddStateValue(resourceName, tfjsonpath.New("id")),
					versionIDDiffer.AddStateValue(resourceName, tfjsonpath.New("version_id")),
					versionIDKept.AddStateValue(resourceName, tfjsonpath.New("version_id")),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelFromExternalResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "version_name", "new_version_name"),
					resource.TestCheckResourceAttr(resourceName, "target.prediction_threshold", "0.6"),
					resource.TestCheckResourceAttr(resourceName, "model_description.location", "s3://bucket/model-v2"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
				),
			},
			// Removing the description clears it in place; the post-apply plan must be empty
			{
				Config: registeredModelFromExternalResourceConfig(removedDescription, true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					idSame.AddStateValue(resourceName, tfjsonpath.New("id")),
					versionIDKept.AddStateValue(resourceName, tfjsonpath.New("version_id")),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// Also checks that the API returns an empty description
					checkRegisteredModelFromExternalResourceExists(),
					resource.TestCheckNoResourceAttr(resourceName, "description"),
				),
			},
			// Update target name replaces the registered model
			{
				Config: registeredModelFromExternalResourceConfig(replaced, true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionReplace),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					idDiffer.AddStateValue(resourceName, tfjsonpath.New("id")),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelFromExternalResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "target.name", "readmitted_new"),
				),
			},
			// Import reads the model package fields back
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// Use Cases are not read back from the API.
				ImportStateVerifyIgnore: []string{
					"use_case_ids",
				},
			},
			// Importing with the configuration plans no changes, apart from the Use Cases
			{
				Config:          registeredModelFromExternalResourceConfig(withoutUseCase, true),
				ResourceName:    resourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
			},
			// Delete is tested automatically
		},
	})
}

func TestRegisteredModelFromExternalResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewRegisteredModelFromExternalResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func TestShouldCreateNewExternalModelVersion(t *testing.T) {
	t.Parallel()

	base := func() RegisteredModelFromExternalResourceModel {
		return RegisteredModelFromExternalResourceModel{
			ID:          types.StringValue("registered-model-id"),
			Name:        types.StringValue("name"),
			VersionName: types.StringValue("version name"),
			Target: &ExternalModelTarget{
				Type:                types.StringValue("Binary"),
				Name:                types.StringValue("target"),
				ClassNames:          []types.String{types.StringValue("yes"), types.StringValue("no")},
				PredictionThreshold: types.Float64Value(0.5),
			},
			ModelDescription: &ExternalModelDescription{
				ModelName:   types.StringValue("model"),
				Description: types.StringValue("version description"),
				Location:    types.StringValue("s3://bucket/model"),
			},
			Tags: types.SetNull(registeredModelTagObjectType()),
		}
	}

	tests := []struct {
		name   string
		change func(plan *RegisteredModelFromExternalResourceModel)
		want   bool
	}{
		{
			name:   "no change",
			change: func(plan *RegisteredModelFromExternalResourceModel) {},
			want:   false,
		},
		{
			name: "registered model name and description are updated in place",
			change: func(plan *RegisteredModelFromExternalResourceModel) {
				plan.Name = types.StringValue("new name")
				plan.Description = types.StringValue("new description")
			},
			want: false,
		},
		{
			name: "version name is updated in place",
			change: func(plan *RegisteredModelFromExternalResourceModel) {
				plan.VersionName = types.StringValue("new version name")
			},
			want: false,
		},
		{
			name: "version description is updated in place",
			change: func(plan *RegisteredModelFromExternalResourceModel) {
				plan.ModelDescription.Description = types.StringValue("new version description")
			},
			want: false,
		},
		{
			name: "removing the version description creates a new version",
			change: func(plan *RegisteredModelFromExternalResourceModel) {
				plan.ModelDescription.Description = types.StringNull()
			},
			want: true,
		},
		{
			name: "prediction threshold creates a new version",
			change: func(plan *RegisteredModelFromExternalResourceModel) {
				plan.Target.PredictionThreshold = types.Float64Value(0.7)
			},
			want: true,
		},
		{
			name: "model location creates a new version",
			change: func(plan *RegisteredModelFromExternalResourceModel) {
				plan.ModelDescription.Location = types.StringValue("s3://bucket/model-v2")
			},
			want: true,
		},
		{
			name: "build environment creates a new version",
			change: func(plan *RegisteredModelFromExternalResourceModel) {
				plan.ModelDescription.BuildEnvironmentType = types.StringValue("Python")
			},
			want: true,
		},
		{
			name: "model id creates a new version",
			change: func(plan *RegisteredModelFromExternalResourceModel) {
				plan.ModelID = types.StringValue("external-model-id")
			},
			want: true,
		},
		{
			name: "datasets create a new version",
			change: func(plan *RegisteredModelFromExternalResourceModel) {
				plan.Datasets = &ExternalModelDatasets{
					TrainingDataCatalogID: types.StringValue("dataset-id"),
				}
			},
			want: true,
		},
		{
			name: "tags create a new version",
			change: func(plan *RegisteredModelFromExternalResourceModel) {
				plan.Tags = types.SetValueMust(registeredModelTagObjectType(), []attr.Value{
					types.ObjectValueMust(registeredModelTagObjectType().AttrTypes, map[string]attr.Value{
						"name":  types.StringValue("team"),
						"value": types.StringValue("mlops"),
					}),
				})
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := base()
			plan := base()
			tt.change(&plan)

			if got := shouldCreateNewExternalModelVersion(state, plan); got != tt.want {
				t.Errorf("shouldCreateNewExternalModelVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldCreateNewExternalModelVersionAddingOnlyVersionDescription(t *testing.T) {
	t.Parallel()

	state := RegisteredModelFromExternalResourceModel{
		Target: &ExternalModelTarget{Type: types.StringValue("Regression"), Name: types.StringValue("target")},
		Tags:   types.SetNull(registeredModelTagObjectType()),
	}
	plan := state
	plan.ModelDescription = &ExternalModelDescription{
		ModelName:            types.StringNull(),
		Description:          types.StringValue("version description"),
		Location:             types.StringNull(),
		BuildEnvironmentType: types.StringNull(),
	}

	if shouldCreateNewExternalModelVersion(state, plan) {
		t.Error("adding only model_description.description should update the version in place")
	}
}

func TestSetExternalModelPackageFromVersion(t *testing.T) {
	t.Parallel()

	t.Run("round trip of every model package field", func(t *testing.T) {
		t.Parallel()

		configured := RegisteredModelFromExternalResourceModel{
			ModelID: types.StringValue("external-model-id"),
			Target: &ExternalModelTarget{
				Type:                          types.StringValue("Multiclass"),
				Name:                          types.StringValue("target"),
				ClassNames:                    []types.String{types.StringValue("a"), types.StringValue("b"), types.StringValue("c")},
				PredictionThreshold:           types.Float64Null(),
				PredictionProbabilitiesColumn: types.StringValue("probabilities"),
			},
			ModelDescription: &ExternalModelDescription{
				ModelName:            types.StringValue("model"),
				Description:          types.StringValue("version description"),
				Location:             types.StringValue("s3://bucket/model"),
				BuildEnvironmentType: types.StringValue("Python"),
			},
			Datasets: &ExternalModelDatasets{
				TrainingDataCatalogID:        types.StringValue("training-id"),
				TrainingDataCatalogVersionID: types.StringValue("training-version-id"),
				HoldoutDataCatalogID:         types.StringValue("holdout-id"),
				HoldoutDataCatalogVersionID:  types.StringNull(),
			},
			Timeseries: &ExternalModelTimeseries{
				DatetimeColumnName:                    types.StringValue("date"),
				DatetimeColumnFormat:                  types.StringValue("%Y-%m-%d"),
				ForecastPointColumnName:               types.StringValue("forecast_point"),
				ForecastDistanceColumnName:            types.StringValue("forecast_distance"),
				ForecastDistancesTimeUnit:             types.StringValue("DAY"),
				ForecastDistances:                     []types.Int64{types.Int64Value(1), types.Int64Value(2)},
				SeriesColumnName:                      types.StringValue("series"),
				FeatureDerivationWindowStart:          types.Int64Value(-28),
				FeatureDerivationWindowEnd:            types.Int64Value(0),
				EffectiveFeatureDerivationWindowStart: types.Int64Null(),
				EffectiveFeatureDerivationWindowEnd:   types.Int64Null(),
				IsCrossSeries:                         types.BoolValue(false),
				IsNewSeriesSupport:                    types.BoolValue(true),
				IsTraditionalTimeSeries:               types.BoolNull(),
			},
			TextGeneration: &ExternalModelTextGeneration{
				Prompt: types.StringValue("prompt"),
			},
			GeospatialMonitoring: &ExternalModelGeospatial{
				PrimaryLocationColumn: types.StringValue("location"),
			},
		}

		version := externalModelPackageResponse(uuid.NewString(), buildExternalModelPackageRequest(configured))
		var imported RegisteredModelFromExternalResourceModel
		var diags diag.Diagnostics
		setExternalModelPackageFromVersion(context.Background(), &imported, &version, &diags)

		if diags.HasError() || diags.WarningsCount() > 0 {
			t.Fatalf("unexpected diagnostics: %+v", diags)
		}
		if !reflect.DeepEqual(imported.ModelID, configured.ModelID) {
			t.Errorf("model_id = %v, want %v", imported.ModelID, configured.ModelID)
		}
		if !reflect.DeepEqual(imported.Target, configured.Target) {
			t.Errorf("target = %+v, want %+v", imported.Target, configured.Target)
		}
		if !reflect.DeepEqual(imported.ModelDescription, configured.ModelDescription) {
			t.Errorf("model_description = %+v, want %+v", imported.ModelDescription, configured.ModelDescription)
		}
		if !reflect.DeepEqual(imported.Datasets, configured.Datasets) {
			t.Errorf("datasets = %+v, want %+v", imported.Datasets, configured.Datasets)
		}
		if !reflect.DeepEqual(imported.Timeseries, configured.Timeseries) {
			t.Errorf("timeseries = %+v, want %+v", imported.Timeseries, configured.Timeseries)
		}
		if !reflect.DeepEqual(imported.TextGeneration, configured.TextGeneration) {
			t.Errorf("text_generation = %+v, want %+v", imported.TextGeneration, configured.TextGeneration)
		}
		if !reflect.DeepEqual(imported.GeospatialMonitoring, configured.GeospatialMonitoring) {
			t.Errorf("geospatial_monitoring = %+v, want %+v", imported.GeospatialMonitoring, configured.GeospatialMonitoring)
		}
	})

	t.Run("values the API fills in are left null", func(t *testing.T) {
		t.Parallel()

		targetName := "target"
		version := externalModelPackageResponse(uuid.NewString(), &client.CreateRegisteredModelFromExternalRequest{
			Name: "version",
			Target: client.ExternalModelPackageTarget{
				Type:       "Binary",
				Name:       &targetName,
				ClassNames: []string{"True", "False"},
			},
		})
		var imported RegisteredModelFromExternalResourceModel
		var diags diag.Diagnostics
		setExternalModelPackageFromVersion(context.Background(), &imported, &version, &diags)

		if !imported.ModelID.IsNull() {
			t.Errorf("model_id = %v, want null", imported.ModelID)
		}
		if want := []types.String{types.StringValue("True"), types.StringValue("False")}; !reflect.DeepEqual(imported.Target.ClassNames, want) {
			t.Errorf("target.class_names = %v, want %v", imported.Target.ClassNames, want)
		}
		if imported.ModelDescription != nil || imported.Datasets != nil || imported.Timeseries != nil ||
			imported.TextGeneration != nil || imported.GeospatialMonitoring != nil {
			t.Errorf("expected no optional model package blocks, got %+v", imported)
		}
	})

	t.Run("prediction threshold matches the configured number exactly", func(t *testing.T) {
		t.Parallel()

		predictionThreshold := 0.6
		version := &client.RegisteredModelVersion{
			ID:      "version-id",
			ModelID: "version-id",
			Target: client.RegisteredModelVersionTarget{
				Type:                "Binary",
				Name:                "target",
				PredictionThreshold: &predictionThreshold,
			},
		}
		var imported RegisteredModelFromExternalResourceModel
		var diags diag.Diagnostics
		setExternalModelPackageFromVersion(context.Background(), &imported, version, &diags)

		// Terraform parses numbers in configuration with 512 bits of precision. A float64 0.6
		// differs from that in its raw value, which plans an update after an import.
		configured, _, err := big.ParseFloat("0.6", 10, 512, big.ToNearestEven)
		if err != nil {
			t.Fatal(err)
		}
		configuredValue := tftypes.NewValue(tftypes.Number, configured)

		naiveThreshold, err := types.Float64Value(predictionThreshold).ToTerraformValue(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if naiveThreshold.Equal(configuredValue) {
			t.Fatal("expected a plain float64 0.6 to differ from the configured 0.6; this test no longer covers the precision difference")
		}

		importedThreshold, err := imported.Target.PredictionThreshold.ToTerraformValue(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if !importedThreshold.Equal(configuredValue) {
			t.Errorf("target.prediction_threshold = %s, want the configured value %s", importedThreshold, configured.Text('g', 20))
		}
	})

	t.Run("truncated class names are not imported", func(t *testing.T) {
		t.Parallel()

		classCount := 150
		version := &client.RegisteredModelVersion{
			ID:      "version-id",
			ModelID: "version-id",
			Target: client.RegisteredModelVersionTarget{
				Type:       "Multiclass",
				Name:       "target",
				ClassNames: []string{"c", "b", "a"},
				ClassCount: &classCount,
			},
		}
		var imported RegisteredModelFromExternalResourceModel
		var diags diag.Diagnostics
		setExternalModelPackageFromVersion(context.Background(), &imported, version, &diags)

		if imported.Target.ClassNames != nil {
			t.Errorf("target.class_names = %v, want null", imported.Target.ClassNames)
		}
		if diags.WarningsCount() != 1 {
			t.Errorf("expected 1 warning, got %+v", diags)
		}
	})
}

func TestIntegrationRegisteredModelFromExternalResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)
	t.Setenv(DataRobotApiKeyEnvVar, "fake")

	registry := newFakeExternalModelRegistry()
	registry.expect(mockService)

	resourceName := registeredModelFromExternalResourceName
	idSame := statecheck.CompareValue(compare.ValuesSame())
	idDiffer := statecheck.CompareValue(compare.ValuesDiffer())
	versionIDSame := statecheck.CompareValue(compare.ValuesSame())
	versionIDDiffer := statecheck.CompareValue(compare.ValuesDiffer())
	versionIDKept := statecheck.CompareValue(compare.ValuesSame())

	name := "registered model " + uuid.NewString()
	newName := "new registered model " + uuid.NewString()

	initial := registeredModelFromExternalTestConfig{
		Name:                    name,
		Description:             "description",
		TargetName:              "target",
		PredictionThreshold:     "0.5",
		ModelName:               "external model",
		ModelVersionDescription: "version description",
		ModelLocation:           "s3://bucket/model-v1",
		Tags:                    true,
	}

	updatedMetadata := initial
	updatedMetadata.Name = newName
	updatedMetadata.Description = "new description"
	updatedMetadata.VersionName = "custom version name"
	updatedMetadata.ModelVersionDescription = "new version description"

	newVersion := updatedMetadata
	newVersion.VersionName = ""
	newVersion.PredictionThreshold = "0.6"

	removedVersionDescription := newVersion
	removedVersionDescription.ModelVersionDescription = ""

	removedDescription := removedVersionDescription
	removedDescription.Description = ""

	replaced := removedVersionDescription
	replaced.TargetName = "new_target"

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create sends the model package and the registered model fields in one request
			{
				Config: registeredModelFromExternalResourceConfig(initial, false),
				ConfigStateChecks: []statecheck.StateCheck{
					idSame.AddStateValue(resourceName, tfjsonpath.New("id")),
					versionIDSame.AddStateValue(resourceName, tfjsonpath.New("version_id")),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "version_name", name+" (v1)"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "1"),
					func(s *terraform.State) error {
						if len(registry.createRequests) != 1 {
							return fmt.Errorf("expected 1 create request, got %d", len(registry.createRequests))
						}
						request := registry.createRequests[0]
						if request.RegisteredModelID != nil {
							return fmt.Errorf("expected no registeredModelId on create, got %q", *request.RegisteredModelID)
						}
						if request.RegisteredModelName == nil || *request.RegisteredModelName != name {
							return fmt.Errorf("expected registeredModelName %q, got %v", name, request.RegisteredModelName)
						}
						if request.RegisteredModelDescription == nil || *request.RegisteredModelDescription != "description" {
							return fmt.Errorf("expected registeredModelDescription %q, got %v", "description", request.RegisteredModelDescription)
						}
						if request.Name != name+" (v1)" {
							return fmt.Errorf("expected version name %q, got %q", name+" (v1)", request.Name)
						}
						if request.Target.Type != "Binary" || *request.Target.Name != "target" ||
							len(request.Target.ClassNames) != 2 || *request.Target.PredictionThreshold != 0.5 {
							return fmt.Errorf("unexpected target %+v", request.Target)
						}
						if request.ModelDescription == nil || *request.ModelDescription.Description != "version description" ||
							*request.ModelDescription.Location != "s3://bucket/model-v1" {
							return fmt.Errorf("unexpected model description %+v", request.ModelDescription)
						}
						if request.Timeseries != nil || request.Datasets != nil {
							return fmt.Errorf("expected no timeseries or datasets, got %+v, %+v", request.Timeseries, request.Datasets)
						}
						if len(request.Tags) != 1 {
							return fmt.Errorf("expected 1 tag, got %d", len(request.Tags))
						}
						return nil
					},
				),
			},
			// Name, description, version name and version description are updated in place
			{
				Config: registeredModelFromExternalResourceConfig(updatedMetadata, false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					idSame.AddStateValue(resourceName, tfjsonpath.New("id")),
					versionIDSame.AddStateValue(resourceName, tfjsonpath.New("version_id")),
					versionIDDiffer.AddStateValue(resourceName, tfjsonpath.New("version_id")),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "description", "new description"),
					resource.TestCheckResourceAttr(resourceName, "version_name", "custom version name"),
					func(s *terraform.State) error {
						if len(registry.createRequests) != 1 {
							return fmt.Errorf("expected no new version, got %d create requests", len(registry.createRequests))
						}
						if len(registry.versionUpdates) != 1 {
							return fmt.Errorf("expected 1 version update, got %d", len(registry.versionUpdates))
						}
						versionUpdate := registry.versionUpdates[0]
						if versionUpdate.Name != "custom version name" || versionUpdate.Description != "new version description" {
							return fmt.Errorf("unexpected version update %+v", versionUpdate)
						}
						return nil
					},
				),
			},
			// Prediction threshold creates a new version, named after the registered model
			// once version_name is removed
			{
				Config: registeredModelFromExternalResourceConfig(newVersion, false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					idSame.AddStateValue(resourceName, tfjsonpath.New("id")),
					versionIDDiffer.AddStateValue(resourceName, tfjsonpath.New("version_id")),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "version_name", newName+" (v2)"),
					resource.TestCheckResourceAttr(resourceName, "target.prediction_threshold", "0.6"),
					func(s *terraform.State) error {
						if len(registry.createRequests) != 2 {
							return fmt.Errorf("expected 2 create requests, got %d", len(registry.createRequests))
						}
						request := registry.createRequests[1]
						if request.RegisteredModelID == nil || *request.RegisteredModelID != s.RootModule().Resources[resourceName].Primary.ID {
							return fmt.Errorf("expected registeredModelId of the existing registered model, got %v", request.RegisteredModelID)
						}
						if request.RegisteredModelName != nil {
							return fmt.Errorf("expected no registeredModelName on a new version, got %q", *request.RegisteredModelName)
						}
						if *request.Target.PredictionThreshold != 0.6 {
							return fmt.Errorf("expected prediction threshold 0.6, got %v", *request.Target.PredictionThreshold)
						}
						if len(registry.versionUpdates) != 1 {
							return fmt.Errorf("expected no further version updates, got %d", len(registry.versionUpdates))
						}
						return nil
					},
				),
			},
			// Removing the version description creates a new version, since PATCH cannot clear it
			{
				Config: registeredModelFromExternalResourceConfig(removedVersionDescription, false),
				ConfigStateChecks: []statecheck.StateCheck{
					idSame.AddStateValue(resourceName, tfjsonpath.New("id")),
					idDiffer.AddStateValue(resourceName, tfjsonpath.New("id")),
					versionIDDiffer.AddStateValue(resourceName, tfjsonpath.New("version_id")),
					versionIDKept.AddStateValue(resourceName, tfjsonpath.New("version_id")),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "version_name", newName+" (v3)"),
					func(s *terraform.State) error {
						if len(registry.createRequests) != 3 {
							return fmt.Errorf("expected 3 create requests, got %d", len(registry.createRequests))
						}
						if description := registry.createRequests[2].ModelDescription.Description; description != nil {
							return fmt.Errorf("expected no version description, got %q", *description)
						}
						return nil
					},
				),
			},
			// Removing the description clears it in place; the post-apply plan must be empty
			{
				Config: registeredModelFromExternalResourceConfig(removedDescription, false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					idSame.AddStateValue(resourceName, tfjsonpath.New("id")),
					versionIDKept.AddStateValue(resourceName, tfjsonpath.New("version_id")),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(resourceName, "description"),
					func(s *terraform.State) error {
						if len(registry.createRequests) != 3 {
							return fmt.Errorf("expected no new version, got %d create requests", len(registry.createRequests))
						}
						if len(registry.modelUpdates) == 0 {
							return fmt.Errorf("expected a registered model update")
						}
						update := registry.modelUpdates[len(registry.modelUpdates)-1]
						if update.Description == nil || *update.Description != "" {
							return fmt.Errorf("expected an empty description in the update, got %v", update.Description)
						}
						if description := registry.models[s.RootModule().Resources[resourceName].Primary.ID].Description; description != "" {
							return fmt.Errorf("expected the description to be cleared, got %q", description)
						}
						return nil
					},
				),
			},
			// Target name replaces the registered model
			{
				Config: registeredModelFromExternalResourceConfig(replaced, false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionReplace),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					idDiffer.AddStateValue(resourceName, tfjsonpath.New("id")),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "target.name", "new_target"),
					resource.TestCheckResourceAttr(resourceName, "version_name", newName+" (v1)"),
					func(s *terraform.State) error {
						if len(registry.createRequests) != 4 {
							return fmt.Errorf("expected 4 create requests, got %d", len(registry.createRequests))
						}
						if registry.createRequests[3].RegisteredModelID != nil {
							return fmt.Errorf("expected a new registered model, got a new version of %q", *registry.createRequests[3].RegisteredModelID)
						}
						if len(registry.deletedIDs) != 1 {
							return fmt.Errorf("expected the previous registered model to be deleted, got %v", registry.deletedIDs)
						}
						return nil
					},
				),
			},
			// Import reads the model package fields back
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Importing with the configuration plans no changes
			{
				Config:          registeredModelFromExternalResourceConfig(replaced, false),
				ResourceName:    resourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
			},
		},
	})
}

func TestRegisteredModelFromExternalResourceRequiresTargetName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)
	defer HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	})()

	mockAPIKey(t)
	t.Setenv(DataRobotApiKeyEnvVar, "fake")

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "datarobot_registered_model_from_external" "test" {
	name   = "registered model"
	target = {
		type = "Regression"
	}
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`target.name is required when target.type is "Regression"`),
			},
			{
				Config: `
resource "datarobot_registered_model_from_external" "test" {
	name   = "registered model"
	target = {
		type = "MCP"
	}
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Training or holdout data needs a target name, whatever the target type
			{
				Config: `
resource "datarobot_registered_model_from_external" "test" {
	name   = "registered model"
	target = {
		type = "MCP"
	}
	datasets = {
		training_data_catalog_id = "training-dataset-id"
	}
}
`,
				PlanOnly:    true,
				ExpectError: planErrorPattern("target.name is required when datasets has training or holdout data"),
			},
			// A dataset ID from another resource is unknown at plan, and still needs a target name
			{
				Config: `
resource "terraform_data" "dataset" {}

resource "datarobot_registered_model_from_external" "test" {
	name   = "registered model"
	target = {
		type = "MCP"
	}
	datasets = {
		holdout_data_catalog_id = terraform_data.dataset.id
	}
}
`,
				PlanOnly:    true,
				ExpectError: planErrorPattern("target.name is required when datasets has training or holdout data"),
			},
			{
				Config: `
resource "datarobot_registered_model_from_external" "test" {
	name   = "registered model"
	target = {
		type = "MCP"
		name = "target"
	}
	datasets = {
		holdout_data_catalog_id = "holdout-dataset-id"
	}
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// AgenticWorkflow targets do not accept datasets, timeseries or geospatial_monitoring
			{
				Config: `
resource "datarobot_registered_model_from_external" "test" {
	name   = "registered model"
	target = {
		type = "AgenticWorkflow"
		name = "target"
	}
	datasets = {
		training_data_catalog_id = "training-dataset-id"
	}
}
`,
				PlanOnly:    true,
				ExpectError: planErrorPattern("not compatible with training or holdout data"),
			},
			{
				Config: `
resource "datarobot_registered_model_from_external" "test" {
	name   = "registered model"
	target = {
		type = "AgenticWorkflow"
	}
	timeseries = {
		datetime_column_name          = "date"
		datetime_column_format        = "%Y-%m-%d"
		forecast_point_column_name    = "forecast_point"
		forecast_distance_column_name = "forecast_distance"
		forecast_distances_time_unit  = "DAY"
	}
}
`,
				PlanOnly:    true,
				ExpectError: planErrorPattern("not compatible with timeseries"),
			},
			{
				Config: `
resource "datarobot_registered_model_from_external" "test" {
	name   = "registered model"
	target = {
		type = "AgenticWorkflow"
	}
	geospatial_monitoring = {
		primary_location_column = "location"
	}
}
`,
				PlanOnly:    true,
				ExpectError: planErrorPattern("not compatible with geospatial_monitoring"),
			},
			// A datasets block without IDs is sent as an empty object, which DataRobot accepts
			{
				Config: `
resource "datarobot_registered_model_from_external" "test" {
	name   = "registered model"
	target = {
		type = "AgenticWorkflow"
	}
	datasets = {}
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// planErrorPattern matches phrase in Terraform's output, which wraps long diagnostics
// across lines.
func planErrorPattern(phrase string) *regexp.Regexp {
	return regexp.MustCompile(strings.ReplaceAll(regexp.QuoteMeta(phrase), " ", `\s+`))
}

type registeredModelFromExternalTestConfig struct {
	Name                    string
	Description             string
	VersionName             string
	TargetName              string
	PredictionThreshold     string
	ModelName               string
	ModelVersionDescription string
	ModelLocation           string
	Tags                    bool
	UseCaseResourceName     string
}

func registeredModelFromExternalResourceConfig(cfg registeredModelFromExternalTestConfig, withUseCases bool) string {
	versionNameStr := ""
	if cfg.VersionName != "" {
		versionNameStr = fmt.Sprintf(`version_name = "%s"`, cfg.VersionName)
	}

	descriptionStr := ""
	if cfg.Description != "" {
		descriptionStr = fmt.Sprintf(`description = "%s"`, cfg.Description)
	}

	versionDescriptionStr := ""
	if cfg.ModelVersionDescription != "" {
		versionDescriptionStr = fmt.Sprintf(`description = "%s"`, cfg.ModelVersionDescription)
	}

	tagsStr := ""
	if cfg.Tags {
		tagsStr = `
	tags = [
		{
			name  = "team"
			value = "mlops"
		},
	]`
	}

	// Both Use Cases stay in the configuration in every step, so that neither is
	// destroyed while the registered model still has versions linked to it.
	useCasesStr := ""
	useCaseIDsStr := ""
	if withUseCases {
		useCasesStr = `
resource "datarobot_use_case" "test_registered_model_from_external" {
	name = "test registered model from external"
}
resource "datarobot_use_case" "test_new_registered_model_from_external" {
	name = "test new registered model from external"
}`
		if cfg.UseCaseResourceName != "" {
			useCaseIDsStr = fmt.Sprintf(`use_case_ids = ["${datarobot_use_case.%s.id}"]`, cfg.UseCaseResourceName)
		}
	}

	return fmt.Sprintf(`
%s
resource "datarobot_registered_model_from_external" "test" {
	name        = "%s"
	%s
	%s
	target = {
		type                 = "Binary"
		name                 = "%s"
		class_names          = ["True", "False"]
		prediction_threshold = %s
	}
	model_description = {
		model_name             = "%s"
		location               = "%s"
		build_environment_type = "Python"
		%s
	}
	%s
	%s
}
`, useCasesStr, cfg.Name, descriptionStr, versionNameStr, cfg.TargetName, cfg.PredictionThreshold,
		cfg.ModelName, cfg.ModelLocation, versionDescriptionStr, useCaseIDsStr, tagsStr)
}

func checkRegisteredModelFromExternalResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[registeredModelFromExternalResourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", registeredModelFromExternalResourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("Provider not found")
		}
		p.service = client.NewService(cl)

		traceAPICall("GetRegisteredModel")
		registeredModel, err := p.service.GetRegisteredModel(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		traceAPICall("GetLatestRegisteredModelVersion")
		latestRegisteredModelVersion, err := p.service.GetLatestRegisteredModelVersion(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if registeredModel.Name != rs.Primary.Attributes["name"] ||
			registeredModel.Description != rs.Primary.Attributes["description"] ||
			latestRegisteredModelVersion.ID != rs.Primary.Attributes["version_id"] ||
			latestRegisteredModelVersion.Name != rs.Primary.Attributes["version_name"] {
			return fmt.Errorf("Registered Model not found")
		}

		if latestRegisteredModelVersion.Target.Name != rs.Primary.Attributes["target.name"] {
			return fmt.Errorf("Target name %q does not match %q", latestRegisteredModelVersion.Target.Name, rs.Primary.Attributes["target.name"])
		}

		if predictionThreshold, ok := rs.Primary.Attributes["target.prediction_threshold"]; ok {
			if latestRegisteredModelVersion.Target.PredictionThreshold == nil ||
				strconv.FormatFloat(*latestRegisteredModelVersion.Target.PredictionThreshold, 'f', -1, 64) != predictionThreshold {
				return fmt.Errorf("Prediction threshold does not match")
			}
		}

		return nil
	}
}

func registeredModelTagObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":  types.StringType,
			"value": types.StringType,
		},
	}
}

// fakeExternalModelRegistry stands in for the Registered Model API in the mock tests,
// recording the requests the resource sends.
type fakeExternalModelRegistry struct {
	models         map[string]*client.RegisteredModel
	versions       map[string][]client.RegisteredModelVersion
	createRequests []client.CreateRegisteredModelFromExternalRequest
	modelUpdates   []client.UpdateRegisteredModelRequest
	versionUpdates []client.UpdateRegisteredModelVersionRequest
	deletedIDs     []string
}

func newFakeExternalModelRegistry() *fakeExternalModelRegistry {
	return &fakeExternalModelRegistry{
		models:   map[string]*client.RegisteredModel{},
		versions: map[string][]client.RegisteredModelVersion{},
	}
}

// expect registers the fake as the mock's implementation. IsRegisteredModelVersionReady
// is deliberately not expected: external model packages have no build to wait for.
func (f *fakeExternalModelRegistry) expect(mockService *mock_client.MockService) {
	mockService.EXPECT().CreateRegisteredModelFromExternal(gomock.Any(), gomock.Any()).DoAndReturn(f.create).AnyTimes()
	mockService.EXPECT().GetRegisteredModel(gomock.Any(), gomock.Any()).DoAndReturn(f.get).AnyTimes()
	mockService.EXPECT().GetLatestRegisteredModelVersion(gomock.Any(), gomock.Any()).DoAndReturn(f.getLatestVersion).AnyTimes()
	mockService.EXPECT().GetRegisteredModelVersion(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(f.getVersion).AnyTimes()
	mockService.EXPECT().UpdateRegisteredModel(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(f.update).AnyTimes()
	mockService.EXPECT().UpdateRegisteredModelVersion(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(f.updateVersion).AnyTimes()
	mockService.EXPECT().DeleteRegisteredModel(gomock.Any(), gomock.Any()).DoAndReturn(f.delete).AnyTimes()
}

func (f *fakeExternalModelRegistry) create(_ context.Context, req *client.CreateRegisteredModelFromExternalRequest) (*client.RegisteredModelVersion, error) {
	f.createRequests = append(f.createRequests, *req)

	var registeredModelID string
	if req.RegisteredModelID != nil {
		registeredModelID = *req.RegisteredModelID
		if _, ok := f.models[registeredModelID]; !ok {
			return nil, &client.NotFoundError{Resource: registeredModelID}
		}
	} else {
		registeredModelID = uuid.NewString()
		registeredModel := &client.RegisteredModel{ID: registeredModelID, Name: *req.RegisteredModelName}
		if req.RegisteredModelDescription != nil {
			registeredModel.Description = *req.RegisteredModelDescription
		}
		f.models[registeredModelID] = registeredModel
	}

	registeredModel := f.models[registeredModelID]
	registeredModel.LastVersionNum++
	version := externalModelPackageResponse(uuid.NewString(), req)
	version.RegisteredModelID = registeredModelID
	version.RegisteredModelVersion = registeredModel.LastVersionNum
	f.versions[registeredModelID] = append(f.versions[registeredModelID], version)
	return &version, nil
}

// externalModelPackageResponse returns the model package the API stores for a create request,
// including the values it fills in for fields the request leaves out.
func externalModelPackageResponse(id string, req *client.CreateRegisteredModelFromExternalRequest) client.RegisteredModelVersion {
	modelID := id
	if req.ModelID != nil {
		modelID = *req.ModelID
	}

	version := client.RegisteredModelVersion{
		ID:      id,
		Name:    req.Name,
		ModelID: modelID,
		Tags:    req.Tags,
		Target: client.RegisteredModelVersionTarget{
			Type:                          req.Target.Type,
			PredictionThreshold:           req.Target.PredictionThreshold,
			PredictionProbabilitiesColumn: req.Target.PredictionProbabilitiesColumn,
		},
		ModelDescription: &client.RegisteredModelVersionModelDescription{
			ModelName:            modelID,
			BuildEnvironmentType: "Other",
		},
		Datasets:             &client.RegisteredModelVersionDatasets{},
		Timeseries:           &client.RegisteredModelVersionTimeseries{},
		GeospatialMonitoring: &client.RegisteredModelVersionGeospatialMonitoring{},
	}

	if req.Target.Name != nil {
		version.Target.Name = *req.Target.Name
	}
	if len(req.Target.ClassNames) > 0 {
		classNames := append([]string(nil), req.Target.ClassNames...)
		if req.Target.Type != "Binary" {
			slices.Reverse(classNames)
		}
		classCount := len(classNames)
		version.Target.ClassNames = classNames
		version.Target.ClassCount = &classCount
	}

	if md := req.ModelDescription; md != nil {
		if md.ModelName != nil {
			version.ModelDescription.ModelName = *md.ModelName
		}
		if md.BuildEnvironmentType != nil {
			version.ModelDescription.BuildEnvironmentType = *md.BuildEnvironmentType
		}
		version.ModelDescription.Description = md.Description
		version.ModelDescription.Location = md.Location
	}

	if ds := req.Datasets; ds != nil {
		version.Datasets = &client.RegisteredModelVersionDatasets{
			TrainingDataCatalogID:        ds.TrainingDataCatalogID,
			TrainingDataCatalogVersionID: ds.TrainingDataCatalogVersionID,
			HoldoutDataCatalogID:         ds.HoldoutDataCatalogID,
			HoldoutDataCatalogVersionID:  ds.HoldoutDataCatalogVersionID,
		}
	}

	if ts := req.Timeseries; ts != nil {
		isNewSeriesSupport, isTraditionalTimeSeries := false, false
		if ts.IsNewSeriesSupport != nil {
			isNewSeriesSupport = *ts.IsNewSeriesSupport
		}
		if ts.IsTraditionalTimeSeries != nil {
			isTraditionalTimeSeries = *ts.IsTraditionalTimeSeries
		}
		version.Timeseries = &client.RegisteredModelVersionTimeseries{
			DatetimeColumnName:                    &ts.DatetimeColumnName,
			DatetimeColumnFormat:                  &ts.DatetimeColumnFormat,
			ForecastPointColumnName:               &ts.ForecastPointColumnName,
			ForecastDistanceColumnName:            &ts.ForecastDistanceColumnName,
			ForecastDistancesTimeUnit:             &ts.ForecastDistancesTimeUnit,
			ForecastDistances:                     ts.ForecastDistances,
			SeriesColumnName:                      ts.SeriesColumnName,
			FeatureDerivationWindowStart:          ts.FeatureDerivationWindowStart,
			FeatureDerivationWindowEnd:            ts.FeatureDerivationWindowEnd,
			EffectiveFeatureDerivationWindowStart: ts.EffectiveFeatureDerivationWindowStart,
			EffectiveFeatureDerivationWindowEnd:   ts.EffectiveFeatureDerivationWindowEnd,
			IsCrossSeries:                         ts.IsCrossSeries,
			IsNewSeriesSupport:                    &isNewSeriesSupport,
			IsTraditionalTimeSeries:               &isTraditionalTimeSeries,
		}
	}

	if req.TextGeneration != nil {
		prompt := req.TextGeneration.Prompt
		version.TextGeneration.Prompt = &prompt
	}

	if req.GeospatialMonitoring != nil {
		primaryLocationColumn := req.GeospatialMonitoring.PrimaryLocationColumn
		version.GeospatialMonitoring.PrimaryLocationColumn = &primaryLocationColumn
	}

	return version
}

func (f *fakeExternalModelRegistry) getVersion(_ context.Context, registeredModelID string, versionID string) (*client.RegisteredModelVersion, error) {
	for _, version := range f.versions[registeredModelID] {
		if version.ID == versionID {
			return &version, nil
		}
	}
	return nil, &client.NotFoundError{Resource: versionID}
}

func (f *fakeExternalModelRegistry) get(_ context.Context, id string) (*client.RegisteredModel, error) {
	registeredModel, ok := f.models[id]
	if !ok {
		return nil, &client.NotFoundError{Resource: id}
	}
	registeredModelCopy := *registeredModel
	return &registeredModelCopy, nil
}

func (f *fakeExternalModelRegistry) getLatestVersion(_ context.Context, id string) (*client.RegisteredModelVersion, error) {
	versions, ok := f.versions[id]
	if !ok || len(versions) == 0 {
		return nil, &client.NotFoundError{Resource: id}
	}
	version := versions[len(versions)-1]
	return &version, nil
}

func (f *fakeExternalModelRegistry) update(_ context.Context, id string, req *client.UpdateRegisteredModelRequest) (*client.RegisteredModel, error) {
	f.modelUpdates = append(f.modelUpdates, *req)
	registeredModel, ok := f.models[id]
	if !ok {
		return nil, &client.NotFoundError{Resource: id}
	}
	if req.Name != "" {
		registeredModel.Name = req.Name
	}
	// Like the API, an empty description clears it and a missing one leaves it unchanged.
	if req.Description != nil {
		registeredModel.Description = *req.Description
	}
	registeredModelCopy := *registeredModel
	return &registeredModelCopy, nil
}

func (f *fakeExternalModelRegistry) updateVersion(_ context.Context, registeredModelID string, versionID string, req *client.UpdateRegisteredModelVersionRequest) (*client.RegisteredModelVersion, error) {
	f.versionUpdates = append(f.versionUpdates, *req)
	for i, version := range f.versions[registeredModelID] {
		if version.ID == versionID {
			if req.Name != "" {
				f.versions[registeredModelID][i].Name = req.Name
			}
			if req.Description != "" {
				description := req.Description
				f.versions[registeredModelID][i].ModelDescription.Description = &description
			}
			updated := f.versions[registeredModelID][i]
			return &updated, nil
		}
	}
	return nil, &client.NotFoundError{Resource: versionID}
}

func (f *fakeExternalModelRegistry) delete(_ context.Context, id string) error {
	f.deletedIDs = append(f.deletedIDs, id)
	delete(f.models, id)
	delete(f.versions, id)
	return nil
}
