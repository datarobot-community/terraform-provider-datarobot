package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccNotificationPolicyResource(t *testing.T) {
	t.Parallel()

	if !strings.Contains(os.Getenv(DataRobotEndpointEnvVar), "staging") {
		t.Skip("Skipping notification policy test")
	}

	resourceName := "datarobot_notification_policy.test"

	folderPath := "notification_policy"
	if err := os.Mkdir(folderPath, 0755); err != nil {
		t.Fatal(err)
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

	name := "example_name"
	newName := "new_example_name"

	eventGroup := "inference_endpoints.health"
	newEventGroup := "batch_predictions.all"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: notificationPolicyResourceConfig(
					name,
					eventGroup,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkNotificationPolicyResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "event_group", eventGroup),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "related_entity_id"),
					resource.TestCheckResourceAttrSet(resourceName, "related_entity_type"),
					resource.TestCheckResourceAttrSet(resourceName, "channel_id"),
					resource.TestCheckResourceAttrSet(resourceName, "channel_scope"),
				),
			},
			// Update
			{
				Config: notificationPolicyResourceConfig(
					newName,
					newEventGroup,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkNotificationPolicyResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "event_group", newEventGroup),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "related_entity_id"),
					resource.TestCheckResourceAttrSet(resourceName, "related_entity_type"),
					resource.TestCheckResourceAttrSet(resourceName, "channel_id"),
					resource.TestCheckResourceAttrSet(resourceName, "channel_scope"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestNotificationPolicyResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewNotificationPolicyResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func notificationPolicyResourceConfig(
	name,
	eventGroup string,
) string {
	return fmt.Sprintf(`
resource "datarobot_custom_model" "test_notification_policy" {
	name = "test deployment"
	description = "test"
	target_type = "Binary"
	target_name = "target"
	base_environment_id = "65f9b27eab986d30d4c64268"
	folder_path = "notification_policy"
}
resource "datarobot_registered_model" "test_notification_policy" {
	name = "test notification policy %s"
	custom_model_version_id = "${datarobot_custom_model.test_notification_policy.version_id}"
}
resource "datarobot_prediction_environment" "test_notification_policy" {
	name = "test notification policy"
	description = "test"
	platform = "datarobotServerless"
}
resource "datarobot_deployment" "test_notification_policy" {
	label = "test notification policy"
	importance = "LOW"
	prediction_environment_id = datarobot_prediction_environment.test_notification_policy.id
	registered_model_version_id = datarobot_registered_model.test_notification_policy.version_id
}
resource "datarobot_notification_channel" "test_notification_policy" {
	name = "test notification policy"
	channel_type = "DataRobotUser"
	related_entity_id = datarobot_deployment.test_notification_policy.id
	related_entity_type = "deployment"
	dr_entities = [{
		id = "66fd35e99a6fbe58dda86733"
		name = "nolan.mccafferty@datarobot.com"
	}]
}
resource "datarobot_notification_policy" "test" {
	name = "%s"
	channel_id          = datarobot_notification_channel.test_notification_policy.id
	channel_scope       = "entity"
	related_entity_id   = datarobot_deployment.test_notification_policy.id
	related_entity_type = "deployment"
	event_group         = "%s"
}
`, nameSalt,
		name,
		eventGroup)
}

func checkNotificationPolicyResourceExists(resourceName string) resource.TestCheckFunc {
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

		traceAPICall("GetNotificationPolicy")
		notificationPolicy, err := p.service.GetNotificationPolicy(
			context.TODO(),
			rs.Primary.Attributes["related_entity_type"],
			rs.Primary.Attributes["related_entity_id"],
			rs.Primary.ID)
		if err != nil {
			return err
		}

		if notificationPolicy.Name == rs.Primary.Attributes["name"] &&
			notificationPolicy.ChannelID == rs.Primary.Attributes["channel_id"] &&
			notificationPolicy.ChannelScope == rs.Primary.Attributes["channel_scope"] &&
			notificationPolicy.RelatedEntityID == rs.Primary.Attributes["related_entity_id"] &&
			notificationPolicy.RelatedEntityType == rs.Primary.Attributes["related_entity_type"] &&
			notificationPolicy.EventGroup.ID == rs.Primary.Attributes["event_group"] {
			return nil
		}

		return fmt.Errorf("Notification Policy not found")
	}
}
