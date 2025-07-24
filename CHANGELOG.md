## Unreleased

## [0.10.13] - 2025-07-24

### Fixed

- Relaxes basic auth password length to 1 character

## [0.10.12] - 2025-07-18

### Added

- Added a parameter `docker_image_uri` for ExecutionEnvironmentResource to allow environment version creation from image URI

## [0.10.11] - 2025-07-15

### Fixed

- Moved OAuth provider resources to `api/v2`

## [0.10.10] - 2025-07-01

### Fixed

- Fixed OAuth resource such that if a Client ID changes, the resource should be replaced

## [0.10.9] - 2025-06-23

### Fixed

- Fixed test name conflicts in batch file tests and LLM blueprint tests that could cause CI failures due to duplicate resource names

## [0.10.8] - 2025-06-19

### Fixed

- Ensure PROMPT_COLUMN_NAME is correctly propagated to newly created Registered Model Versions

## [0.10.7] - 2025-06-16

### Added

- Added support for specifying the `resources` field when creating a Custom Application from another Custom Application.

## [0.10.6] - 2025-06-12

### Added

- Added ability to create Custom Applications consisting of 100+ files.

## [0.10.5] - 2025-06-10

### Added

- Added new resource for managing OAuth providers in DataRobot (11.1+). This resource allows you to create, read, update, and delete OAuth provider configurations.

## [0.10.4] - 2025-06-03

### Added

- Added AgenticWorkflow custom model target type

## [0.10.3] - 2025-06-02

### Removed

- Added back symlink support from revert

## [0.10.2] - 2025-06-02

### Removed

- Reverted a fix to allow uploading 100+ files due to issues with Pulumi bridge provider

## [0.10.1] - 2025-05-29

### Fixed

- Windows build does not have inodes, dropped cycle detection for symlinks

## [0.10.0] - 2025-05-29

### Added

- Added support for following symlinks for folders in custom models


## [v0.9.5] - 2025-05-20

### Fixed

- Fixed batch file uploads and deletions to avoid API limits by processing them in groups of 100.
- Fixed the DynamicPseudoType error happening w/ the pulumi client.

## [v0.9.4] - 2025-05-14

### Added

- Schedule support for the `CustomJob` resource.
- DeploymentRetrainingPolicy now supports `use_case_id` attribute setting.
- Password length validation in `basic_credential` resource.

### Fixed

- Fixed batch file uploads and deletions to avoid API limits by processing them in groups of 100.
- Fixed test naming convention to avoid conflicts with other test files.

## [v0.9.3] - 2025-04-23

### Fixed

- Fixed memory size overwrite issue in the Custom Model resource. Added a check to ensure that the memory size attribute is not set when the `resource_bundle_id` is set to a non-empty value. This prevents the memory size from being unintentionally overwritten when using resource bundles.

## v0.9.2

- Trigger new Execution Environment version on Docker Image changes

## [v0.9.1] - 2025-04-17

### Added
- Add `retraining_settings` to the Deployment resource.
- Add functionality to dynamic creation/delete folders when testing some resources, to prevent silly errors.

### Changed
- Flow of how environment variables are set in the provider.
- README.md, and DEVELOPMENT.md updated with a contributing information and some tips.
- Updated some resources to load environment variables from the provider instead of directly from the environment during resource initialization (applies only to tests, not the provider itself).

### Fixed
- Fix `notebook_resource` tests

## 0.8.19

- Fix Application Source panic

## 0.8.18

- Fix Application Source resource session affinity being set by default

## 0.8.17

- Add AzureCredentialResource

## 0.8.16

- Add requiresReplace to CustomModelLLMValidation attributes

## 0.8.15

- Update CustomModelLLMValidation resource attributes based on underlying API changes

## 0.8.14

- Fix version_name error on Registered Model From Leaderboard updates

## 0.8.13

- Trigger new Execution Environment version on Docker Context content changes

## 0.8.12

- Vector Database versioning
- Fix adding Use Case to entity idempotency issue

## 0.8.11

- Add more parameter validators for Enum parameters

## 0.8.10

- Set retraining_user_id when Retraining Policy has a schedule trigger

## 0.8.9

- Fix custom_model_llm_settings for LLM Blueprint resource

## 0.8.8

- Add allow_auto_stopping attribute to Custom Applications

## 0.8.7

- Fix bug in Custom Model guard updates, by making sure to update the Custom Model guards even if there are no new guard configs

## 0.8.6

- Fix Schedule conversion for Deployment feature cache settings

## 0.8.5

- Custom Model LLM Validation Resource

## 0.8.4

- Application Source from Template Resource

## 0.8.3

- Custom Metric resource

## 0.8.2

- Deployment feature cache settings

## 0.8.1

- Add batch_monitoring Deployment setting

## 0.8.0

- Notification Channel resource
- Notification Policy resource

## 0.7.6

- enable manual feature selection for Deployment drift tracking

## 0.7.5

- wait for Execution Environments to finish building

## 0.7.4

- add Resource Bundle for Deployments

## 0.7.3

- runtime_parameter_values for Deployments

## 0.7.0

- Custom Application from Environment resource

## 0.6.3

- Fix Application Source resource bug

## 0.6.2

- docker_image param for Execution Environment

## 0.6.0

- Application Source resources

## 0.5.4

- SAP Datasphere batch prediction type

## 0.5.3

- AWS Credential Resource

## 0.5.2

- Deployment Retraining Policy resource

## 0.5.1

- Custom Metric Job Resource
- Custom Metric From Job Resource

## 0.5.0

- Batch Prediction Job Definition Resource

## 0.4.8

- Custom Job Resource

## 0.4.7

- Dataset from Datasource Resource

## 0.4.6

- Add Datasource Resource

## 0.4.5

- Add Datastore Resource

## 0.4.4

- Add NeMo Info to Custom Model GuardrailsConfiguration

## 0.4.3

- Registered Model prompt

## 0.4.2

- Make tests more flexible

## 0.4.1

- Support DR endpoint with trailing slash

## 0.4.0

- Execution Environment Resource and Data Source

## 0.3.6

- add DATAROBOT_TRACE_CONTEXT env var for X-DataRobot-Api-Consumer-Trace header
- remove unnecessary Custom App replacements

## 0.3.5

- populate User-Agent Header for analytics tracing

## 0.3.4

- Paginate List API calls

## 0.3.3

- Block updating immutable Custom Model attributes if deployed

## 0.3.1

- Clean up integration tests

## 0.3.0

- Remove real-time from Deployment predictions settings
- Add Resource Bundles for Custom Models
- Refactor Custom Model resource settings to match Python SDK and API specs

## 0.2.7

- Make `description` Computed for Custom Model in order to fix phantom updates

## 0.2.6

- Link entities to Use Cases
- Increase test coverage

## 0.2.5

- Support other types of comparand in Custom Model Guard conditions
- Registered Model from Leaderboard resource

## 0.2.4

- Make sure `moderation-config.yaml` is preserved on Custom Model updates

## 0.2.3

- Throw error when Deployment fails instead of waiting indefinitely
- Fix runtime params not being set on Custom Model update files
- Don't hardcode base environment for Application Source

## 0.2.2

- Use API return value instead of state for Custom Model deployments count

## 0.2.1

- Fix Runtime Parameter updates for Application Source
- Require replace for Credential udpates
- Always create new version for Custom Model + App Source updates
- Fix type errors for Guard intervention comparand field

## 0.2.0

- CustomModel/App Source/Dataset/Credential -- trigger updates when file contents change

## 0.1.39

- GCP key string parameter for credentials
- Remove base_environment_name from Custom Model, use base_environment_id and base_environment_version_id instead
- Make targetType computed for Custom Model

## 0.1.38

FEATURES:

- Replace Application Source for Custom Applications
- Fix updates for deployed Custom Models
- Dataset from URL

## 0.1.37

FEATURES:

- Deployment settings
- Rename DATAROBOT_API_KEY to DATAROBOT_API_TOKEN

## 0.1.28

FEATURES:

- Support the remaining Prediction Environment settings
- Match Dataset parameter names with Python SDK
- Change Custom Model relative folder path to root

## 0.1.27

FEATURES:

- Support Dependency Builds for Custom Models

## 0.0.24

FEATURES:

- Support Faithfulness Guard with OpenAI credentials
- Add name to Registered Model version
- Support directories and relative file paths for Application Sources
- Support LLM Settings and Vector Database Settings for LLM Blueprints

## 0.0.23

FEATURES:

- Support directories and relative file paths for Custom Models

## 0.0.22

FEATURES:

- Fix Custom Model runtime parameter ordering

## 0.0.21

FEATURES:

- Add Custom Model training data and class labels

## 0.0.20

FEATURES:

- Fix phantom updates for Custom Model and API Token Credential resources
- Add `language` and `prediction_threshold` parameters to Custom Model resource
- Fix CodeQL security warning

## 0.0.19

FEATURES:

- Rename `chat_application` to `qa_application`
- Support other types of Runtime Parameters besides string (type conversion is handled internally).
- Abort Deployment create if an Error occurs
