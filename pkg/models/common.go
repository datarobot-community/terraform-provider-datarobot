package models

import "github.com/hashicorp/terraform-plugin-framework/types"

const (
	DataRobotUserNameEnvVar     string = "DATAROBOT_USERNAME"
	DatarobotUserIDEnvVar       string = "DATAROBOT_USER_ID"
	DataRobotApiKeyEnvVar       string = "DATAROBOT_API_TOKEN"
	DataRobotEndpointEnvVar     string = "DATAROBOT_ENDPOINT"
	DataRobotTraceContextEnvVar string = "DATAROBOT_TRACE_CONTEXT"
	TimeoutMinutesEnvVar        string = "DATAROBOT_TIMEOUT_MINUTES"
	UserAgent                   string = "DataRobotTerraformClient"

	PromptRuntimeParameterName string = "PROMPT_COLUMN_NAME"
)

type RuntimeParameterValue struct {
	Key   types.String `json:"key" tfsdk:"key"`
	Type  types.String `json:"type" tfsdk:"type"`
	Value types.String `json:"value" tfsdk:"value"`
}


type FileTuple struct {
	LocalPath   string
	PathInModel string
}
