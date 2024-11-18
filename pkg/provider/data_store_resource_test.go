package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccDatastoreResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_datastore.test"

	name := "example_datastore"
	newName := "new_example_datastore"

	connectorDatastoreType := "dr-connector-v1"
	connectorID := "65538041dde6a1d664d0b2ec"
	newConnectorID := "65d4a92347c41afc8bec5a4d"
	connectorFields := `[
			{
				"id": "fs.defaultFS",
				"name": "Bucket Name",
				"value": "my-bucket"
			}
		]
	`
	connectorFieldsNewBucket := `[
			{
				"id": "fs.defaultFS",
				"name": "Bucket Name",
				"value": "my-new-bucket"
			}
		]
	`
	newConnectorFields := `[
            {
                "id": "fs.adls.gen2.accountName",
                "name": "Azure Storage Account Name",
                "value": "account_name"
            }
    	]
	`

	jdbcType := "jdbc"
	driverID := "5b4752844bf542000175dbea"
	newDriverID := "63ff328b24f812dfdab4c194"
	driverFields := `[
			{
				"name": "address",
				"value": "my-address"
			},
			{
				"name": "database",
				"value": "my-database"
			}	
		]
	`
	newDriverFields := `[
			{
				"name": "address",
				"value": "my-new-address"
			},
			{
				"name": "AwsRegion",
				"value": "us-east-1"
			},
			{
				"name": "S3OutputLocation",
				"value": "location"
			}
		]
	`

	jdbcUrl := "jdbc:awsathena://.test"

	databaseType := "dr-database-v1"
	databaseDriverID := "64a288a50636598d75df7f82"
	databaseFields := `[
		{
			"id": "bq.project_id",
			"name": "Project Id",
			"value": "project-id"
		}
	]`

	compareValuesSame := statecheck.CompareValue(compare.ValuesSame())
	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: datastoreResourceConfig(
					name,
					connectorDatastoreType,
					&connectorID,
					nil,
					nil,
					connectorFields),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatastoreResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "canonical_name", name),
					resource.TestCheckResourceAttr(resourceName, "data_store_type", connectorDatastoreType),
					resource.TestCheckResourceAttr(resourceName, "connector_id", connectorID),
					resource.TestCheckResourceAttr(resourceName, "fields.0.id", "fs.defaultFS"),
					resource.TestCheckResourceAttr(resourceName, "fields.0.value", "my-bucket"),
					resource.TestCheckNoResourceAttr(resourceName, "driver_id"),
					resource.TestCheckNoResourceAttr(resourceName, "jdbc_url"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// update name and fields
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: datastoreResourceConfig(
					newName,
					connectorDatastoreType,
					&connectorID,
					nil,
					nil,
					connectorFieldsNewBucket),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatastoreResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "canonical_name", newName),
					resource.TestCheckResourceAttr(resourceName, "data_store_type", connectorDatastoreType),
					resource.TestCheckResourceAttr(resourceName, "connector_id", connectorID),
					resource.TestCheckResourceAttr(resourceName, "fields.0.value", "my-new-bucket"),
					resource.TestCheckNoResourceAttr(resourceName, "driver_id"),
					resource.TestCheckNoResourceAttr(resourceName, "jdbc_url"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update connector_id triggers replacement
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: datastoreResourceConfig(
					newName,
					connectorDatastoreType,
					&newConnectorID,
					nil,
					nil,
					newConnectorFields),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatastoreResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "canonical_name", newName),
					resource.TestCheckResourceAttr(resourceName, "data_store_type", connectorDatastoreType),
					resource.TestCheckResourceAttr(resourceName, "connector_id", newConnectorID),
					resource.TestCheckResourceAttr(resourceName, "fields.0.id", "fs.adls.gen2.accountName"),
					resource.TestCheckResourceAttr(resourceName, "fields.0.value", "account_name"),
					resource.TestCheckNoResourceAttr(resourceName, "driver_id"),
					resource.TestCheckNoResourceAttr(resourceName, "jdbc_url"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update data_source_type triggers replacement
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: datastoreResourceConfig(
					newName,
					jdbcType,
					nil,
					&driverID,
					nil,
					driverFields),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatastoreResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "canonical_name", newName),
					resource.TestCheckResourceAttr(resourceName, "data_store_type", jdbcType),
					resource.TestCheckNoResourceAttr(resourceName, "connector_id"),
					resource.TestCheckResourceAttr(resourceName, "fields.0.name", "address"),
					resource.TestCheckResourceAttr(resourceName, "fields.0.value", "my-address"),
					resource.TestCheckResourceAttr(resourceName, "fields.1.name", "database"),
					resource.TestCheckResourceAttr(resourceName, "fields.1.value", "my-database"),
					resource.TestCheckResourceAttr(resourceName, "driver_id", driverID),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update driver_id triggers replacement
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: datastoreResourceConfig(
					newName,
					jdbcType,
					nil,
					&newDriverID,
					nil,
					newDriverFields),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatastoreResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "canonical_name", newName),
					resource.TestCheckResourceAttr(resourceName, "data_store_type", jdbcType),
					resource.TestCheckNoResourceAttr(resourceName, "connector_id"),
					resource.TestCheckResourceAttr(resourceName, "fields.0.name", "address"),
					resource.TestCheckResourceAttr(resourceName, "fields.0.value", "my-new-address"),
					resource.TestCheckResourceAttr(resourceName, "fields.1.name", "AwsRegion"),
					resource.TestCheckResourceAttr(resourceName, "fields.1.value", "us-east-1"),
					resource.TestCheckResourceAttr(resourceName, "fields.2.name", "S3OutputLocation"),
					resource.TestCheckResourceAttr(resourceName, "fields.2.value", "location"),
					resource.TestCheckResourceAttr(resourceName, "driver_id", newDriverID),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update jdbc_url triggers replacement
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: datastoreResourceConfig(
					newName,
					jdbcType,
					nil,
					&newDriverID,
					&jdbcUrl,
					"[]"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatastoreResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "canonical_name", newName),
					resource.TestCheckResourceAttr(resourceName, "data_store_type", jdbcType),
					resource.TestCheckNoResourceAttr(resourceName, "connector_id"),
					resource.TestCheckNoResourceAttr(resourceName, "fields.0"),
					resource.TestCheckResourceAttr(resourceName, "driver_id", newDriverID),
					resource.TestCheckResourceAttr(resourceName, "jdbc_url", jdbcUrl),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update data_source_type to database triggers replacement
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: datastoreResourceConfig(
					newName,
					databaseType,
					nil,
					&databaseDriverID,
					nil,
					databaseFields),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatastoreResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "canonical_name", newName),
					resource.TestCheckResourceAttr(resourceName, "data_store_type", databaseType),
					resource.TestCheckNoResourceAttr(resourceName, "connector_id"),
					resource.TestCheckNoResourceAttr(resourceName, "jdbc_url"),
					resource.TestCheckResourceAttr(resourceName, "fields.0.id", "bq.project_id"),
					resource.TestCheckResourceAttr(resourceName, "fields.0.name", "Project Id"),
					resource.TestCheckResourceAttr(resourceName, "fields.0.value", "project-id"),
					resource.TestCheckResourceAttr(resourceName, "driver_id", databaseDriverID),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestDatastoreResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewDatastoreResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func datastoreResourceConfig(
	name,
	datastoreType string,
	connectorID,
	driverID,
	jdbcUrl *string,
	fields string,
) string {
	connectorIDStr := ""
	if connectorID != nil {
		connectorIDStr = fmt.Sprintf(`connector_id = "%s"`, *connectorID)
	}

	driverIDStr := ""
	if driverID != nil {
		driverIDStr = fmt.Sprintf(`driver_id = "%s"`, *driverID)
	}

	jdbcUrlStr := ""
	if jdbcUrl != nil {
		jdbcUrlStr = fmt.Sprintf(`jdbc_url = "%s"`, *jdbcUrl)
	}

	return fmt.Sprintf(`
resource "datarobot_datastore" "test" {
	canonical_name = "%s"
	data_store_type = "%s"
	%s
	%s
	%s
	fields = %v
}
`, name, datastoreType, connectorIDStr, driverIDStr, jdbcUrlStr, fields)
}

func checkDatastoreResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["datarobot_datastore.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "datarobot_datastore.test")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("Provider not found")
		}
		p.service = client.NewService(cl)

		traceAPICall("GetDatastore")
		datastore, err := p.service.GetDatastore(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if datastore.CanonicalName == rs.Primary.Attributes["canonical_name"] &&
			datastore.Type == rs.Primary.Attributes["data_store_type"] {
			if datastore.Params.ConnectorID != nil {
				if *datastore.Params.ConnectorID == rs.Primary.Attributes["connector_id"] {
					return nil
				}
			}
			if datastore.Params.DriverID != nil {
				if *datastore.Params.DriverID == rs.Primary.Attributes["driver_id"] {
					return nil
				}
			}
		}

		return fmt.Errorf("Datastore not found")
	}
}
