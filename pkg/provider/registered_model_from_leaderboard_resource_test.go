package provider

import (
	"context"
	"fmt"
	"strconv"
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

	resourceName := "datarobot_registered_model_from_leaderboard.test"
	compareValuesSame := statecheck.CompareValue(compare.ValuesSame())
	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	versionName := "version_name"
	newVersionName := "new_version_name"

	predictionThreshold := "0.6"
	newPredictionThreshold := "0.7"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: registeredModelFromLeaderboardResourceConfig(
					"6706bf087c2049e466c6650b",
					"example_name",
					"example_description",
					&versionName,
					&predictionThreshold),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelFromLeaderboardResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", "example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "example_description"),
					resource.TestCheckResourceAttr(resourceName, "version_name", versionName),
					resource.TestCheckResourceAttr(resourceName, "prediction_threshold", predictionThreshold),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "model_id"),
				),
			},
			// Update name, description, and version name
			{
				Config: registeredModelFromLeaderboardResourceConfig(
					"6706bf087c2049e466c6650b",
					"new_example_name",
					"new_example_description",
					&newVersionName,
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
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelFromLeaderboardResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "version_name", newVersionName),
					resource.TestCheckResourceAttr(resourceName, "prediction_threshold", predictionThreshold),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
					resource.TestCheckResourceAttr(resourceName, "model_id", "6706bf087c2049e466c6650b"),
				),
			},
			// Update model id creates new registered model version
			{
				Config: registeredModelFromLeaderboardResourceConfig(
					"6706bbdb1f1a2176cc114440",
					"new_example_name",
					"new_example_description",
					&newVersionName,
					&predictionThreshold),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelFromLeaderboardResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "version_name", newVersionName),
					resource.TestCheckResourceAttr(resourceName, "prediction_threshold", predictionThreshold),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "model_id", "6706bbdb1f1a2176cc114440"),
					resource.TestCheckResourceAttrSet(resourceName, "version_id"),
				),
			},
			// Update prediction threshold creates new registered model version
			{
				Config: registeredModelFromLeaderboardResourceConfig(
					"6706bbdb1f1a2176cc114440",
					"new_example_name",
					"new_example_description",
					&newVersionName,
					&newPredictionThreshold),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("version_id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRegisteredModelFromLeaderboardResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", "new_example_name"),
					resource.TestCheckResourceAttr(resourceName, "description", "new_example_description"),
					resource.TestCheckResourceAttr(resourceName, "version_name", newVersionName),
					resource.TestCheckResourceAttr(resourceName, "prediction_threshold", newPredictionThreshold),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "model_id", "6706bbdb1f1a2176cc114440"),
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

func registeredModelFromLeaderboardResourceConfig(modelID, name, description string, versionName, predictionThreshold *string) string {
	versionNameStr := ""
	if versionName != nil {
		versionNameStr = `
		version_name = "` + *versionName + `"`
	}

	predictionThresholdStr := ""
	if predictionThreshold != nil {
		predictionThresholdStr = `
		prediction_threshold = "` + *predictionThreshold + `"`
	}

	return fmt.Sprintf(`
resource "datarobot_registered_model_from_leaderboard" "test" {
	name = "%s"
	description = "%s"
	model_id = "%s"
	%s
	%s
}
`, name, description, modelID, versionNameStr, predictionThresholdStr)
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
