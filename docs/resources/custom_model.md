---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "datarobot_custom_model Resource - datarobot"
subcategory: ""
description: |-
  Custom Model
---

# datarobot_custom_model (Resource)

Custom Model

## Example Usage

```terraform
resource "datarobot_remote_repository" "example" {
  name        = "Datarobot User Models"
  description = "GitHub repository with Datarobot user models"
  location    = "https://github.com/datarobot/datarobot-user-models"
  source_type = "github"

  # set the credential id for private repositories
  # credential_id = datarobot_api_token_credential.example.id
}

resource "datarobot_custom_model" "example" {
  name        = "Example from GitHub"
  description = "An example custom model from GitHub repository"
  files = [
    "file1.py",
    "file2.py",
  ]
  target_type         = "Binary"
  target_name         = "my_label"
  base_environment_id = "65f9b27eab986d30d4c64268"

  # Optional
  # source_remote_repositories = [
  #   {
  #     id  = datarobot_remote_repository.example.id
  #     ref = "master"
  #     source_paths = [
  #       "model_templates/python3_dummy_binary",
  #     ]
  #   }
  # ]
  # guard_configurations = [
  #   {
  #     template_name = "Rouge 1"
  #     name          = "Rouge 1 response"
  #     stages        = ["response"]
  #     intervention = {
  #       action  = "block"
  #       message = "response has been blocked by Rogue 1 guard"
  #       condition = jsonencode({
  #         "comparand": 0.5, 
  #         "comparator": "greaterThan"
  #       })
  #     }
  #   },
  # ]
  # overall_moderation_configuration = {
  #   timeout_sec    = 120
  #   timeout_action = "score"
  # }
  # memory_mb      = 512
  # replicas       = 2
  # network_access = "NONE"
}

output "example_id" {
  value       = datarobot_custom_model.example.id
  description = "The id for the example custom model"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The name of the Custom Model.

### Optional

- `base_environment_id` (String) The ID of the base environment for the Custom Model.
- `base_environment_version_id` (String) The ID of the base environment version for the Custom Model.
- `class_labels` (List of String) Class labels for multiclass classification. Cannot be used with class_labels_file.
- `class_labels_file` (String) Path to file containing newline separated class labels for multiclass classification. Cannot be used with class_labels.
- `description` (String) The description of the Custom Model.
- `files` (Attributes List) List of files to upload, each with a source (local path) and destination (path in model). (see [below for nested schema](#nestedatt--files))
- `folder_path` (String) The path to a folder containing files to build the Custom Model. Each file in the folder is uploaded under path relative to a folder path.
- `guard_configurations` (Attributes List) The guard configurations for the Custom Model. (see [below for nested schema](#nestedatt--guard_configurations))
- `is_proxy` (Boolean) Flag indicating if the Custom Model is a proxy model.
- `language` (String) The language used to build the Custom Model.
- `memory_mb` (Number) The memory in MB for the Custom Model.
- `negative_class_label` (String) The negative class label of the Custom Model.
- `network_access` (String) The network access for the Custom Model.
- `overall_moderation_configuration` (Attributes) The overall moderation configuration for the Custom Model. (see [below for nested schema](#nestedatt--overall_moderation_configuration))
- `positive_class_label` (String) The positive class label of the Custom Model.
- `prediction_threshold` (Number) The prediction threshold of the Custom Model.
- `replicas` (Number) The replicas for the Custom Model.
- `resource_bundle_id` (String) A single identifier that represents a bundle of resources: Memory, CPU, GPU, etc.
- `runtime_parameter_values` (Attributes List) The runtime parameter values for the Custom Model. (see [below for nested schema](#nestedatt--runtime_parameter_values))
- `source_llm_blueprint_id` (String) The ID of the source LLM Blueprint for the Custom Model.
- `source_remote_repositories` (Attributes List) The source remote repositories for the Custom Model. (see [below for nested schema](#nestedatt--source_remote_repositories))
- `target_name` (String) The target name of the Custom Model.
- `target_type` (String) The target type of the Custom Model.
- `training_data_partition_column` (String) The name of the partition column in the training dataset assigned to the Custom Model.
- `training_dataset_id` (String) The ID of the training dataset assigned to the Custom Model.
- `use_case_ids` (List of String) The list of Use Case IDs to add the Custom Model version to.

### Read-Only

- `deployments_count` (Number) The number of deployments for the Custom Model.
- `files_hashes` (List of String) The hash of file contents for each file in files.
- `folder_path_hash` (String) The hash of the folder path contents.
- `id` (String) The ID of the Custom Model.
- `training_dataset_name` (String) The name of the training dataset assigned to the Custom Model.
- `training_dataset_version_id` (String) The version ID of the training dataset assigned to the Custom Model.
- `version_id` (String) The ID of the latest Custom Model version.

<a id="nestedatt--files"></a>
### Nested Schema for `files`

Required:

- `destination` (String) Path in the model.
- `source` (String) Local filesystem path.


<a id="nestedatt--guard_configurations"></a>
### Nested Schema for `guard_configurations`

Required:

- `intervention` (Attributes) The intervention for the guard configuration. (see [below for nested schema](#nestedatt--guard_configurations--intervention))
- `name` (String) The name of the guard configuration.
- `stages` (List of String) The list of stages for the guard configuration.
- `template_name` (String) The template name of the guard configuration.

Optional:

- `deployment_id` (String) The deployment ID of this guard.
- `input_column_name` (String) The input column name of this guard.
- `llm_type` (String) The LLM type for this guard.
- `nemo_info` (Attributes) Configuration info for NeMo guards. (see [below for nested schema](#nestedatt--guard_configurations--nemo_info))
- `openai_api_base` (String) The OpenAI API base URL for this guard.
- `openai_credential` (String) The ID of an OpenAI credential for this guard.
- `openai_deployment_id` (String) The ID of an OpenAI deployment for this guard.
- `output_column_name` (String) The output column name of this guard.

<a id="nestedatt--guard_configurations--intervention"></a>
### Nested Schema for `guard_configurations.intervention`

Required:

- `action` (String) The action of the guard intervention.
- `condition` (String) The JSON-encoded condition of the guard intervention. e.g. `{"comparand": 0.5, "comparator": "lessThan"}`

Optional:

- `message` (String) The message of the guard intervention.


<a id="nestedatt--guard_configurations--nemo_info"></a>
### Nested Schema for `guard_configurations.nemo_info`

Optional:

- `actions` (String) The actions for the NeMo information.
- `blocked_terms` (String) NeMo guardrails blocked terms list.
- `llm_prompts` (String) NeMo guardrails prompts.
- `main_config` (String) Overall NeMo configuration YAML.
- `rails_config` (String) NeMo guardrails configuration Colang.



<a id="nestedatt--overall_moderation_configuration"></a>
### Nested Schema for `overall_moderation_configuration`

Optional:

- `timeout_action` (String) The timeout action of the overall moderation configuration.
- `timeout_sec` (Number) The timeout in seconds of the overall moderation configuration.


<a id="nestedatt--runtime_parameter_values"></a>
### Nested Schema for `runtime_parameter_values`

Required:

- `key` (String) The name of the runtime parameter.
- `type` (String) The type of the runtime parameter.
- `value` (String) The value of the runtime parameter (type conversion is handled internally).


<a id="nestedatt--source_remote_repositories"></a>
### Nested Schema for `source_remote_repositories`

Required:

- `id` (String) The ID of the source remote repository.
- `ref` (String) The reference of the source remote repository.
- `source_paths` (List of String) The list of source paths in the source remote repository.
