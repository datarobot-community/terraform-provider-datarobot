package models

import "github.com/hashicorp/terraform-plugin-framework/types"


type DatasetFromURLResourceModel struct {
	ID         types.String   `tfsdk:"id"`
	URL        types.String   `tfsdk:"url"`
	Name       types.String   `tfsdk:"name"`
	UseCaseIDs []types.String `tfsdk:"use_case_ids"`
}


// DatasetFromFileResourceModel describes the dataset uploaded from a local file.
type DatasetFromFileResourceModel struct {
    ID         types.String   `tfsdk:"id"`
    FilePath   types.String   `tfsdk:"file_path"`
    FileHash   types.String   `tfsdk:"file_hash"`
    Name       types.String   `tfsdk:"name"`
    UseCaseIDs []types.String `tfsdk:"use_case_ids"`
}


type DatasetFromDatasourceResourceModel struct {
	ID                        types.String   `tfsdk:"id"`
	DataSourceID              types.String   `tfsdk:"data_source_id"`
	CredentialID              types.String   `tfsdk:"credential_id"`
	DoSnapshot                types.Bool     `tfsdk:"do_snapshot"`
	PersistDataAfterIngestion types.Bool     `tfsdk:"persist_data_after_ingestion"`
	UseKerberos               types.Bool     `tfsdk:"use_kerberos"`
	SampleSizeRows            types.Int64    `tfsdk:"sample_size_rows"`
	Categories                []types.String `tfsdk:"categories"`
	UseCaseIDs                []types.String `tfsdk:"use_case_ids"`
}


// DatastoreResourceModel describes the datastore resource.
type DatastoreResourceModel struct {
	ID            types.String `tfsdk:"id"`
	DataStoreType types.String `tfsdk:"data_store_type"`
	CanonicalName types.String `tfsdk:"canonical_name"`
	DriverID      types.String `tfsdk:"driver_id"`
	JDBCUrl       types.String `tfsdk:"jdbc_url"`
	Fields        []types.Map  `tfsdk:"fields"`
	ConnectorID   types.String `tfsdk:"connector_id"`
}


// DatasourceResourceModel describes the datasource resource.
type DatasourceResourceModel struct {
	ID             types.String          `tfsdk:"id"`
	DataSourceType types.String          `tfsdk:"data_source_type"`
	CanonicalName  types.String          `tfsdk:"canonical_name"`
	Params         DatasourceParamsModel `tfsdk:"params"`
}


type DatasourceParamsModel struct {
	DataStoreID     types.String `tfsdk:"data_store_id"`
	Table           types.String `tfsdk:"table"`
	Schema          types.String `tfsdk:"schema"`
	PartitionColumn types.String `tfsdk:"partition_column"`
	Query           types.String `tfsdk:"query"`
	FetchSize       types.Int64  `tfsdk:"fetch_size"`
	Path            types.String `tfsdk:"path"`
	Catalog         types.String `tfsdk:"catalog"`
}
