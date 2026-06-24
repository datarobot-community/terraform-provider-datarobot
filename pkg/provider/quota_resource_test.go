package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccQuotaResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_quota.test"

	// id must stay stable when only default_rules change (update in place),
	// and change when the governed resource changes (RequiresReplace).
	compareValuesSame := statecheck.CompareValue(compare.ValuesSame())

	folderPath, err := prepareTestFolder("quota")
	if err != nil {
		t.Fatalf("Failed to create test folder: %v", err)
	}
	defer os.RemoveAll(folderPath)

	modelContents := `from typing import Any, Dict
import pandas as pd

def load_model(code_dir: str) -> Any:
	return "dummy"

def score(data: pd.DataFrame, model: Any, **kwargs: Dict[str, Any]) -> pd.DataFrame:
	positive_label = kwargs["positive_class_label"]
	negative_label = kwargs["negative_class_label"]
	preds = pd.DataFrame([[0.75, 0.25]] * data.shape[0], columns=[positive_label, negative_label])
	return preds
`
	if err := os.WriteFile(folderPath+"/custom.py", []byte(modelContents), 0644); err != nil {
		t.Fatal(err)
	}

	modelMetadata := `name: quota-test

type: inference
targetType: binary
inferenceModel:
  targetName: target
  positiveClassLabel: 1
  negativeClassLabel: 0
`
	if err := os.WriteFile(folderPath+"/model-metadata.yaml", []byte(modelMetadata), 0644); err != nil {
		t.Fatal(err)
	}

	twoRules := `
		{
			rule   = "requests"
			limit  = 750
			window = "day"
		},
		{
			rule   = "token"
			limit  = 100000
			window = "hour"
		},`

	oneRule := `
		{
			rule   = "requests"
			limit  = 1000
			window = "day"
		},`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: quotaResourceConfig(twoRules),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(resourceName, tfjsonpath.New("id")),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkQuotaResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "resource_type", "deployment"),
					resource.TestCheckResourceAttrSet(resourceName, "resource_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "default_rules.#", "2"),
				),
			},
			// Update default_rules in place (id must stay the same)
			{
				Config: quotaResourceConfig(oneRule),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(resourceName, tfjsonpath.New("id")),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkQuotaResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "default_rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "default_rules.0.rule", "requests"),
					resource.TestCheckResourceAttr(resourceName, "default_rules.0.limit", "1000"),
					resource.TestCheckResourceAttr(resourceName, "default_rules.0.window", "day"),
				),
			},
			// Import by the governed resource id (the deployment id)
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources[resourceName]
					if !ok {
						return "", fmt.Errorf("Not found: %s", resourceName)
					}
					return rs.Primary.Attributes["resource_id"], nil
				},
			},
			// Delete is tested automatically
		},
	})
}

func quotaResourceConfig(rules string) string {
	return fmt.Sprintf(`
resource "datarobot_custom_model" "test_quota" {
	name = "test quota %s"
	description = "test"
	target_type = "Binary"
	target_name = "target"
	base_environment_id = "`+testGenAIBaseEnvID+`"
	folder_path = "quota"
}
resource "datarobot_registered_model" "test_quota" {
	name = "test quota %s"
	description = "test"
	custom_model_version_id = "${datarobot_custom_model.test_quota.version_id}"
}
resource "datarobot_prediction_environment" "test_quota" {
	name = "test quota %s"
	description = "test"
	platform = "datarobotServerless"
}
resource "datarobot_deployment" "test_quota" {
	label = "test quota %s"
	importance = "MODERATE"
	prediction_environment_id = datarobot_prediction_environment.test_quota.id
	registered_model_version_id = datarobot_registered_model.test_quota.version_id
}
resource "datarobot_quota" "test" {
	resource_id = datarobot_deployment.test_quota.id
	default_rules = [%s
	]
}
`, nameSalt, nameSalt, nameSalt, nameSalt, rules)
}

func checkQuotaResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["datarobot_quota.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "datarobot_quota.test")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("Provider not found")
		}
		p.service = client.NewService(cl)

		traceAPICall("GetQuotaForResource")
		quota, err := p.service.GetQuotaForResource(
			context.TODO(),
			rs.Primary.Attributes["resource_type"],
			rs.Primary.Attributes["resource_id"],
		)
		if err != nil {
			return err
		}

		if quota.ID != rs.Primary.ID {
			return fmt.Errorf("Quota ID mismatch: api=%s state=%s", quota.ID, rs.Primary.ID)
		}

		return nil
	}
}
