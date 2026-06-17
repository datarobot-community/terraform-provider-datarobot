# Implementation: Pipelines API Terraform Resources

## Overview

4 new resources added to `terraform-provider-datarobot`:
1. `datarobot_pipeline`
2. `datarobot_pipeline_input`
3. `datarobot_pipeline_schedule`
4. `datarobot_pipeline_image`

Pattern mirrors the existing `artifact` + `workload` resources:
- Client types/service methods → `internal/client/pipelines_service.go`
- Resource implementations → `pkg/provider/pipeline_*_resource.go`
- Registration → `pkg/provider/provider.go`

---

## Key API Notes (discovered during acceptance testing)

**JSON field names:** The pipelines-api uses Pydantic's `alias_generator=to_camel` on all response schemas, so every multi-word field is serialized as camelCase (`taskNames`, `cronExpression`, `latestVersion`, etc.). Primary ID fields use `Field(alias="id")` so they all serialize as `"id"` regardless of their Python field name.

**Request bodies:** Pydantic has `populate_by_name=True`, so the server accepts both snake_case and camelCase in request bodies. Go request struct tags use snake_case (e.g. `json:"cron_expression"`) and the server accepts them fine.

**Two Pipeline response shapes:**
- `PipelineCreateResponse` (returned by POST create and PATCH lock): has a top-level `"version": int` field, no `versions` array.
- `PipelineDetailResponse` (returned by GET): has a `"versions": [...]` array, no top-level `"version"` field.
- The Go `Pipeline` struct handles both by having `VersionNumber *int json:"version"` AND `Versions []PipelineVersion json:"versions"`.

**`version_id` vs version number:** `PipelineInputResponse.version_id` is an internal MongoDB FK (auto-increment integer like 35), NOT the user-facing pipeline version number (1, 2, 3). The Go `loadPipelineInputIntoModel` does NOT read `VersionID` — the version number is user-provided and preserved from plan/state.

**Scheduling not yet wired:** `POST /pipelines/{id}/versions/{v}/schedules/` returns 500 on staging — the K8s CronJob creation is not yet implemented. `TestAccPipelineScheduleResource` is gated behind `ACCEPTANCE_RUN_PIPELINES_SCHEDULING=1`.

---

## `internal/client/pipelines_service.go`

### Types

```go
// Enums
type PipelineMode string          // "draft" | "locked"
type PipelineVersionStatus string // "PENDING" | "READY" | "FAILED"
type PipelineInputState string    // "VALID" | "INVALID"
type PipelineScheduleStatus string // "ACTIVE" | "PAUSED" | "DELETED"
type PipelineImageStatus string // "CREATING" | "READY" | "ERROR"

// Pipeline — handles both PipelineCreateResponse and PipelineDetailResponse
type PipelineVersion struct {
    Version       int                   `json:"version"`
    Status        PipelineVersionStatus `json:"status"`
    TaskNames     []string              `json:"taskNames,omitempty"`
    PythonVersion string                `json:"pythonVersion"`
    ErrorDetail   *string               `json:"errorDetail,omitempty"`
    CreatedAt     string                `json:"createdAt"`
}

type Pipeline struct {
    PipelineID    string            `json:"id"`           // alias="id" in Python
    Name          string            `json:"name"`
    Description   *string           `json:"description,omitempty"`
    Mode          PipelineMode      `json:"mode"`
    IsActive      bool              `json:"isActive"`
    TaskNames     []string          `json:"taskNames,omitempty"`
    PythonVersion *string           `json:"pythonVersion,omitempty"`
    VersionNumber *int              `json:"version,omitempty"` // from PipelineCreateResponse only
    Versions      []PipelineVersion `json:"versions"`          // from PipelineDetailResponse only
    CreatedAt     string            `json:"createdAt"`
    UpdatedAt     string            `json:"updatedAt"`
}

// PipelineInput
type PipelineInput struct {
    InputID    string             `json:"id"`          // alias="id" in Python
    PipelineID string             `json:"pipelineId"`
    VersionID  *int               `json:"versionId,omitempty"` // internal DB FK, NOT version number
    IsDraft    bool               `json:"isDraft"`
    Payload    map[string]any     `json:"payload"`
    State      PipelineInputState `json:"state"`
    CreatedAt  string             `json:"createdAt"`
    UpdatedAt  string             `json:"updatedAt"`
}
type PipelineInputCreateRequest struct {
    Payload map[string]any `json:"payload"`
}
type PipelineInputUpdateRequest struct {
    Payload map[string]any `json:"payload"`
}

// PipelineSchedule
type PipelineSchedule struct {
    ScheduleID     string                 `json:"id"`             // alias="id" in Python
    PipelineID     string                 `json:"pipelineId"`
    Version        int                    `json:"version"`
    CronExpression string                 `json:"cronExpression"`
    Timezone       string                 `json:"timezone"`
    Status         PipelineScheduleStatus `json:"status"`
    CreatedAt      string                 `json:"createdAt"`
    UpdatedAt      string                 `json:"updatedAt"`
}
// pipeline_input_id is NOT returned by GET — stored in TF state only
type PipelineScheduleCreateRequest struct {
    CronExpression  string `json:"cron_expression"`   // snake_case OK (populate_by_name=True)
    PipelineInputID string `json:"pipeline_input_id"`
    Timezone        string `json:"timezone,omitempty"`
}
type PipelineScheduleUpdateRequest struct {
    CronExpression *string `json:"cron_expression,omitempty"`
    Timezone       *string `json:"timezone,omitempty"`
}

// PipelineImage
type PipelineImageVersion struct {
    Version     int                       `json:"version"`
    Packages    []string                  `json:"packages"`
    Status      PipelineImageStatus `json:"status"`
    ErrorDetail *string                   `json:"errorDetail,omitempty"`
    CreatedAt   string                    `json:"createdAt"`
    UpdatedAt   string                    `json:"updatedAt"`
}
type PipelineImage struct {
    ImageID string                       `json:"id"`            // alias="id" in Python
    Name          string                       `json:"name"`
    Description   *string                      `json:"description,omitempty"`
    LatestVersion int                          `json:"latestVersion"`
    Versions      []PipelineImageVersion `json:"versions"`
    CreatedAt     string                       `json:"createdAt"`
    UpdatedAt     string                       `json:"updatedAt"`
}
type PipelineImageCreateRequest struct {
    Name        string   `json:"name"`
    Description *string  `json:"description,omitempty"`
    Packages    []string `json:"packages"`
}
type PipelineImageUpdateRequest struct {
    Packages []string `json:"packages"`
}
```

### Service interface methods

```go
// Pipeline (Pipelines API)
CreatePipeline(ctx context.Context, fileName string, content []byte, description *string) (*Pipeline, error)
GetPipeline(ctx context.Context, id string) (*Pipeline, error)
UpdatePipelineDraft(ctx context.Context, id string, fileName string, content []byte) (*Pipeline, error)
LockPipeline(ctx context.Context, id string) (*Pipeline, error)
DeletePipeline(ctx context.Context, id string) error

// PipelineInput (Pipelines API)
CreateDraftPipelineInput(ctx context.Context, pipelineID string, req *PipelineInputCreateRequest) (*PipelineInput, error)
CreateLockedPipelineInput(ctx context.Context, pipelineID string, version int, req *PipelineInputCreateRequest) (*PipelineInput, error)
GetDraftPipelineInput(ctx context.Context, pipelineID, inputID string) (*PipelineInput, error)
GetLockedPipelineInput(ctx context.Context, pipelineID string, version int, inputID string) (*PipelineInput, error)
UpdateDraftPipelineInput(ctx context.Context, pipelineID, inputID string, req *PipelineInputUpdateRequest) (*PipelineInput, error)
DeleteDraftPipelineInput(ctx context.Context, pipelineID, inputID string) error
DeleteLockedPipelineInput(ctx context.Context, pipelineID string, version int, inputID string) error

// PipelineSchedule (Pipelines API)
CreatePipelineSchedule(ctx context.Context, pipelineID string, version int, req *PipelineScheduleCreateRequest) (*PipelineSchedule, error)
GetPipelineSchedule(ctx context.Context, pipelineID string, version int, scheduleID string) (*PipelineSchedule, error)
UpdatePipelineSchedule(ctx context.Context, pipelineID string, version int, scheduleID string, req *PipelineScheduleUpdateRequest) (*PipelineSchedule, error)
DeletePipelineSchedule(ctx context.Context, pipelineID string, version int, scheduleID string) error

// PipelineImage (Pipelines API)
CreatePipelineImage(ctx context.Context, req *PipelineImageCreateRequest) (*PipelineImage, error)
GetPipelineImage(ctx context.Context, id string) (*PipelineImage, error)
UpdatePipelineImage(ctx context.Context, id string, req *PipelineImageUpdateRequest) (*PipelineImage, error)
DeletePipelineImage(ctx context.Context, id string) error
```

### API paths
- Pipelines: `POST/GET/PATCH/DELETE /api/v2/pipelines/{id}/` and `PATCH /api/v2/pipelines/{id}/mode/`
- Draft inputs: `POST/GET/PATCH/DELETE /api/v2/pipelines/{id}/inputs/{input_id}/`
- Locked inputs: `POST/GET/DELETE /api/v2/pipelines/{id}/versions/{ver}/inputs/{input_id}/`
- Schedules: `POST/GET/PATCH/DELETE /api/v2/pipelines/{id}/versions/{ver}/schedules/{schedule_id}/`
- Images: `POST/GET/PATCH/DELETE /api/v2/pipelines/images/{env_id}/`

**Multipart upload:** `CreatePipeline` and `UpdatePipelineDraft` use `uploadFileFromBinary[T]` with field name `"file"` and optional `"description"` string in `extraFields`.

---

## `datarobot_pipeline` — `pkg/provider/pipeline_resource.go`

### Schema attributes

| Attribute | TF Type | Behavior |
|---|---|---|
| `id` | string, computed | = PipelineID (`"id"` in API) |
| `source_file` | string, required | Path to `.py` file using `@dr.task` / `@dr.pipeline` decorators |
| `source_file_hash` | string, computed | SHA-256 of file contents; triggers update on content change |
| `description` | string, optional | Forces replace when locked |
| `mode` | string, optional, default `"draft"` | `"draft"` or `"locked"` |
| `current_version` | int, computed | Null while draft; set from `VersionNumber` (create/lock) or `Versions[0].Version` (read) |
| `task_names` | list(string), computed | From `Pipeline.TaskNames` or `Versions[0].TaskNames` |

### Lifecycle logic
- **Create:** Read file bytes; multipart `POST /pipelines/`; if `mode = "locked"`, call `PATCH /pipelines/{id}/mode/`. `current_version` comes from `Pipeline.VersionNumber` in the lock response.
- **Read:** `GET /pipelines/{id}/`; `current_version` from `Versions[0].Version`; `source_file`/`source_file_hash` preserved from state (API doesn't return the file path).
- **Update (draft, file changed):** Re-upload via `PATCH /pipelines/{id}/`
- **Update (draft → locked):** `PATCH /pipelines/{id}/mode/`; subsequent locked changes use `RequiresReplace`
- **Update (locked, any field):** `RequiresReplace` via `ModifyPlan`
- **Delete:** `DELETE /pipelines/{id}/`

### Pipeline source file format
Must use DataRobot SDK decorators (validated by the API):
```python
import datarobot as dr

@dr.task
def my_task(x):
    return x

@dr.pipeline
def my_pipeline(x):
    return my_task(x)
```
The `@pipeline` function name must be **unique per user** — the API returns 409 if a draft with the same function name already exists.

---

## `datarobot_pipeline_input` — `pkg/provider/pipeline_input_resource.go`

### Schema attributes

| Attribute | TF Type | Behavior |
|---|---|---|
| `id` | string, computed | = InputID (`"id"` in API) |
| `pipeline_id` | string, required | Forces replace on change |
| `version` | int, optional | Set → locked scope; unset → draft scope; forces replace on change |
| `payload` | string, required | JSON string; normalized for diff (whitespace/key-order insensitive) |
| `state` | string, computed | `VALID` / `INVALID` |

### Lifecycle logic
- **Create:** Draft URL if `version` unset; locked URL if set
- **Read:** URL matches `version` from state; 404 → remove from state
- **Update (draft):** `PATCH /pipelines/{id}/inputs/{input_id}/` in-place (same ID preserved)
- **Update (locked, payload change):** `ModifyPlan` forces `RequiresReplace` on `payload` — delete old + create new (locked inputs are immutable; produces new `input_id`)
- **`loadPipelineInputIntoModel`:** Does NOT read `VersionID` from API response — `VersionID` is an internal DB FK (auto-increment), not the user-facing version number. `data.Version` is preserved from plan/state.

### ModifyPlan
```go
// Forces RequiresReplace when payload changes on a locked input, preventing
// "Provider produced inconsistent result" (locked update delete+recreates → new ID,
// but UseStateForUnknown would plan the old ID).
if !state.Version.IsNull() && !state.Version.IsUnknown() &&
    !normalizedPayloadsEqual(plan.Payload, state.Payload) {
    resp.RequiresReplace.Append(path.Root("payload"))
}
```

---

## `datarobot_pipeline_schedule` — `pkg/provider/pipeline_schedule_resource.go`

### Schema attributes

| Attribute | TF Type | Behavior |
|---|---|---|
| `id` | string, computed | = ScheduleID (`"id"` in API) |
| `pipeline_id` | string, required | Forces replace on change |
| `version` | int, required | Forces replace on change |
| `pipeline_input_id` | string, required | Forces replace on change; NOT returned by GET — preserved from state |
| `cron_expression` | string, required | Patchable in-place |
| `timezone` | string, optional, default `"UTC"` | Patchable in-place |
| `status` | string, computed | `ACTIVE` / `PAUSED` |

### Notes
- `ValidateConfig` rejects `version < 1`
- Schedule creation requires k8s CronJob RBAC — **not yet wired on staging**. Acceptance test `TestAccPipelineScheduleResource` is gated behind `ACCEPTANCE_RUN_PIPELINES_SCHEDULING=1`.

---

## `datarobot_pipeline_image` — `pkg/provider/pipeline_image_resource.go`

### Schema attributes

| Attribute | TF Type | Behavior |
|---|---|---|
| `id` | string, computed | = ImageID (`"id"` in API) |
| `name` | string, required | Forces replace |
| `description` | string, optional | Forces replace (no PATCH for description) |
| `packages` | list(string), required, min 1 | Append-only; removal forces replace via `ModifyPlan` |
| `latest_version` | int, computed | |
| `latest_status` | string, computed | `CREATING` / `READY` / `ERROR` |
| `created_at` / `updated_at` | string, computed | |

### Lifecycle logic
- **Update (superset):** `PATCH /pipelines/images/{id}/` sends only the newly added packages
- **Update (removal):** `ModifyPlan` forces `RequiresReplace` when any package is removed from the list
- **Read:** `latest_status` comes from `versions[0].status` (not a top-level field in `PipelineImageResponse`)

---

## Tests

### Mock-based integration tests (`pkg/provider/pipeline_*_resource_test.go`)
Use `IsUnitTest: true` + `HookGlobal(&NewService, ...)`. No real API needed.

**Read call counts per test pattern (critical for mock setup):**
- **1-step:** 1 GetX total (post-create serves as pre-destroy too)
- **2-step update (same ID):** 3 GetX (post-create + pre-step2-plan + pre-destroy-plan)
- **2-step replace (new ID):** GetX(id1) ×2 (post-create + pre-replace-plan) + GetX(id2) ×1 (post-replace/pre-destroy combined)

### Acceptance tests (`pkg/provider/pipeline_acc_test.go`)
Hit real staging API. Require `TF_ACC=1` + `DATAROBOT_API_TOKEN` set (via `.env`).

```bash
source .env && TF_ACC=1 go test ./pkg/provider/... -run "TestAccPipeline" -v -timeout 10m
```

Tests: `TestAccPipelineResource`, `TestAccPipelineReplaceOnDescriptionChange`, `TestAccPipelineImageResource`, `TestAccPipelineImageReplaceOnNameChange`, `TestAccPipelineInputDraftResource`.

`TestAccPipelineScheduleResource` — skipped by default; set `ACCEPTANCE_RUN_PIPELINES_SCHEDULING=1` when scheduling is wired.

**Cleanup:** The Terraform testing framework runs `terraform destroy` after every test (pass or fail). Each test uses a unique pipeline function name (`pipeline_<nameSalt>_<testSuffix>`) to avoid 409 collisions between parallel runs.
