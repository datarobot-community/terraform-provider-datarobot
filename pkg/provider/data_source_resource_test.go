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

func TestAccDatasourceResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_datasource.test"

	_, _, bigQueryDriverID, err := GetExternalDataDrivers()
	if err != nil {
		t.Fatalf("Failed to list external data drivers: %v", err)
	}

	s3ConnectorID, _, err := GetExternalConnectors()
	if err != nil {
		t.Fatalf("Failed to list external connectors: %v", err)
	}

	name := "example_datasource " + nameSalt
	newName := "new_example_datasource " + nameSalt

	connectorDatasourceType := "dr-connector-v1"
	databaseDatasourceType := "dr-database-v1"

	tableName := "table_name"
	newTableName := "new_table_name"

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
				Config: datasourceResourceConfig(
					bigQueryDriverID,
					s3ConnectorID,
					name,
					databaseDatasourceType,
					tableName),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatasourceResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "canonical_name", name),
					resource.TestCheckResourceAttr(resourceName, "data_source_type", databaseDatasourceType),
					resource.TestCheckResourceAttrSet(resourceName, "params.data_store_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// update name
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
				Config: datasourceResourceConfig(
					bigQueryDriverID,
					s3ConnectorID,
					newName,
					databaseDatasourceType,
					tableName),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatasourceResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "canonical_name", newName),
					resource.TestCheckResourceAttr(resourceName, "data_source_type", databaseDatasourceType),
					resource.TestCheckResourceAttrSet(resourceName, "params.data_store_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update params triggers replacement
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: datasourceResourceConfig(
					bigQueryDriverID,
					s3ConnectorID,
					newName,
					databaseDatasourceType,
					newTableName),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatasourceResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "canonical_name", newName),
					resource.TestCheckResourceAttr(resourceName, "data_source_type", databaseDatasourceType),
					resource.TestCheckResourceAttrSet(resourceName, "params.data_store_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update type triggers replacement
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: datasourceResourceConfig(
					bigQueryDriverID,
					s3ConnectorID,
					newName,
					connectorDatasourceType,
					newTableName),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatasourceResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "canonical_name", newName),
					resource.TestCheckResourceAttr(resourceName, "data_source_type", connectorDatasourceType),
					resource.TestCheckResourceAttrSet(resourceName, "params.data_store_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestDatasourceResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewDatasourceResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func datasourceResourceConfig(
	bigQueryDriverID,
	s3ConnectorID,
	name,
	datasourceType string,
	tableName string,
) string {
	resourceName := "test_datasource_connector"
	if datasourceType == "dr-database-v1" {
		resourceName = "test_datasource_db"
	}

	params := fmt.Sprintf(`
		catalog = "my-catalog"
		schema = "my-schema"
		table = "%s"
	`, tableName)
	if datasourceType == "dr-connector-v1" {
		params = `
			path = "/path/to-bucket"
		`
	}

	return fmt.Sprintf(`
resource "datarobot_datastore" "test_datasource_db" {
	canonical_name = "test datasource db %s"
	data_store_type = "dr-database-v1"
	driver_id = "%s"
	fields = [
		{
			"id": "bq.project_id",
			"name": "Project Id",
			"value": "my-project"
		}
	]
}
resource "datarobot_datastore" "test_datasource_connector" {
	canonical_name = "test datasource connector %s"
	data_store_type = "dr-connector-v1"
	connector_id = "%s"
	fields = [
		{
			"id": "fs.defaultFS",
			"name": "Bucket Name",
			"value": "my-bucket"
		}
	]
}
resource "datarobot_datasource" "test" {
	canonical_name = "%s"
	data_source_type = "%s"
	params = {
		data_store_id = "${datarobot_datastore.%s.id}"
		%s
	}
}
`, nameSalt, bigQueryDriverID, nameSalt, s3ConnectorID, name, datasourceType, resourceName, params)
}

func checkDatasourceResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["datarobot_datasource.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "datarobot_datasource.test")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("Provider not found")
		}
		p.service = client.NewService(cl)

		traceAPICall("GetDatasource")
		datasource, err := p.service.GetDatasource(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if datasource.CanonicalName == rs.Primary.Attributes["canonical_name"] &&
			datasource.Type == rs.Primary.Attributes["data_source_type"] {
			return nil
		}

		return fmt.Errorf("Datasource not found")
	}
}
