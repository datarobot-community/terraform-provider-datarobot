package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccQAApplicationResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_qa_application.test"

	name := "qa_application " + nameSalt
	newName := "new_qa_application " + nameSalt

	folderPath := "qa_application"
	if err := os.Mkdir(folderPath, 0755); err != nil {
		t.Fatal(err)
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

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: qaApplicationResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkQAApplicationResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_version_id"),
				),
			},
			// Update name
			{
				Config: qaApplicationResourceConfig(newName),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkQAApplicationResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_version_id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func qaApplicationResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "datarobot_custom_model" "test_qa_application" {
	name = "test qa application"
	target_type = "TextGeneration"
	target_name = "target"
	base_environment_id = "65f9b27eab986d30d4c64268"
	folder_path = "qa_application"
}
resource "datarobot_registered_model" "test_qa_application" {
	name = "test Q&A application %s"
	description = "test"
	custom_model_version_id = "${datarobot_custom_model.test_qa_application.version_id}"
}
resource "datarobot_prediction_environment" "test_qa_application" {
	name = "test Q&A application"
	description = "test"
	platform = "aws"
}
resource "datarobot_deployment" "test_qa_application" {
	label = "test Q&A application"
	prediction_environment_id = datarobot_prediction_environment.test_qa_application.id
	registered_model_version_id = datarobot_registered_model.test_qa_application.version_id
}
resource "datarobot_qa_application" "test" {
	name = "%s"
	deployment_id = datarobot_deployment.test_qa_application.id
  }
`, nameSalt, name)
}

func checkQAApplicationResourceExists(resourceName string) resource.TestCheckFunc {
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

		traceAPICall("GetApplication")
		application, err := p.service.GetApplication(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if application.Name == rs.Primary.Attributes["name"] &&
			application.ApplicationUrl == rs.Primary.Attributes["application_url"] &&
			application.CustomApplicationSourceID == rs.Primary.Attributes["source_id"] &&
			application.CustomApplicationSourceVersionID == rs.Primary.Attributes["source_version_id"] {
			return nil
		}

		return fmt.Errorf("Q&A Application not found")
	}
}
