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

func TestAccNotificationChannelResource(t *testing.T) {
	t.Parallel()

	if !strings.Contains(os.Getenv(DataRobotEndpointEnvVar), "staging") {
		t.Skip("Skipping custom application from environment test")
	}

	resourceName := "datarobot_notification_channel.test"

	folderPath := "notification_channel"
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

	channelType := "DataRobotUser"
	newChannelType := "DataRobotGroup"

	drEntityID := "66fd35e99a6fbe58dda86733"
	newDrEntityID := "6036d237608973bf082aba1e"

	drEntityName := "nolan.mccafferty@datarobot.com"
	newDrEntityName := "test_group"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: notificationChannelResourceConfig(
					name,
					channelType,
					drEntityID,
					drEntityName,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkNotificationChannelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "channel_type", channelType),
					resource.TestCheckResourceAttr(resourceName, "dr_entities.0.id", drEntityID),
					resource.TestCheckResourceAttr(resourceName, "dr_entities.0.name", drEntityName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "related_entity_id"),
					resource.TestCheckResourceAttrSet(resourceName, "related_entity_type"),
				),
			},
			// Update
			{
				Config: notificationChannelResourceConfig(
					newName,
					newChannelType,
					newDrEntityID,
					newDrEntityName,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkNotificationChannelResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "channel_type", newChannelType),
					resource.TestCheckResourceAttr(resourceName, "dr_entities.0.id", newDrEntityID),
					resource.TestCheckResourceAttr(resourceName, "dr_entities.0.name", newDrEntityName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "related_entity_id"),
					resource.TestCheckResourceAttrSet(resourceName, "related_entity_type"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestNotificationChannelResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewNotificationChannelResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func notificationChannelResourceConfig(
	name,
	channelType,
	drEntityID,
	drEntityName string,
) string {
	return fmt.Sprintf(`
resource "datarobot_custom_model" "test_notification_channel" {
	name = "test deployment"
	description = "test"
	target_type = "Binary"
	target_name = "target"
	base_environment_id = "65f9b27eab986d30d4c64268"
	folder_path = "notification_channel"
}
resource "datarobot_registered_model" "test_notification_channel" {
	name = "test notification channel %s"
	custom_model_version_id = "${datarobot_custom_model.test_notification_channel.version_id}"
}
resource "datarobot_prediction_environment" "test_notification_channel" {
	name = "test notification channel"
	description = "test"
	platform = "datarobotServerless"
}
resource "datarobot_deployment" "test_notification_channel" {
	label = "test notification channel"
	importance = "LOW"
	prediction_environment_id = datarobot_prediction_environment.test_notification_channel.id
	registered_model_version_id = datarobot_registered_model.test_notification_channel.version_id
}
resource "datarobot_notification_channel" "test" {
	name = "%s"
	channel_type = "%s"
	related_entity_id = datarobot_deployment.test_notification_channel.id
	related_entity_type = "deployment"
	dr_entities = [{
		id = "%s"
		name = "%s"
	}]
}
`, nameSalt,
		name,
		channelType,
		drEntityID,
		drEntityName,
	)
}

func checkNotificationChannelResourceExists(resourceName string) resource.TestCheckFunc {
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

		traceAPICall("GetNotificationChannel")
		notificationChannel, err := p.service.GetNotificationChannel(
			context.TODO(),
			rs.Primary.Attributes["related_entity_type"],
			rs.Primary.Attributes["related_entity_id"],
			rs.Primary.ID)
		if err != nil {
			return err
		}

		if notificationChannel.Name == rs.Primary.Attributes["name"] &&
			notificationChannel.ChannelType == rs.Primary.Attributes["channel_type"] &&
			notificationChannel.RelatedEntityID == rs.Primary.Attributes["related_entity_id"] &&
			notificationChannel.RelatedEntityType == rs.Primary.Attributes["related_entity_type"] {
			drEntities := *notificationChannel.DREntities
			if drEntities[0].ID == rs.Primary.Attributes["dr_entities.0.id"] &&
				drEntities[0].Name == rs.Primary.Attributes["dr_entities.0.name"] {
				return nil
			}
		}

		return fmt.Errorf("Notification Channel not found")
	}
}
