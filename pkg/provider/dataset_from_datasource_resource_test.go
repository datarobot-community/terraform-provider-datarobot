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

func TestAccDatasetFromDatasourceResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_dataset_from_datasource.test"

	dataSource, credential, err := GetDemoDataSource()
	if err != nil {
		t.Skip("Demo data source not found")
	}

	useCase := "test_dataset_from_datasource"
	newUseCase := "test_new_dataset_from_datasource"

	category := "TRAINING"
	sampleSizeRows := 10

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
				Config: datasetFromDatasourceResourceConfig(
					dataSource.ID,
					credential.ID,
					true,
					&sampleSizeRows,
					nil,
					&useCase),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatasetFromDatasourceResourceExists(),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "data_source_id", dataSource.ID),
					resource.TestCheckResourceAttr(resourceName, "credential_id", credential.ID),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttr(resourceName, "do_snapshot", "true"),
					resource.TestCheckResourceAttr(resourceName, "persist_data_after_ingestion", "true"),
					resource.TestCheckResourceAttr(resourceName, "use_kerberos", "false"),
					resource.TestCheckResourceAttr(resourceName, "sample_size_rows", "10"),
					resource.TestCheckNoResourceAttr(resourceName, "categories"),
				),
			},
			// Update use case IDs and categories
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
				Config: datasetFromDatasourceResourceConfig(
					dataSource.ID,
					credential.ID,
					true,
					&sampleSizeRows,
					&category,
					&newUseCase),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatasetFromDatasourceResourceExists(),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "data_source_id", dataSource.ID),
					resource.TestCheckResourceAttr(resourceName, "credential_id", credential.ID),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttr(resourceName, "do_snapshot", "true"),
					resource.TestCheckResourceAttr(resourceName, "do_snapshot", "true"),
					resource.TestCheckResourceAttr(resourceName, "persist_data_after_ingestion", "true"),
					resource.TestCheckResourceAttr(resourceName, "use_kerberos", "false"),
					resource.TestCheckResourceAttr(resourceName, "sample_size_rows", "10"),
					resource.TestCheckResourceAttr(resourceName, "categories.0", category),
				),
			},
			// Update dataset creation settings triggers replace
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: datasetFromDatasourceResourceConfig(
					dataSource.ID,
					credential.ID,
					false,
					nil,
					&category,
					&newUseCase),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDatasetFromDatasourceResourceExists(),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "data_source_id", dataSource.ID),
					resource.TestCheckResourceAttr(resourceName, "credential_id", credential.ID),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
					resource.TestCheckResourceAttr(resourceName, "do_snapshot", "false"),
					resource.TestCheckResourceAttr(resourceName, "persist_data_after_ingestion", "true"),
					resource.TestCheckResourceAttr(resourceName, "use_kerberos", "false"),
					resource.TestCheckNoResourceAttr(resourceName, "sample_size_rows"),
					resource.TestCheckResourceAttr(resourceName, "categories.0", category),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestDatasetFromDatasourceResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewDatasetFromDatasourceResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func datasetFromDatasourceResourceConfig(
	dataSourceID string,
	credentialID string,
	doSnapshot bool,
	sampleSizeRows *int,
	category *string,
	useCaseID *string) string {
	useCaseIDsStr := ""
	if useCaseID != nil {
		useCaseIDsStr = fmt.Sprintf(`use_case_ids = ["${datarobot_use_case.%s.id}"]`, *useCaseID)
	}

	sampleSizeRowsStr := ""
	if sampleSizeRows != nil {
		sampleSizeRowsStr = fmt.Sprintf(`sample_size_rows = %d`, *sampleSizeRows)
	}

	categoriesStr := ""
	if category != nil {
		categoriesStr = fmt.Sprintf(`categories = ["%s"]`, *category)
	}

	return fmt.Sprintf(`
resource "datarobot_use_case" "test_dataset_from_datasource" {
	  name = "test"
}
resource "datarobot_use_case" "test_new_dataset_from_datasource" {
	  name = "test 2"
}
resource "datarobot_dataset_from_datasource" "test" {
	data_source_id = "%s"
	credential_id = "%s"
	do_snapshot = %t
	%s
	%s
	%s
}
`, dataSourceID, credentialID, doSnapshot, useCaseIDsStr, sampleSizeRowsStr, categoriesStr)
}

func checkDatasetFromDatasourceResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["datarobot_dataset_from_datasource.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "datarobot_dataset_from_datasource.test")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("Provider not found")
		}
		p.service = client.NewService(cl)

		traceAPICall("GetDataset")
		dataset, err := p.service.GetDataset(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if dataset.DataSourceID == rs.Primary.Attributes["data_source_id"] {
			return nil
		}

		return fmt.Errorf("Dataset not found")
	}
}

func GetDemoDataSource() (ds client.Datasource, credential client.Credential, err error) {
	dataSources, err := client.NewService(cl).ListDatasources(context.Background(), &client.ListDataSourcesRequest{
		Type: "all",
	})
	if err != nil {
		return
	}

	for _, dataSource := range dataSources {
		if dataSource.CanonicalName == "FLIGHT_DELAYS_DATA" {
			ds = dataSource
			var credentials []client.Credential
			credentials, err = client.NewService(cl).ListDatastoreCredentials(context.Background(), ds.Params.DataStoreID)
			if err != nil {
				return
			}
			credential = credentials[0]
			return
		}
	}
	return
}
