package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccCustomMetricJobResource(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_custom_metric_job.test"

	compareValuesSame := statecheck.CompareValue(compare.ValuesSame())

	name := "custom_metric_job " + nameSalt
	newName := "new_custom_metric_job " + nameSalt

	description := "example_description"
	newDescription := "new_example_description"

	folderPath := "custom_metric_job"
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
				Config: customMetricJobResourceConfig(
					name,
					description,
					&folderPath,
					nil,
					nil,
					noneEgressNetworkPolicy,
					"gauge",
					"higherIsBetter",
					"y",
					true),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomMetricJobResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttr(resourceName, "folder_path", folderPath),
					resource.TestCheckResourceAttr(resourceName, "egress_network_policy", noneEgressNetworkPolicy),
					resource.TestCheckResourceAttr(resourceName, "directionality", "higherIsBetter"),
					resource.TestCheckResourceAttr(resourceName, "units", "y"),
					resource.TestCheckResourceAttr(resourceName, "type", "gauge"),
					resource.TestCheckResourceAttr(resourceName, "time_step", "hour"),
					resource.TestCheckResourceAttr(resourceName, "is_model_specific", "true"),
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
				Config: customMetricJobResourceConfig(
					name,
					description,
					&folderPath,
					nil,
					nil,
					noneEgressNetworkPolicy,
					"sum",
					"lowerIsBetter",
					"label",
					false),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomMetricJobResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttr(resourceName, "folder_path", folderPath),
					resource.TestCheckResourceAttr(resourceName, "egress_network_policy", noneEgressNetworkPolicy),
					resource.TestCheckResourceAttr(resourceName, "directionality", "lowerIsBetter"),
					resource.TestCheckResourceAttr(resourceName, "units", "label"),
					resource.TestCheckResourceAttr(resourceName, "type", "sum"),
					resource.TestCheckResourceAttr(resourceName, "time_step", "hour"),
					resource.TestCheckResourceAttr(resourceName, "is_model_specific", "false"),
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
				Config: customMetricJobResourceConfig(
					newName,
					newDescription,
					nil,
					[]FileTuple{{LocalPath: folderPath + "/" + metadataFileName}},
					nil,
					publicEgressNetworkPolicy,
					"sum",
					"lowerIsBetter",
					"label",
					false),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomMetricJobResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "description", newDescription),
					resource.TestCheckNoResourceAttr(resourceName, "folder_path"),
					resource.TestCheckResourceAttr(resourceName, "files.0.0", folderPath+"/"+metadataFileName),
					resource.TestCheckResourceAttr(resourceName, "egress_network_policy", publicEgressNetworkPolicy),
					resource.TestCheckResourceAttr(resourceName, "directionality", "lowerIsBetter"),
					resource.TestCheckResourceAttr(resourceName, "units", "label"),
					resource.TestCheckResourceAttr(resourceName, "type", "sum"),
					resource.TestCheckResourceAttr(resourceName, "time_step", "hour"),
					resource.TestCheckResourceAttr(resourceName, "is_model_specific", "false"),
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
				},
				Config: customMetricJobResourceConfig(
					newName,
					newDescription,
					nil,
					[]FileTuple{{LocalPath: folderPath + "/" + metadataFileName}},
					&runtimeParameters,
					publicEgressNetworkPolicy,
					"sum",
					"lowerIsBetter",
					"label",
					false),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomMetricJobResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "description", newDescription),
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

func customMetricJobResourceConfig(
	name,
	description string,
	folderPath *string,
	files []FileTuple,
	runtimeParameters *string,
	egressNetworkPolicy,
	hostedMetricType,
	directionality,
	units string,
	isModelSpecific bool,
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

	return fmt.Sprintf(`
resource "datarobot_custom_metric_job" "test" {
	name = "%s"
	description = "%s"
	environment_id = "66d07fae0513a1edf18595bb"
	egress_network_policy = "%s"
	type = "%s"
	directionality = "%s"
	units = "%s"
	is_model_specific = %t
	%s
	%s
	%s
}
`, name, description, egressNetworkPolicy, hostedMetricType, directionality, units, isModelSpecific, folderPathStr, filesStr, runtimeParametersStr)
}

func checkCustomMetricJobResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["datarobot_custom_metric_job.test"]
		if !ok {
			return fmt.Errorf("Not found: %s", "datarobot_custom_metric_job.test")
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

		traceAPICall("GetHostedCustomMetricTemplate")
		hostedCustomMetricTemplate, err := p.service.GetHostedCustomMetricTemplate(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if job.Name == rs.Primary.Attributes["name"] &&
			job.Description == rs.Primary.Attributes["description"] &&
			job.Resources.EgressNetworkPolicy == rs.Primary.Attributes["egress_network_policy"] &&
			hostedCustomMetricTemplate.Directionality == rs.Primary.Attributes["directionality"] &&
			hostedCustomMetricTemplate.Units == rs.Primary.Attributes["units"] &&
			hostedCustomMetricTemplate.Type == rs.Primary.Attributes["type"] &&
			hostedCustomMetricTemplate.TimeStep == rs.Primary.Attributes["time_step"] {
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

		return fmt.Errorf("Custom Job not found")
	}
}
