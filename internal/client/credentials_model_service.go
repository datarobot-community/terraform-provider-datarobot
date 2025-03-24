package client

const (
	CredentialTypeBasic                             string = "basic"
	CredentialTypeApiToken                          string = "api_token"
	CredentialTypeS3                                string = "s3"
	CredentialTypeGCP                               string = "gcp"
	CredentialTypeAzure                             string = "azure"
	CredentialTypeAzureServicePrincipal             string = "azure_service_principal"
	CredentialTypeSnowflakeOAuthUserAccount         string = "snowflake_oauth_user_account"
	CredentialTypeADLSGen2OAuth                     string = "adls_gen2_oauth"
	CredentialTypeSnowflakeKeyPairUserAccount       string = "snowflake_key_pair_user_account"
	CredentialTypeDatabricksAccessTokenAccount      string = "databricks_access_token_account"
	CredentialTypeDatabricksServicePrincipalAccount string = "databricks_service_principal_account"
	CredentialTypeSAPOAuth                          string = "sap_oauth"
)

type CredentialRequest struct {
	Name                 string  `json:"name"`
	Description          string  `json:"description,omitempty"`
	CredentialType       string  `json:"credentialType,omitempty"`
	ApiToken             string  `json:"apiToken,omitempty"`
	User                 string  `json:"user,omitempty"`
	Password             string  `json:"password,omitempty"`
	Token                string  `json:"token,omitempty"`
	RefreshToken         string  `json:"refreshToken,omitempty"`
	GCPKey               *GCPKey `json:"gcpKey,omitempty"`
	AWSAccessKeyID       string  `json:"awsAccessKeyId,omitempty"`
	AWSSecretAccessKey   string  `json:"awsSecretAccessKey,omitempty"`
	AWSSessionToken      string  `json:"awsSessionToken,omitempty"`
	ConfigID             string  `json:"configId,omitempty"`
	AzureConnectionString string  `json:"azureConnectionString,omitempty"`
}

type GCPKey struct {
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url,omitempty"`
	AuthURI                 string `json:"auth_uri,omitempty"`
	ClientEmail             string `json:"client_email,omitempty"`
	ClientID                string `json:"client_id,omitempty"`
	ClientX509CertURL       string `json:"client_x509_cert_url,omitempty"`
	PrivateKey              string `json:"private_key,omitempty"`
	PrivateKeyID            string `json:"private_key_id,omitempty"`
	ProjectID               string `json:"project_id,omitempty"`
	TokenURI                string `json:"token_uri,omitempty"`
	Type                    string `json:"type,omitempty"`
	UniverseDomain          string `json:"universe_domain,omitempty"`
}

type Credential struct {
	ID             string `json:"credentialId"`
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	CredentialType string `json:"credentialType,omitempty"`
	ConfigID       string `json:"configId,omitempty"`
}

type CredentialAssociation struct {
	ObjectID   string `json:"objectId"`
	ObjectType string `json:"objectType"`
	Link       string `json:"link"`
	IsDefault  bool   `json:"isDefault"`
}
