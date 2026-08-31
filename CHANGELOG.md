## [Unreleased]

### Fixed

- `datarobot_artifact` `source.dir_hash` is no longer recomputed from the local tree during Read/refresh. Refresh runs before plan, so overwriting the last-applied hash with the current disk contents made `terraform plan` report no changes after editing source files. The hash is still computed at plan time and stored after a successful apply.
- `datarobot_workload` replacement wait no longer fails when `GET /workloads/{id}/replacement` returns 404. The Workload API detaches a finished replacement from the workload, so that 404 means the replacement completed. Apply now treats it as success.

### Added

- `source.generate_ignore` on `datarobot_artifact` (default `true`): when `source.dir` has neither `.drignore` nor `.wapiignore`, write a default `.drignore` at apply (never overwrite). Uploads and `source.dir_hash` honor `.drignore` (gitignore syntax) plus system excludes (`.datarobot.yaml` is never uploaded).
- Plan-time warnings on `datarobot_artifact` when `source.dir` uses the deprecated `.wapiignore` name, or holds both `.drignore` and `.wapiignore` (in which case `.drignore` wins and the other file's patterns are not applied).
- `source` block on `datarobot_artifact`: upload a local directory (`source.dir`) to the DataRobot catalog on create and update, auto-populate the primary container's `image_build_config.code_ref`, and track changes via computed `source.dir_hash`. After a successful upload on a draft artifact with `image_build_config`, the provider triggers an image build and, by default (`source.wait_for_build`, default `true`), polls until completion and populates the primary container's computed `image_uri`. Set `wait_for_build = false` to trigger a build without blocking apply. Requires a primary container with `image_build_config`. On draft artifacts, uploads are applied in-place; on locked artifacts, source changes clone to a new draft version, upload, build, patch `code_ref`, and lock the new version. Manual `code_ref` and `source` are mutually exclusive.
- Automatic artifact image build trigger after source upload on `datarobot_artifact`: when `source` is configured on a draft artifact with `image_build_config`, the provider triggers an image build and polls until completion by default (`source.wait_for_build`, default `true`), populating the primary container's computed `image_uri`. Set `wait_for_build = false` to trigger a build without blocking apply.
- Computed `build` block on `datarobot_artifact` container specs (`artifact_image_build_id`, `status`, `created_at`) exposing server-set image build metadata in resource state after a source-triggered build.
- `DATAROBOT_ARTIFACT_BUILD_POLL_INTERVAL` and `DATAROBOT_ARTIFACT_BUILD_POLL_TIMEOUT` environment variables to tune artifact image build polling (defaults: `10s` and `10m`; Go duration syntax). Set `DATAROBOT_SKIP_ARTIFACT_BUILD_ACC=1` to skip acceptance tests that require the Image Build Service.
- `datarobot_artifact` now streams artifact image build OTEL logs to the provider's stderr while waiting for a source-triggered image build to complete (`source.wait_for_build = true`, the default). Terraform routes provider stderr through its own logging pipeline, so this progress output is visible when `TF_LOG` is set (e.g. `TF_LOG=DEBUG`), not on a plain `terraform apply`. On failure, the apply error always includes a tailed excerpt of build logs (WAPI build logs with OTEL fallback) and a link to the full build log page in the DataRobot UI, regardless of `TF_LOG`. Default tail length is 30 lines; override with `DATAROBOT_ARTIFACT_BUILD_LOGS_TAIL_LINES`.
- `DATAROBOT_DEBUG` environment variable to opt into verbose HTTP request/response dumps and curl reproductions in error messages.
- `examples/resources/datarobot_workload` example: end-to-end code-to-workload flow (`datarobot_artifact` with `source` + build wait → `datarobot_workload`).
- `routes` attribute on `datarobot_artifact` container specs (primary containers only, at most 50): a list of `{ path, auth }` objects exposing additional paths from the workload's public endpoint with per-route authentication (`required`, `optional`, or `disabled`). Paths must start with `/`, be at most 1024 characters, and be unique within a container; all of this is checked at plan time. Route configuration is a cluster-level capability that is disabled by default — on a cluster without it, an artifact declaring `routes` fails with `Route configuration is disabled on this cluster`.

### Changed

- `datarobot_artifact`: `source.dir_hash` is computed by a different algorithm. 0.10.46 walked the whole directory, hashed each file's contents, concatenated those hashes in path order and hashed the result, so the file names never entered the digest. The hash now folds each file's path in beside its content hash and skips whatever `.drignore` and the system excludes cover. Both changes move the value, so every configuration already on 0.10.46 sees a one-time `source.dir_hash` diff on the first plan after upgrading, and that diff re-uploads the source. On a locked artifact the upload clones to a new draft version, rebuilds the image and locks again, so plan the upgrade for a window where that is acceptable. Subsequent plans are stable.
- `datarobot_artifact`: `*.tfvars` and `*.tfvars.json` under `source.dir` are never uploaded. They are system excludes alongside `terraform.tfstate` and `.terraform`, for the same reason: they hold the credentials the configuration was given, and they are input to Terraform rather than to the image being built. Unlike the entries in the generated `.drignore`, a system exclude cannot be re-enabled with a negation pattern, and it applies to projects that already have an ignore file and so never receive the generated template.
- Verbose HTTP request/response logging is no longer enabled implicitly by `TF_LOG=DEBUG`/`TRACE`; it now requires `DATAROBOT_DEBUG`. This keeps `TF_LOG=DEBUG` usable for provider progress output (such as artifact image build log tailing) without also dumping every API payload.
- Provider diagnostic output (HTTP dumps, `TRACE_API_CALLS` traces) now goes to stderr instead of stdout. Terraform reserves a plugin's stdout for the go-plugin handshake and reported anything written there as `[WARN] unexpected data`; these lines now appear as ordinary provider log entries.
- `datarobot_artifact`: `spec.container_groups.*.containers.*.image_uri` is now `Computed` in addition to `Optional`, allowing the provider to populate the image URI after a source-driven build.

## [0.10.46] - 2026-08-20

### Added

- `source` block on `datarobot_artifact`: upload a local directory (`source.dir`) to the DataRobot catalog on create and update, auto-populate the primary container's `image_build_config.code_ref`, and track changes via computed `source.dir_hash`. Requires a primary container with `image_build_config`. On draft artifacts, uploads are applied in-place; on locked artifacts, source changes clone to a new draft version, upload, patch `code_ref`, and lock the new version. Manual `code_ref` and `source` are mutually exclusive.
- Documentation and examples for in-place `datarobot_workload` replacement: operator guide in `docs/resources/workload.md` (WAPI rolling replacement vs legacy destroy/create, artifact dual-ID wiring, update-trigger table, apply duration), registry example at `examples/resources/datarobot_workload/`, and runnable workflow at `examples/workflows/workload_replacement/`.
- Plan-time handling for provider-managed `image_build_config.code_ref` when `source` is set: unknown values are decoded as null on create and restored from the primary container's state on update (including container reorder), so Terraform plan/apply stays consistent with computed catalog references.
- `datarobot_artifact` `type = "agent"` and optional `spec.a2a_enabled` for Workload API agent artifacts (A2A card management). `a2a_enabled` is valid only when `type` is `agent`.
- Computed `type` on `datarobot_workload`, mirroring the deployed artifact type (`service`, `nim`, or `agent`).
- `datarobot_execution_environment` now surfaces the execution environment version's build logs (via the OTel logs API, falling back to the legacy per-version build log file when OTel has nothing recorded yet, tailed to the last 30 lines by default and overridable via `DATAROBOT_EXECUTION_ENVIRONMENT_BUILD_LOG_TAIL_LINES`) and a link to the build logs in the DataRobot UI in the error message when a version fails to build.
### Fixed

- `datarobot_memory_space` updates no longer fail with `422 Unprocessable Entity` on `llmBaseUrl`. The provider sent an empty string for every attribute the config does not set, and the API parses `llmBaseUrl` as a URL, so any update to a memory space without `llm_base_url` was rejected. Unset attributes are now sent as null, which is also what the API requires to clear a stored value: it applies only the keys present in the request body, so an omitted key would leave the old value in place.
- `datarobot_memory_space` no longer requires `ENABLE_AGENTIC_MEMORY_API`. Either `ENABLE_AGENTIC_MEMORY_API` or `ENABLE_GENAI_EXPERIMENTATION` is now enough, which is what the memory service and the DataRobot API gateway already allow for the chat history surface (memory spaces and their sessions). The persistent memory (mem0) API stays gated on `ENABLE_AGENTIC_MEMORY_API`, so a memory space created with only `ENABLE_GENAI_EXPERIMENTATION` holds chat history but no persistent memories.
- Feature flag checks now evaluate the effective flag value via `POST /api/v2/entitlements/evaluate/` instead of reading the `permissions` map from `GET /api/v2/account/info/`, which only contains flags set directly on the user record. Flags inherited from the user's organization or groups (e.g. `ENABLE_AGENTIC_MEMORY_API` for `datarobot_memory_space`) no longer fail with a false "Feature not enabled" error. A flag missing from the evaluation response is now an explicit error instead of silently reading as disabled.
- Bumped `google.golang.org/grpc` from `1.79.3` to `1.82.1` to resolve `GO-2026-6061` (xDS RBAC authorization engine and HTTP/2 server transport) detected by `govulncheck`. The advisory is reachable from the provider's plugin gRPC server (`providerserver.Serve` → `transport.NewServerTransport`), not merely imported. Dependency-only change; no provider behavior change.
- Bumped the Go toolchain in `go.mod` from `1.26.5` to `1.26.6` to resolve eight standard library advisories detected by `govulncheck`, all fixed in `go1.26.6`. Four are reachable from provider code: `GO-2026-6218` (`net/url` quadratic `resolvePath`), `GO-2026-6090` (`crypto/tls` post-handshake message limit), `GO-2026-5972` (`encoding/asn1` recursion depth), and `GO-2026-5026` (`net/http` Punycode label rejection via `golang.org/x/net/idna`), traced through `providerserver.Serve` and the Files API client's `PollStatus`/`DownloadFile`. The remaining four are imported or required but not called: `GO-2026-6091` (`html/template`), `GO-2026-6089` (`net/http` `ReadHeaderTimeout`), `GO-2026-5942` (`net` DNS SVCB/HTTPS RR parsing), and `GO-2026-6088` (`encoding/xml` decode recursion). Toolchain-only change; no dependency or provider behavior change. All CI jobs resolve Go via `go-version-file: go.mod`, so this also raises the version used to build releases.

## [0.10.45] - 2026-07-27

### Added

- updated autoscaling object scheme on workload resources according to the new API version
- new `api-key` source for artifact container `environment_vars`, injecting a platform-managed DataRobot API token (`name` is optional and defaults to `DATAROBOT_API_TOKEN`)
- plan-time validation aligned with the Workload API OpenAPI schema: `cpuAverageUtilization` scaling requires `min_replica_count > 0`, and policy `target` must be non-negative
- `success_threshold` on `datarobot_artifact` container probes (`startup_probe`, `readiness_probe`, `liveness_probe`): minimum consecutive successes for the probe to be considered successful after a failure (matches `ProbeConfig.successThreshold`)

### Fixed

- `datarobot_deployment`: updating `registered_model_version_id` is no longer silently missed when the referenced value changes in the same apply (e.g. a new `datarobot_registered_model` version created upstream). The attribute's `UseStateForUnknown` plan modifier was substituting the prior state value whenever the config value was still unknown at plan time, suppressing the diff so `Update()` never ran and the deployment kept serving the previous model version. `Update()` already applied the model replacement correctly once triggered; only the missed diff detection needed fixing.
- `datarobot_deployment` updates no longer hangs when the new model version fails to start.
- `datarobot_workload` updates (artifact or runtime changes) no longer intermittently fail with `Error replacing Workload ... not found`. The provider now waits for the replacement by polling the workload record (`workload.replacement`) instead of the `/replacement` endpoint. The Workload API cleans up a `completed` replacement record almost immediately (~1s), so a `GET /replacement` between polls could 404 and was treated as a failure; the workload record makes completion (`replacement` cleared while `running`) and failure (`replacement.status == errored`, which persists) unambiguous and race-free.
- `datarobot_deployment`: the "deployment is not ready" timeout error now includes the deployment id and a console activity-log URL, so a failed activation/deactivation can be traced to the specific deployment and its logs.
- `datarobot_workload`: a container group that sets neither `replica_count` nor `autoscaling` no longer shows perpetual plan drift. The Workload API fills in a cluster-dependent scaling default (a scale-to-zero `autoscaling` block where `KEDA_DEFAULT_SCALE_TO_ZERO_ENABLED` is on, otherwise `replica_count`); that backend-owned `autoscaling` is now kept out of state so it matches the empty config, mirroring how `resource_bundles` and `bundle_selection_policy` are handled.

## [0.10.43] - 2026-07-15

### Added

- `internal/client/filesapi` package: context-aware Files API client (catalog create, staged upload, zip upload, async status polling, manifest listing) ported from the DR CLI as foundation for `datarobot_artifact` `source { dir }` support
- `internal/artifactsource` package: push-only directory upload orchestration (`PushDirectory`) over the Files API client (walk, hash, stage/zip routing, async poll) as foundation for `datarobot_artifact` `source { dir }` support
- `image_build_config` block on `datarobot_artifact` containers: configure code-to-workload image builds with optional `code_ref` and a `dockerfile` (`provided` or `generated`). Use with `status = "draft"` for pre-build artifacts; locking requires a completed build so workload-api has populated `image_uri`.
- `status` attribute on `datarobot_artifact`: `draft` (the current artifact version is mutable; spec changes are applied in-place and `artifact_id` stays the same) or `locked` (artifact versions are immutable; spec changes create a new version with a new `artifact_id` in the same `artifact_repository_id`). Defaults to `locked`. Locking a draft artifact is one-way. Changing `status` from `locked` to `draft` creates a new draft artifact (the Workload API cannot unlock in place).
- `datarobot_artifact` data source for looking up an existing Workload API artifact by ID
- `datarobot_artifacts` data source for listing Workload API artifacts with optional `status` and `limit` filters
- `datarobot_deployment` now surfaces deployment logs in the error message when a deployment fails to create
- `datarobot_workload`: in-place updates for `artifact_id` and `runtime` changes via the Workload API (`POST /workloads/{id}/replacement`, `PATCH /workloads/{id}/settings`) with polling until rollout completes
- `datarobot_workload`: optional `runtime.replacement_policy` block (`warmup_minutes`, `keep_old_version_minutes`) for rolling replacement when `artifact_id` or replacement policy changes

### Changed

- **Breaking:** `datarobot_workload` changes to `artifact_id` or `runtime` no longer destroy and recreate the workload. Updates run in place through the Workload API; the workload `id` and `endpoint` remain stable across artifact and runtime updates. This replaces the previous `RequiresReplace` (delete + create) behavior.

  **Migration from destroy/create:**

  | Before (pre-in-place replacement) | After (this release) |
  |-----------------------------------|----------------------|
  | `artifact_id` change → new workload ID | `artifact_id` change → same workload ID |
  | `endpoint` URL changed | `endpoint` URL stays the same |
  | `lifecycle { create_before_destroy = true }` required | Remove that `lifecycle` block: it existed only to preserve the endpoint during artifact updates |

  - Wire `artifact_id` to `datarobot_artifact.<name>.artifact_id` (artifact version ID), not `.id` (Terraform resource ID).
  - No Terraform state migration is required for typical configs; run `terraform plan` after upgrading to confirm the workload is updated in place rather than replaced.
  - If downstream automation assumed a **new** workload ID after each deploy, update it to rely on the stable ID instead.
  - Replacement is asynchronous and can block `terraform apply` for several minutes; if apply is interrupted or times out, wait for the workload replacement to finish on the platform, then run terraform apply again to sync state. See [`docs/resources/workload.md`](docs/resources/workload.md) and [`examples/workflows/workload_replacement`](examples/workflows/workload_replacement).
- `DATAROBOT_WORKLOAD_REPLACEMENT_POLL_INTERVAL` and `DATAROBOT_WORKLOAD_REPLACEMENT_POLL_TIMEOUT` environment variables to tune workload replacement polling (defaults: `5s` and `30m`; Go duration syntax)

### Fixed

- `datarobot_deployment` creation no longer hangs for the full timeout when the creation task reports a definitive `ERROR` status (e.g. custom model failed to start). The provider now fails fast with the task's error message instead of blind-polling deployment status, which never resolves in that case.
- `datarobot_deployment` activation and deactivation (e.g. around runtime parameter updates) now track the async status-change task and fail fast with the task's error message when it reports `ERROR`, instead of blind-polling deployment status until the timeout.
- Deployment model replacement updates now wait for backend `modelPackageId` propagation after the deployment returns to `active`, reducing false positives from short consistency delays. The update now fails only after a bounded wait if the deployment still serves the previous model package, and reports a clear mismatch error instead of writing incorrect planned state.
- Added integration-style mock tests for deployment model replacement success and mismatch scenarios to validate backend `modelPackageId` verification behavior.
- Bumped the Go toolchain target from `1.26.4` to `1.26.5` to pick up the standard-library `crypto/tls` fix for `GO-2026-5856` detected by `govulncheck`.
- when custom model is created with `source_llm_blueprint_id` and user defines `runtime_parameter_values` on the resource, default LLM blueprint runtime parameters are not removed overwritten anymore

## [0.10.42] - 2026-06-30

### Added

- `MemorySpace` resource now supports `llm_model_name`, `llm_base_url`, and `custom_instructions` fields

## [0.10.41] - 2026-06-29

### Fixed

- Example configurations for `datarobot_custom_application_from_environment`, `datarobot_datasource`, `datarobot_custom_metric`, `datarobot_batch_prediction_job_definition`, and `datarobot_deployment_retraining_policy` are now self-contained: they declare every resource they reference, so the examples can be applied as-is and downstream SDK/doc generation can convert them.

## [0.10.40] - 2026-06-26

### Added

- `custom_chunking` field on the `datarobot_vector_database` resource: when enabled, DataRobot treats each row of the source dataset as a finished chunk instead of running the built-in chunker (use for pre-chunked datasets).
- `datarobot_custom_model_from_vector_database` resource: packages a vector database into a deployable RAG custom model via the GenAI workshop endpoint, with optional compute settings (replicas, memory, resource bundle, network egress).
- `datarobot_quota` resource: manage per-resource usage quotas (e.g. on a deployment) via `/api/v2/quotas/`, with order-insensitive `default_rules` of `{rule, limit, window}`.

## [0.10.39] - 2026-05-28

### Fixed
- changed credential_id into dr_credential_id to correspond to the Workload API vocab

### Added
- added support of human-readable memory definitions in resource allocations in workload resource

## [0.10.38] - 2026-05-22

### Fixed

- added credential type for environment variables on the `datarobot_artifact` resource

## [0.10.37] - 2026-05-20

### Fixed

- dropped resource_request attribute from `datarobot_artifact` resource and added container groups to the `datarobot_workload`. Now resources are managed via resource_allocation for each container in the container_group in workload resources

## [0.10.36] - 2026-05-12

### Fixed

- `datarobot_memory_space` now checks the correct feature flag name `ENABLE_AGENTIC_MEMORY_API` (previously checked `AGENTIC_MEMORY_API`, which always evaluated to disabled even on accounts where Memory Spaces were enabled)

## [0.10.35] - 2026-05-11

### Added

- `datarobot_artifact` & `datarobot_workload` resources are added. It creates Artifact objects in Workload API.

### Fixed

- Fixed external sharing settings being stripped on `CustomApplicationSource`/`CustomApplication` updates even better

## [0.10.34] - 2026-04-20

### Fixed

- Fixed external sharing settings being stripped on `CustomApplicationSource`/`CustomApplication` updates

## [0.10.33] - 2026-04-14

### Added

- `runtime_parameters` added in `0.10.29` was removed in a favour of `runtime_parameter_values`. Now this parameter handles API changes internally in the provider code. This will make no need in updating existing terraform declarations.

## [0.10.32] - 2026-04-03

### Added

- Added `datarobot_memory_space` resource for managing Memory Spaces (requires enabled feature flag `AGENTIC_MEMORY_API`)

## [0.10.31] - 2026-03-24

### Added

- Added `health_endpoint_path` field to Application Source, Application Source From Template, Custom Application, and Custom Application From Environment resources. When set, this path is used for Kubernetes liveness and readiness probes instead of the path derived from `service_web_requests_on_root_path`.

### Fixed

- Fixed deployment creation task status polling getting stuck when the status API returns `INITIALIZED` indefinitely. After 2 minutes of `INITIALIZED` with no progress, the provider now skips the status check and verifies the deployment readiness directly
- Added detailed logging to task status polling — each poll now logs the task ID, current status, and elapsed time via `tflog.Info` for easier debugging
- Fixed `waitForDeploymentStatus` race condition where the fail-fast check on `inactive` status could race with `activateDeployment`, causing spurious permanent errors. Removed the fail-fast and added status/elapsed info to timeout errors instead
- Fixed deployment creation/model replacement to proceed to deployment readiness check even if task status polling fails, preventing unnecessary resource deletion on transient status API issues
- Deduplicated registered model version creation logic in Update — extracted `createNewRegisteredModelVersion` helper to eliminate near-identical code blocks for custom model version changes and tag-only changes
- Fixed QA Application acceptance test to set `GUARD_CONFIG_PLACEHOLDER` runtime parameter on the deployment, required by the updated `POST /customApplications/qanda/` API
- Fixed deployment runtime parameter updates failing with 500 after model replacement by reordering Update operations to apply runtime parameters before triggering model replacement
- Fixed deployment updates hanging indefinitely when the deployment is inactive. The provider now detects inactive deployments and automatically activates them before proceeding with updates or model replacements
- Fixed custom metric job runtime parameter values not being cleared when removed from configuration. The Update path was skipping empty lists instead of sending `"[]"` to the API

## [0.10.30] - 2026-03-10

### Added

- Add Terraform support of saving user MCP tool metadata: pkg/provider/user_mcp_tool_metadata_resource.go
- Add Terraform support of saving user MCP prompt metadata: pkg/provider/user_mcp_prompt_metadata_resource.go
- Add Terraform support of saving user MCP resource metadata: pkg/provider/user_mcp_resource_metadata_resource.go

## [0.10.29] - 2026-03-04

### Fixed

- Fixed 409 "Cannot delete custom model with existing deployments" error when DataRobot API returns from DELETE operations before async backend deletion completes. Added retry logic with exponential backoff (up to 5 minutes) to Custom Model and Registered Model deletion to wait for in-flight deployment deletions to complete. This handles race conditions caused by the API's async deletion behavior during resource replacement workflows.
- Fixed file upload issue. When uploading files (Application Sources, Custom Models, etc.), file paths with whitespace (especially Windows CRLF \r\n) were being sent directly to the API without normalization, causing the error: "Can not store file ..."
- Fixed Terraform plan drift issues in Custom Application, Custom Model, and Registered Model resources:
  - Nested attributes in `resources` block now properly marked as Computed to prevent unexpected plan changes
  - Removed incorrect `UseStateForUnknown()` plan modifiers from `version_id` fields that can change when new versions are created
  - Fixed `memory_mb` handling to properly transition between null and computed values when switching to/from `resource_bundle_id`
  - Fixed runtime parameter values filtering to skip nil values, preventing `<nil>` strings in state
- Fixed test isolation issues by adding timestamp-based unique identifiers to prevent name collisions in parallel test runs
- Fixed `required_key_scope_level` field to trigger resource replacement instead of in-place update when changed, since the DataRobot API does not support PATCH operations on this field (affects Application Source and Custom Application resources). Also fixed Update functions to skip sending this field in PATCH requests when the value has not changed, preventing spurious API errors during updates to other fields.
- Fixed runtime parameter removal for custom models - parameters removed from config are now properly reset to default values
- Fix the two-step create-then-rename race: POST /customApplicationSources/ always gets the default name "CustomApplicationSource", and the rename happens in a separate PATCH call. When two tests run in parallel, both POSTs land before either PATCH, causing the 409.
- added support of `runtime_parameters` attribute on custom model, application source & job resources. Now this field allows runtime parameters creation without providing runtime parameters definitions in the model-metadata.yaml file;
- `runtime_parameter_values` attribute on custom model, application source & job resources is marked as deprecated.

## [0.10.28] - 2026-01-15

### Fixed

- Fixing flaky tests in CI
- Fixed PROMPT_COLUMN_NAME runtime parameter propagation for AgenticWorkflow custom models in Registered Model Versions

## [0.10.27] - 2025-12-05

- Added support for creating tags for custom model

## [0.10.26] - 2025-12-04

- Fixed `required_key_scope_level` field not being populated in Custom Application - now supports scoped API key levels (viewer, user, admin)

## [0.10.24] - 2025-11-13
- fixes the problem of keeping Terraform state in sync during custom model creation

## [0.10.23] - 2025-11-10

- Fixed `resources` field not being populated in Application Source and Custom Application resources - now properly exposed as computed fields for Terraform state and Pulumi outputs
- Added ability to provide version ID for Execution Environment Read operation;
- Added ability to lookup Execution Environment entity by environment ID & version ID, alongside the name attribute;
- Fixed the problem of creating custom model with execution environment version ID set but still ending up with the latest version ID instead of provided one.

## [0.10.21] - 2025-10-15

- Added support for creating tags for registered model version
- Added MCP custom model target type

## [0.10.20] - 2025-09-22

### Fixed

- Fixed Windows zip file portability issue for making execution environments

## [0.10.19] - 2025-09-18

### Fixed

- Fixed maximumMemory field in custom models that is occassionally a float


## [0.10.18] - 2025-09-18

### Fixed

- Fixed version label auto-increment for Application Source (no wrap after v10)

## [0.10.17] - 2025-09-01

- Added a new key `llm_settings.custom_model_id` to `datarobot_llm_blueprint` resource

## [0.10.16] - 2025-08-27

- Added a new `playground_type` key to `datarobot_playground` resource
- Added documentation for the DATAROBOT_TIMEOUT_MINUTES environment variable for deployments and other long-running operations

## [0.10.15] - 2025-08-26

### Changed

- Dropped validators for model names on LLM Blueprints

## [0.10.14] - 2025-08-13

### Added

- Added a new `additional_guard_config` field for GuardConfiguration structure.

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
