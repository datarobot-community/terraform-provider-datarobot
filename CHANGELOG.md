## 1.0.0

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
