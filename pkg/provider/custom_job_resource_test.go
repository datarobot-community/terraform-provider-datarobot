package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwpath "github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccCustomJobResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_custom_job.test"

	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())
	compareValuesSame := statecheck.CompareValue(compare.ValuesSame())

	name := "custom_job " + nameSalt
	newName := "new_custom_job " + nameSalt

	description := "example_description"
	newDescription := "new_example_description"

	folderPath := "custom_job"
	err := os.Mkdir(folderPath, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(folderPath)

	runScript := `#!/bin/bash

	echo "Job Starting: ($0)"

	echo "===== Runtime Parameters ======"
	echo "Model Package:     $MODEL_PACKAGE"
	echo "Deployment:        $DEPLOYMENT"
	echo "STRING_PARAMETER:  $STRING_PARAMETER"
	echo
	echo
	echo "===== Generic Variables ==========================="
	echo "CURRENT_CUSTOM_JOB_RUN_ID: $CURRENT_CUSTOM_JOB_RUN_ID"
	echo "CURRENT_CUSTOM_JOB_ID:     $CURRENT_CUSTOM_JOB_ID"
	echo "DATAROBOT_ENDPOINT:        $DATAROBOT_ENDPOINT"
	echo "DATAROBOT_API_TOKEN:       Use the environment variable $DATAROBOT_API_TOKEN"
	echo "==================================================="

	echo
	echo "How to check how much memory your job has"
	  memory_limit_bytes=$(cat /sys/fs/cgroup/memory/memory.limit_in_bytes)
	  memory_limit_megabytes=$((memory_limit_bytes / 1024 / 1024))
	echo "Memory Limit (in Megabytes): $memory_limit_megabytes"
	echo

	# Uncomment the following if you want to check if the job has network access
	## Define the IP address of an external server to ping (e.g., Google's DNS)
	#external_server="8.8.8.8"
	#echo "Checking internet connection"
	## Try to ping the external server
	#ping -c 1 $external_server > /dev/null 2>&1
	#
	## Check the exit status of the ping command
	#if [ $? -eq 0 ]; then
	#    echo "Internet connection is available."
	#else
	#    echo "No internet connection."
	#fi
	#echo
	#echo

	# Run the code in job.py
	dir_path=$(dirname $0)
	echo "Entrypoint is at $dir_path - cd into it"
	cd $dir_path

	if command -v python3 &>/dev/null; then
		echo "python3 is installed and available."
	else
		echo "Error: python3 is not installed or not available."
		exit 1
	fi

	python_file="job.py"
	if [ -f "$python_file" ]; then
		echo "Found $python_file .. running it"
		python3 ./job.py
	else
		echo "File $python_file does not exist"
		exit 1
	fi`

	err = os.WriteFile(folderPath+"/run.sh", []byte(runScript), 0644)
	if err != nil {
		t.Fatal(err)
	}

	jobCode := `import os
import datarobot as dr
from datarobot import Deployment

def main():
    print(f"Running python code: {__file__}")

    # Using this job runtime parameters
    print()
    print("Runtime parameters:")
    print("-------------------")
    string_param = os.environ.get("STRING_PARAMETER", None)
    print(f"string param: {string_param}")

    deployment_param = os.environ.get("DEPLOYMENT", None)
    print(f"deployment_param: {deployment_param}")

    retraining_policy_param = os.environ.get("RETRAINING_POLICY_ID", None)
    print(f"retraining_policy_param: {retraining_policy_param}")

    model_package_param = os.environ.get("MODEL_PACKAGE", None)
    print(f"model_package_param: {model_package_param}")

    # An example of using the python client to list deployments
    deployments = Deployment.list()
    print()
    print("List of all deployments")
    print("-----------------------")
    for deployment in deployments:
        print(deployment)

if __name__ == "__main__":
    main()`

	err = os.WriteFile(folderPath+"/job.py", []byte(jobCode), 0644)
	if err != nil {
		t.Fatal(err)
	}

	metadataFileName := "metadata.yaml"
	metadataFileContents := `name: runtime-params

runtimeParameterDefinitions:
  - fieldName: DEPLOYMENT
    type: deployment
    description: Deployment that will be used for retraining job
  - fieldName: RETRAINING_POLICY_ID
    type: string
    description: Retraining policy ID
  - fieldName: OPENAI_API_BASE
    type: string
    description: OpenAI API Base URL
    defaultValue: null`

	runtimeParameters := `[
		{
			key="OPENAI_API_BASE",
			type="string",
			value="https://datarobot-genai-enablement.openai.azure.com/"
		},
	  ]`
	schedule := &map[string]string{ // Add schedule configuration
		"minute":       "10 15",
		"hour":         "*",
		"day_of_week":  "*",
		"month":        "*",
		"day_of_month": "*",
	}
	err = os.WriteFile(folderPath+"/"+metadataFileName, []byte(metadataFileContents), 0644)
	if err != nil {
		t.Fatal(err)
	}

	noneEgressNetworkPolicy := "none"
	publicEgressNetworkPolicy := "public"

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
				Config: customJobResourceConfig(
					name,
					description,
					defaultJobType,
					&folderPath,
					nil,
					nil,
					noneEgressNetworkPolicy,
					nil,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomJobResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttr(resourceName, "job_type", defaultJobType),
					resource.TestCheckResourceAttr(resourceName, "folder_path", folderPath),
					resource.TestCheckResourceAttr(resourceName, "egress_network_policy", noneEgressNetworkPolicy),
					resource.TestCheckResourceAttrSet(resourceName, "environment_id"),
					resource.TestCheckResourceAttrSet(resourceName, "environment_version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update folder contents triggers update
			{
				PreConfig: func() {
					err := os.WriteFile(folderPath+"/new_file.txt", []byte("new file"), 0644)
					if err != nil {
						t.Fatal(err)
					}
				},
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: customJobResourceConfig(
					name,
					description,
					defaultJobType,
					&folderPath,
					nil,
					nil,
					noneEgressNetworkPolicy,
					nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomJobResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttr(resourceName, "job_type", defaultJobType),
					resource.TestCheckResourceAttr(resourceName, "folder_path", folderPath),
					resource.TestCheckResourceAttr(resourceName, "egress_network_policy", noneEgressNetworkPolicy),
					resource.TestCheckResourceAttrSet(resourceName, "environment_id"),
					resource.TestCheckResourceAttrSet(resourceName, "environment_version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update name, description, and files
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: customJobResourceConfig(
					newName,
					newDescription,
					defaultJobType,
					nil,
					[]FileTuple{{LocalPath: folderPath + "/" + metadataFileName}},
					nil,
					publicEgressNetworkPolicy,
					nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomJobResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "description", newDescription),
					resource.TestCheckResourceAttr(resourceName, "job_type", defaultJobType),
					resource.TestCheckNoResourceAttr(resourceName, "folder_path"),
					resource.TestCheckResourceAttr(resourceName, "files.0.0", folderPath+"/"+metadataFileName),
					resource.TestCheckResourceAttr(resourceName, "egress_network_policy", publicEgressNetworkPolicy),
					resource.TestCheckResourceAttrSet(resourceName, "environment_id"),
					resource.TestCheckResourceAttrSet(resourceName, "environment_version_id"),
				),
			},
			// Add runtime parameters
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
				Config: customJobResourceConfig(
					newName,
					newDescription,
					defaultJobType,
					nil,
					[]FileTuple{{LocalPath: folderPath + "/" + metadataFileName}},
					&runtimeParameters,
					publicEgressNetworkPolicy, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomJobResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "description", newDescription),
					resource.TestCheckResourceAttr(resourceName, "job_type", defaultJobType),
					resource.TestCheckResourceAttr(resourceName, "files.0.0", folderPath+"/"+metadataFileName),
					resource.TestCheckResourceAttr(resourceName, "runtime_parameter_values.0.key", "OPENAI_API_BASE"),
					resource.TestCheckResourceAttr(resourceName, "runtime_parameter_values.0.type", "string"),
					resource.TestCheckResourceAttr(resourceName, "egress_network_policy", publicEgressNetworkPolicy),
					resource.TestCheckResourceAttrSet(resourceName, "environment_id"),
					resource.TestCheckResourceAttrSet(resourceName, "environment_version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update job type triggers replacement
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: customJobResourceConfig(
					newName,
					newDescription,
					notificationJobType,
					nil,
					[]FileTuple{{LocalPath: folderPath + "/" + metadataFileName, PathInModel: metadataFileName}},
					&runtimeParameters,
					publicEgressNetworkPolicy, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomJobResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "description", newDescription),
					resource.TestCheckResourceAttr(resourceName, "job_type", notificationJobType),
					resource.TestCheckResourceAttr(resourceName, "files.0.0", folderPath+"/"+metadataFileName),
					resource.TestCheckResourceAttr(resourceName, "runtime_parameter_values.0.key", "OPENAI_API_BASE"),
					resource.TestCheckResourceAttr(resourceName, "runtime_parameter_values.0.type", "string"),
					resource.TestCheckResourceAttr(resourceName, "egress_network_policy", publicEgressNetworkPolicy),
					resource.TestCheckResourceAttrSet(resourceName, "environment_id"),
					resource.TestCheckResourceAttrSet(resourceName, "environment_version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},

			// Set schedule
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: customJobResourceConfig(
					name,
					description,
					defaultJobType,
					nil,
					[]FileTuple{{LocalPath: folderPath + "/" + metadataFileName, PathInModel: metadataFileName}},
					nil,
					noneEgressNetworkPolicy,
					schedule,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomJobResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "schedule.minute.0", "10"),
					resource.TestCheckResourceAttr(resourceName, "schedule.minute.1", "15"),
					resource.TestCheckResourceAttr(resourceName, "schedule.hour.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "schedule.day_of_week.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "schedule.month.0", "*"),
					resource.TestCheckResourceAttr(resourceName, "schedule.day_of_month.0", "*"),
					resource.TestCheckResourceAttrSet(resourceName, "schedule_id"),
				),
			},
			// Update job type to retraining
			{
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Config: customJobResourceConfig(
					newName,
					newDescription,
					retrainingJobType,
					nil,
					[]FileTuple{{LocalPath: folderPath + "/" + metadataFileName, PathInModel: metadataFileName}},
					&runtimeParameters,
					publicEgressNetworkPolicy, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomJobResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "description", newDescription),
					resource.TestCheckResourceAttr(resourceName, "job_type", retrainingJobType),
					resource.TestCheckResourceAttr(resourceName, "files.0.0", folderPath+"/"+metadataFileName),
					resource.TestCheckResourceAttr(resourceName, "runtime_parameter_values.0.key", "OPENAI_API_BASE"),
					resource.TestCheckResourceAttr(resourceName, "runtime_parameter_values.0.type", "string"),
					resource.TestCheckResourceAttr(resourceName, "egress_network_policy", publicEgressNetworkPolicy),
					resource.TestCheckResourceAttrSet(resourceName, "environment_id"),
					resource.TestCheckResourceAttrSet(resourceName, "environment_version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func TestCustomJobResourceConfigValidators_RuntimeParametersConflicting(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	r := NewCustomJobResource().(*CustomJobResource)

	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}
	r.Schema(ctx, schemaRequest, schemaResponse)
	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema diagnostics: %+v", schemaResponse.Diagnostics)
	}
	s := schemaResponse.Schema

	// Build a config with both runtime fields set (and environment_id set to satisfy AtLeastOneOf).
	configAttrs := make(map[string]tftypes.Value, len(s.GetAttributes()))
	for attrName := range s.GetAttributes() {
		attrType, diags := s.TypeAtPath(ctx, fwpath.Root(attrName))
		if diags.HasError() {
			t.Fatalf("failed to get type for attribute %q: %+v", attrName, diags)
		}
		configAttrs[attrName] = tftypes.NewValue(attrType.TerraformType(ctx), nil)
	}

	configAttrs["name"] = tftypes.NewValue(tftypes.String, "cj-test")
	configAttrs["environment_id"] = tftypes.NewValue(tftypes.String, "env-1")

	// runtime_parameter_values element
	rpvType, diags := s.TypeAtPath(ctx, fwpath.Root("runtime_parameter_values"))
	if diags.HasError() {
		t.Fatalf("failed to get type for runtime_parameter_values: %+v", diags)
	}
	rpvListType, ok := rpvType.(basetypes.ListType)
	if !ok || rpvListType.ElemType == nil {
		t.Fatalf("unexpected runtime_parameter_values type: %T", rpvType)
	}
	rpvElemTfType := rpvListType.ElemType.TerraformType(ctx)
	rpvElem := tftypes.NewValue(rpvElemTfType, map[string]tftypes.Value{
		"key":   tftypes.NewValue(tftypes.String, "FOO"),
		"type":  tftypes.NewValue(tftypes.String, "string"),
		"value": tftypes.NewValue(tftypes.String, "bar"),
	})
	configAttrs["runtime_parameter_values"] = tftypes.NewValue(rpvType.TerraformType(ctx), []tftypes.Value{rpvElem})

	// runtime_parameters element
	rpType, diags := s.TypeAtPath(ctx, fwpath.Root("runtime_parameters"))
	if diags.HasError() {
		t.Fatalf("failed to get type for runtime_parameters: %+v", diags)
	}
	rpListType, ok := rpType.(basetypes.ListType)
	if !ok || rpListType.ElemType == nil {
		t.Fatalf("unexpected runtime_parameters type: %T", rpType)
	}
	rpElemTfType := rpListType.ElemType.TerraformType(ctx)
	rpElem := tftypes.NewValue(rpElemTfType, map[string]tftypes.Value{
		"key":   tftypes.NewValue(tftypes.String, "BAZ"),
		"type":  tftypes.NewValue(tftypes.String, "string"),
		"value": tftypes.NewValue(tftypes.String, "qux"),
	})
	configAttrs["runtime_parameters"] = tftypes.NewValue(rpType.TerraformType(ctx), []tftypes.Value{rpElem})

	configRaw := tftypes.NewValue(s.Type().TerraformType(ctx), configAttrs)

	req := fwresource.ValidateConfigRequest{
		Config: tfsdk.Config{Schema: s, Raw: configRaw},
	}
	resp := &fwresource.ValidateConfigResponse{}

	for _, v := range r.ConfigValidators(ctx) {
		v.ValidateResource(ctx, req, resp)
	}

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected diagnostics error, got none")
	}

	found := false
	for _, d := range resp.Diagnostics {
		if strings.Contains(strings.ToLower(d.Detail()), "cannot be configured together") ||
			strings.Contains(strings.ToLower(d.Summary()), "cannot be configured together") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected conflicting-attributes diagnostic, got: %+v", resp.Diagnostics)
	}
}

type fakeServiceUpdateCustomJob struct {
	client.Service
	calls []fakeUpdateCustomJobCall
	fn    func(ctx context.Context, id string, req *client.UpdateCustomJobRequest) (*client.CustomJob, error)
}

type fakeUpdateCustomJobCall struct {
	id  string
	req *client.UpdateCustomJobRequest
}

func (f *fakeServiceUpdateCustomJob) UpdateCustomJob(ctx context.Context, id string, req *client.UpdateCustomJobRequest) (*client.CustomJob, error) {
	f.calls = append(f.calls, fakeUpdateCustomJobCall{id: id, req: req})
	if f.fn == nil {
		return &client.CustomJob{}, nil
	}
	return f.fn(ctx, id, req)
}

func TestUpdateCustomJobWithRuntimeParamsFallback_RuntimeParameters_SendsRuntimeParameters(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	runtimeParamObjectType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"key":   types.StringType,
			"type":  types.StringType,
			"value": types.StringType,
		},
	}

	runtimeParams, diags := listValueFromRuntimParameters(ctx, []RuntimeParameterValue{
		{
			Key:   types.StringValue("FOO"),
			Type:  types.StringValue("string"),
			Value: types.StringValue("bar"),
		},
	})
	if diags.HasError() {
		t.Fatalf("failed to build runtime parameters list: %+v", diags)
	}

	svc := &fakeServiceUpdateCustomJob{}
	r := &CustomJobResource{provider: &Provider{service: svc}}
	plan := CustomJobResourceModel{
		RuntimeParameters:      runtimeParams,
		RuntimeParameterValues: types.ListNull(runtimeParamObjectType),
	}
	updateReq := &client.UpdateCustomJobRequest{Name: "cj-test"}

	var d diag.Diagnostics
	err := r.updateCustomJobWithRuntimeParamsFallback(ctx, "cj-1", updateReq, plan, &d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.HasError() {
		t.Fatalf("unexpected diagnostics errors: %+v", d)
	}

	if got := len(svc.calls); got != 1 {
		t.Fatalf("expected 1 UpdateCustomJob call, got %d", got)
	}
	call := svc.calls[0]
	if call.id != "cj-1" {
		t.Fatalf("expected custom job id cj-1, got %q", call.id)
	}
	if call.req.RuntimeParameters == "" {
		t.Fatalf("expected RuntimeParameters to be set, got empty string")
	}
	if call.req.RuntimeParameterValues != "" {
		t.Fatalf("expected RuntimeParameterValues to be empty, got %q", call.req.RuntimeParameterValues)
	}

	var decoded []client.RuntimeParameterRequest
	if err := json.Unmarshal([]byte(call.req.RuntimeParameters), &decoded); err != nil {
		t.Fatalf("failed to unmarshal RuntimeParameters JSON %q: %v", call.req.RuntimeParameters, err)
	}
	if len(decoded) != 1 || decoded[0].FieldName != "FOO" || decoded[0].Type != "string" {
		t.Fatalf("unexpected decoded RuntimeParameters: %+v", decoded)
	}
}

func TestUpdateCustomJobWithRuntimeParamsFallback_RuntimeParameterValues_SendsRuntimeParameterValues(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	runtimeParamObjectType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"key":   types.StringType,
			"type":  types.StringType,
			"value": types.StringType,
		},
	}

	runtimeParamValues, diags := listValueFromRuntimParameters(ctx, []RuntimeParameterValue{
		{
			Key:   types.StringValue("FOO"),
			Type:  types.StringValue("string"),
			Value: types.StringValue("bar"),
		},
	})
	if diags.HasError() {
		t.Fatalf("failed to build runtime parameter values list: %+v", diags)
	}

	svc := &fakeServiceUpdateCustomJob{}
	r := &CustomJobResource{provider: &Provider{service: svc}}
	plan := CustomJobResourceModel{
		RuntimeParameters:      types.ListNull(runtimeParamObjectType),
		RuntimeParameterValues: runtimeParamValues,
	}
	updateReq := &client.UpdateCustomJobRequest{Name: "cj-test"}

	var d diag.Diagnostics
	err := r.updateCustomJobWithRuntimeParamsFallback(ctx, "cj-1", updateReq, plan, &d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.HasError() {
		t.Fatalf("unexpected diagnostics errors: %+v", d)
	}

	if got := len(svc.calls); got != 1 {
		t.Fatalf("expected 1 UpdateCustomJob call, got %d", got)
	}
	call := svc.calls[0]
	if call.req.RuntimeParameterValues == "" {
		t.Fatalf("expected RuntimeParameterValues to be set, got empty string")
	}
	if call.req.RuntimeParameters != "" {
		t.Fatalf("expected RuntimeParameters to be empty, got %q", call.req.RuntimeParameters)
	}

	var decoded []client.RuntimeParameterValueRequest
	if err := json.Unmarshal([]byte(call.req.RuntimeParameterValues), &decoded); err != nil {
		t.Fatalf("failed to unmarshal RuntimeParameterValues JSON %q: %v", call.req.RuntimeParameterValues, err)
	}
	if len(decoded) != 1 || decoded[0].FieldName != "FOO" || decoded[0].Type != "string" {
		t.Fatalf("unexpected decoded RuntimeParameterValues: %+v", decoded)
	}
}

func TestUpdateCustomJobWithRuntimeParamsFallback_RuntimeParameters_FeatureNotEnabled_FallsBackToRuntimeParameterValues(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	runtimeParams, diags := listValueFromRuntimParameters(ctx, []RuntimeParameterValue{
		{
			Key:   types.StringValue("FOO"),
			Type:  types.StringValue("string"),
			Value: types.StringValue("bar"),
		},
	})
	if diags.HasError() {
		t.Fatalf("failed to build runtime params list: %+v", diags)
	}
	// Ensure legacy list is known and non-empty so fallback branch will execute.
	runtimeParamValues := runtimeParams

	tests := []struct {
		name       string
		firstError string
	}{
		{
			name:       "runtimeParameters not allowed key",
			firstError: "runtimeParameters is not allowed key",
		},
		{
			name:       "feature not enabled",
			firstError: "field requires the RUNTIME_PARAMETERS_IMPROVEMENTS feature to be enabled",
		},
		{
			name:       "both substrings",
			firstError: "field requires the RUNTIME_PARAMETERS_IMPROVEMENTS feature to be enabled; runtimeParameters is not allowed key",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &fakeServiceUpdateCustomJob{}
			svc.fn = func(ctx context.Context, id string, req *client.UpdateCustomJobRequest) (*client.CustomJob, error) {
				// First attempt uses RuntimeParameters and fails with a feature flag / schema error.
				if req.RuntimeParameters != "" && req.RuntimeParameterValues == "" {
					return nil, fmt.Errorf("%s", tc.firstError)
				}
				// Second attempt uses legacy field and succeeds.
				if req.RuntimeParameterValues != "" && req.RuntimeParameters == "" {
					return &client.CustomJob{}, nil
				}
				return nil, fmt.Errorf("unexpected request shape: runtime_parameters=%q runtime_parameter_values=%q", req.RuntimeParameters, req.RuntimeParameterValues)
			}

			r := &CustomJobResource{provider: &Provider{service: svc}}
			plan := CustomJobResourceModel{
				RuntimeParameters:      runtimeParams,
				RuntimeParameterValues: runtimeParamValues,
			}
			updateReq := &client.UpdateCustomJobRequest{Name: "cj-test"}

			var d diag.Diagnostics
			err := r.updateCustomJobWithRuntimeParamsFallback(ctx, "cj-1", updateReq, plan, &d)
			if err != nil {
				t.Fatalf("expected nil error (fallback), got: %v", err)
			}
			if d.HasError() {
				t.Fatalf("unexpected diagnostics errors: %+v", d)
			}

			warnings := 0
			for _, diagItem := range d {
				if diagItem.Severity() == diag.SeverityWarning {
					warnings++
				}
			}
			if warnings != 1 {
				t.Fatalf("expected 1 warning diagnostic, got %d (%+v)", warnings, d)
			}

			if got := len(svc.calls); got != 2 {
				t.Fatalf("expected 2 UpdateCustomJob calls (new then legacy), got %d", got)
			}
			if svc.calls[0].req.RuntimeParameters == "" || svc.calls[0].req.RuntimeParameterValues != "" {
				t.Fatalf("expected first call to use RuntimeParameters only, got: %+v", svc.calls[0].req)
			}
			if svc.calls[1].req.RuntimeParameterValues == "" || svc.calls[1].req.RuntimeParameters != "" {
				t.Fatalf("expected second call to use RuntimeParameterValues only, got: %+v", svc.calls[1].req)
			}
		})
	}
}

func customJobResourceConfig(
	name,
	description,
	jobType string,
	folderPath *string,
	files []FileTuple,
	runtimeParameters *string,
	egressNetworkPolicy string,
	schedule *map[string]string,
) string {
	folderPathStr := ""
	if folderPath != nil {
		folderPathStr = fmt.Sprintf(`
	folder_path = "%s"`, *folderPath)
	}

	filesStr := ""
	if len(files) > 0 {
		filesStr = "files = ["
		for _, file := range files {
			if file.PathInModel != "" {
				filesStr += fmt.Sprintf(`
				["%s", "%s"],`, file.LocalPath, file.PathInModel)
			} else {
				filesStr += fmt.Sprintf(`
				["%s"],`, file.LocalPath)
			}
		}

		filesStr += "]"
	}

	runtimeParametersStr := ""
	if runtimeParameters != nil {
		runtimeParametersStr = fmt.Sprintf(`
	runtime_parameter_values = %s`, *runtimeParameters)
	}

	scheduleStr := ""
	if schedule != nil {
		scheduleStr = `
		schedule = {`
		for key, value := range *schedule {
			values := strings.Fields(value)
			scheduleStr += fmt.Sprintf(`
			%s = [`, key)
			for _, v := range values {
				scheduleStr += fmt.Sprintf(`"%s", `, v)
			}
			scheduleStr = scheduleStr[:len(scheduleStr)-2] + `]`
		}
		scheduleStr += `
		}`
	}

	log.Printf("Schedule: %s", scheduleStr)

	return fmt.Sprintf(`
resource "datarobot_custom_job" "test" {
	name = "%s"
	description = "%s"
	job_type = "%s"
	environment_id = "66d07fae0513a1edf18595bb"
	egress_network_policy = "%s"
	%s
	%s
	%s
	%s
}
`, name, description, jobType, egressNetworkPolicy, folderPathStr, filesStr, runtimeParametersStr, scheduleStr)
}

func checkCustomJobResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["datarobot_custom_job.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "datarobot_custom_job.test")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("Provider not found")
		}
		p.service = NewService(cl)

		traceAPICall("GetCustomJob")
		job, err := p.service.GetCustomJob(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if job.Name == rs.Primary.Attributes["name"] &&
			job.Description == rs.Primary.Attributes["description"] &&
			job.JobType == rs.Primary.Attributes["job_type"] &&
			job.Resources.EgressNetworkPolicy == rs.Primary.Attributes["egress_network_policy"] {
			if rs.Primary.Attributes["runtime_parameter_values.0.value"] != "" {
				for _, runtimeParam := range job.RuntimeParameters {
					if runtimeParam.FieldName == rs.Primary.Attributes["runtime_parameter_values.0.key"] &&
						runtimeParam.CurrentValue == rs.Primary.Attributes["runtime_parameter_values.0.value"] {
						return nil
					}
				}
				return fmt.Errorf("Runtime parameter value does not match")
			}
			return nil
		}

		// Validate schedule
		schedules, err := p.service.ListCustomJobSchedules(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if len(schedules) > 0 {
			schedule := schedules[0]
			if schedule.Schedule.Minute != rs.Primary.Attributes["schedule.minute.0"] ||
				schedule.Schedule.Hour != rs.Primary.Attributes["schedule.hour.0"] ||
				schedule.Schedule.DayOfWeek != rs.Primary.Attributes["schedule.day_of_week.0"] {
				return fmt.Errorf("Schedule does not match")
			}
		}

		return fmt.Errorf("Custom Job not found")
	}
}
