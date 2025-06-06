package client

type CreateDatasetRequest struct {
	DoSnapshot bool `json:"doSnapshot"`
}

type CreateDatasetFromURLRequest struct {
	URL                       string             `json:"url"`
	DoSnapshot                *bool              `json:"doSnapshot,omitempty"`
	PersistDataAfterIngestion *bool              `json:"persistDataAfterIngestion,omitempty"`
	SampleSize                *DatasetSampleSize `json:"sampleSize,omitempty"`
}

type CreateDatasetFromDatasourceRequest struct {
	CredentialID              string             `json:"credentialId"`
	DataSourceID              string             `json:"dataSourceId"`
	DoSnapshot                *bool              `json:"doSnapshot"`
	PersistDataAfterIngestion *bool              `json:"persistDataAfterIngestion"`
	UseKerberos               *bool              `json:"useKerberos"`
	SampleSize                *DatasetSampleSize `json:"sampleSize,omitempty"`
	Categories                []string           `json:"categories,omitempty"`
}

type DatasetSampleSize struct {
	Type  string `json:"type"`
	Value int64  `json:"value"`
}

type CreateDatasetResponse struct {
	ID string `json:"datasetId"`
}

type CreateDatasetVersionResponse struct {
	// The ID of the catalog entry.
	ID string `json:"catalogId"`
	// The ID of the latest version of the catalog entry.
	VersionID string `json:"catalogVersionId"`
	// ID that can be used with GET /api/v2/status/{statusId}/ to poll for the testing job's status.
	StatusID string `json:"statusId"`
}

type UpdateDatasetRequest struct {
	Name       string    `json:"name,omitempty"`
	Categories *[]string `json:"categories,omitempty"`
}

type Dataset struct {
	ID                                    string            `json:"datasetId"`
	Name                                  string            `json:"name"`
	VersionID                             string            `json:"versionId"`
	DataSourceType                        string            `json:"dataSourceType"`
	DataSourceID                          string            `json:"dataSourceId"`
	ProcessingState                       string            `json:"processingState"`
	Error                                 *string           `json:"error"`
	IsSnapshot                            bool              `json:"isSnapshot"`
	IsVectorDatabaseEligible              bool              `json:"isVectorDatabaseEligible"`
	IsDataEngineEligible                  bool              `json:"isDataEngineEligible"`
	IsWranglingEligible                   bool              `json:"isWranglingEligible"`
	IsPlaygroundEvaluationDatasetEligible bool              `json:"isPlaygroundEvaluationDatasetEligible"`
	Uri                                   string            `json:"uri"`
	SampleSize                            DatasetSampleSize `json:"sampleSize"`
	DataPersisted                         bool              `json:"dataPersisted"`
	Categories                            []string          `json:"categories"`
}

type CreateDatastoreRequest struct {
	Type          string          `json:"type"`
	CanonicalName string          `json:"canonicalName"`
	Params        DatastoreParams `json:"params"`
}

type DatastoreParams struct {
	Fields      []map[string]string `json:"fields,omitempty"`
	JDBCFields  []map[string]string `json:"jdbcFields,omitempty"`
	DriverID    *string             `json:"driverId,omitempty"`
	JDBCUrl     *string             `json:"jdbcUrl,omitempty"`
	ConnectorID *string             `json:"connectorId,omitempty"`
}

type Datastore struct {
	ID            string          `json:"id"`
	CanonicalName string          `json:"canonicalName"`
	Type          string          `json:"type"`
	Params        DatastoreParams `json:"params"`
}

type UpdateDatastoreRequest struct {
	CanonicalName string          `json:"canonicalName"`
	Params        DatastoreParams `json:"params"`
}

type ListExternalDataDriversRequest struct {
	Type string `json:"type" url:"type"`
}

type ExternalDataDriver struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	CanonicalName   string `json:"canonicalName"`
	Version         string `json:"version"`
	DatabaseDriver  string `json:"databaseDriver"`
	ConfigurationID string `json:"configurationId"`
	ClassName       string `json:"className"`
}

type ExternalConnector struct {
	ID              string `json:"id"`
	Version         string `json:"version"`
	ConnectorType   string `json:"connectorType"`
	CanonicalName   string `json:"canonicalName"`
	ConfigurationID string `json:"configurationId"`
}

type TestDatastoreConnectionRequest struct {
	CredentialID string `json:"credentialId"`
}

type TestDatastoreConnectionResponse struct {
	Message string `json:"message"`
}

type CreateDatasourceRequest struct {
	CanonicalName string           `json:"canonicalName"`
	Params        DatasourceParams `json:"params"`
	Type          string           `json:"type"`
}

type DatasourceParams struct {
	DataStoreID     string  `json:"dataStoreId"`
	Catalog         *string `json:"catalog,omitempty"`
	FetchSize       *int64  `json:"fetchSize,omitempty"`
	PartitionColumn *string `json:"partitionColumn,omitempty"`
	Schema          *string `json:"schema,omitempty"`
	Table           *string `json:"table,omitempty"`
	Query           *string `json:"query,omitempty"`
	Path            *string `json:"path,omitempty"`
}

type Datasource struct {
	ID            string           `json:"id"`
	CanonicalName string           `json:"canonicalName"`
	Params        DatasourceParams `json:"params"`
	Type          string           `json:"type"`
}

type UpdateDatasourceRequest struct {
	CanonicalName string `json:"canonicalName"`
}

type ListDataSourcesRequest struct {
	Type string `json:"type" url:"type"`
}
