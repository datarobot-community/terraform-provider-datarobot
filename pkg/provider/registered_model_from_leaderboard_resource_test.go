package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccRegisteredModelFromLeaderboardResource(t *testing.T) {
	t.Parallel()
	t.Skip("Skipping TestAccRegisteredModelFromLeaderboardResource until we can get a model id that works in all environments")
	modelID := "673b722dfd279fd86944d088"
	modelID2 := "673b6fd8e060b90658aebe66"
	if strings.Contains(globalTestCfg.Endpoint, "staging") {
		modelID = "673b75ec97f1021bbfb61d3b"
		modelID2 = "673b75ec97f1021bbfb61d34"
	} else if strings.Contains(globalTestCfg.Endpoint, "dr-app-charts") {
		t.Skip("Skipping registered model from leaderboard test for environment")
	}

	resourceName := "datarobot_registered_model_from_leaderboard.test"
	compareValuesSame := statecheck.CompareValue(compare.ValuesSame())
	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	name := "registered_model_from_leaderboard " + nameSalt
	newName := "new_registered_model_from_leaderboard " + nameSalt

	versionName := "version_name"
	newVersionName := "new_version_name"

	predictionThreshold := "0.6"
	newPredictionThreshold := "0.7"

	useCaseResourceName := "test_registered_model_from_leaderboard"
	useCaseResourceName2 := "test_new_registered_model_from_leaderboard"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: registeredModelFromLeaderboardResourceConfig(
					modelID,
					name,
					"example_description",
					&versionName,
					&useCaseResourceName,
					&predictionThreshold),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("use_case_ids"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelFromLeaderboardResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttr(resourceName, "version_name", versionName),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttr(resourceName, "prediction_threshold", predictionThreshold),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "model_id"),
				),
			},
			// Update name, description, version name, and use case id
			{
				Config: registeredModelFromLeaderboardResourceConfig(
					modelID,
					newName,
					"new_example_description",
					&newVersionName,
					&useCaseResourceName2,
					&predictionThreshold),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("use_case_ids"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelFromLeaderboardResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "version_name", newVersionName),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttr(resourceName, "prediction_threshold", predictionThreshold),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttr(resourceName, "model_id", modelID),
				),
			},
			// Update model id creates new registered model version
			// and remove use case id
			{
				Config: registeredModelFromLeaderboardResourceConfig(
					modelID2,
					newName,
					"new_example_description",
					&newVersionName,
					nil,
					&predictionThreshold),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelFromLeaderboardResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "version_name", newVersionName),
					resource.TestCheckNoResourceAttr(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttr(resourceName, "prediction_threshold", predictionThreshold),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "model_id", modelID2),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update prediction threshold creates new registered model version
			{
				Config: registeredModelFromLeaderboardResourceConfig(
					modelID2,
					newName,
					"new_example_description",
					nil,
					nil,
					&newPredictionThreshold),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelFromLeaderboardResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttrSet(resourceName, "version_name"),
					resource.TestCheckNoResourceAttr(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttr(resourceName, "prediction_threshold", newPredictionThreshold),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "model_id", modelID2),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestRegisteredModelFromLeaderboardResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewRegisteredModelResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func registeredModelFromLeaderboardResourceConfig(modelID, name, description string, versionName, useCaseResourceName, predictionThreshold *string) string {
	versionNameStr := ""
	if versionName != nil {
		versionNameStr = `
		version_name = "` + *versionName + `"`
	}

	useCaseIDsStr := ""
	if useCaseResourceName != nil {
		useCaseIDsStr = fmt.Sprintf(`use_case_ids = ["${datarobot_use_case.%s.id}"]`, *useCaseResourceName)
	}

	predictionThresholdStr := ""
	if predictionThreshold != nil {
		predictionThresholdStr = `
		prediction_threshold = "` + *predictionThreshold + `"`
	}

	return fmt.Sprintf(`
resource "datarobot_use_case" "test_registered_model_from_leaderboard" {
	name = "test registered model from leaderboard"
}
resource "datarobot_use_case" "test_new_registered_model_from_leaderboard" {
	name = "test new registered model from leaderboard"
}
resource "datarobot_registered_model_from_leaderboard" "test" {
	name = "%s"
	description = "%s"
	model_id = "%s"
	%s
	%s
	%s
}
`, name, description, modelID, versionNameStr, useCaseIDsStr, predictionThresholdStr)
}

func checkRegisteredModelFromLeaderboardResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["datarobot_registered_model_from_leaderboard.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "datarobot_registered_model_from_leaderboard.test")
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

		traceAPICall("ListRegisteredModelVersions")
		latestRegisteredModelVersion, err := p.service.GetLatestRegisteredModelVersion(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if registeredModel.Name == rs.Primary.Attributes["name"] &&
			registeredModel.Description == rs.Primary.Attributes["description"] &&
			latestRegisteredModelVersion.ID == rs.Primary.Attributes["version_id"] &&
			latestRegisteredModelVersion.Name == rs.Primary.Attributes["version_name"] &&
			latestRegisteredModelVersion.ModelID == rs.Primary.Attributes["model_id"] {
			if predictionThreshold, ok := rs.Primary.Attributes["prediction_threshold"]; ok {
				if strconv.FormatFloat(*latestRegisteredModelVersion.Target.PredictionThreshold, 'f', -1, 64) != predictionThreshold {
					return fmt.Errorf("Prediction threshold does not match")
				}
			}

			return nil
		}

		return fmt.Errorf("Registered Model not found")
	}
}
