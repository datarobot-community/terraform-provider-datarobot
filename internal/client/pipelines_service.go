package client

import (
	"context"
	"fmt"
	"net/http"
)

type PipelineMode string
type PipelineVersionStatus string
type PipelineInputState string
type PipelineScheduleStatus string
type PipelineEnvironmentStatus string

const (
	PipelineModeDraft  PipelineMode = "draft"
	PipelineModeLocked PipelineMode = "locked"

	PipelineVersionStatusPending PipelineVersionStatus = "PENDING"
	PipelineVersionStatusReady   PipelineVersionStatus = "READY"
	PipelineVersionStatusFailed  PipelineVersionStatus = "FAILED"

	PipelineInputStateValid   PipelineInputState = "VALID"
	PipelineInputStateInvalid PipelineInputState = "INVALID"

	PipelineScheduleStatusActive  PipelineScheduleStatus = "ACTIVE"
	PipelineScheduleStatusPaused  PipelineScheduleStatus = "PAUSED"
	PipelineScheduleStatusDeleted PipelineScheduleStatus = "DELETED"

	PipelineEnvironmentStatusCreating PipelineEnvironmentStatus = "CREATING"
	PipelineEnvironmentStatusReady    PipelineEnvironmentStatus = "READY"
	PipelineEnvironmentStatusError    PipelineEnvironmentStatus = "ERROR"
)

// Pipeline types

type PipelineVersion struct {
	Version       int                   `json:"version"`
	Status        PipelineVersionStatus `json:"status"`
	LatticeName   string                `json:"lattice_name"`
	ElectronNames []string              `json:"electron_names,omitempty"`
	PythonVersion string                `json:"python_version"`
	ErrorDetail   *string               `json:"error_detail,omitempty"`
	CreatedAt     string                `json:"created_at"`
}

// Pipeline maps to the PipelineDetailResponse schema.
// LatticeName is the @pipeline-decorated function name; ElectronNames are @task names.
type Pipeline struct {
	PipelineID    string            `json:"pipeline_id"`
	Name          string            `json:"name"`
	Description   *string           `json:"description,omitempty"`
	Mode          PipelineMode      `json:"mode"`
	IsActive      bool              `json:"is_active"`
	LatticeName   *string           `json:"lattice_name,omitempty"`
	ElectronNames []string          `json:"electron_names,omitempty"`
	PythonVersion *string           `json:"python_version,omitempty"`
	Versions      []PipelineVersion `json:"versions"`
	CreatedAt     string            `json:"created_at"`
	UpdatedAt     string            `json:"updated_at"`
}

type lockPipelineModeRequest struct {
	Mode PipelineMode `json:"mode"`
}

func (s *ServiceImpl) CreatePipeline(ctx context.Context, fileName string, content []byte, description *string) (*Pipeline, error) {
	extraFields := map[string]string{}
	if description != nil && *description != "" {
		extraFields["description"] = *description
	}
	return uploadFileFromBinary[Pipeline](s.client, ctx, "/pipelines/", http.MethodPost, fileName, content, extraFields)
}

func (s *ServiceImpl) GetPipeline(ctx context.Context, id string) (*Pipeline, error) {
	return Get[Pipeline](s.client, ctx, "/pipelines/"+id+"/")
}

func (s *ServiceImpl) UpdatePipelineDraft(ctx context.Context, id string, fileName string, content []byte) (*Pipeline, error) {
	return uploadFileFromBinary[Pipeline](s.client, ctx, "/pipelines/"+id+"/", http.MethodPatch, fileName, content, map[string]string{})
}

func (s *ServiceImpl) LockPipeline(ctx context.Context, id string) (*Pipeline, error) {
	return Patch[Pipeline](s.client, ctx, "/pipelines/"+id+"/mode/", &lockPipelineModeRequest{Mode: PipelineModeLocked})
}

func (s *ServiceImpl) DeletePipeline(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/pipelines/"+id+"/")
}

// PipelineInput types

type PipelineInput struct {
	InputID    string             `json:"input_id"`
	PipelineID string             `json:"pipeline_id"`
	VersionID  *int               `json:"version_id,omitempty"`
	IsDraft    bool               `json:"is_draft"`
	Payload    map[string]any     `json:"payload"`
	State      PipelineInputState `json:"state"`
	CreatedAt  string             `json:"created_at"`
	UpdatedAt  string             `json:"updated_at"`
}

type PipelineInputCreateRequest struct {
	Payload map[string]any `json:"payload"`
}

type PipelineInputUpdateRequest struct {
	Payload map[string]any `json:"payload"`
}

func (s *ServiceImpl) CreateDraftPipelineInput(ctx context.Context, pipelineID string, req *PipelineInputCreateRequest) (*PipelineInput, error) {
	return Post[PipelineInput](s.client, ctx, fmt.Sprintf("/pipelines/%s/inputs/", pipelineID), req)
}

func (s *ServiceImpl) CreateLockedPipelineInput(ctx context.Context, pipelineID string, version int, req *PipelineInputCreateRequest) (*PipelineInput, error) {
	return Post[PipelineInput](s.client, ctx, fmt.Sprintf("/pipelines/%s/versions/%d/inputs/", pipelineID, version), req)
}

func (s *ServiceImpl) GetDraftPipelineInput(ctx context.Context, pipelineID, inputID string) (*PipelineInput, error) {
	return Get[PipelineInput](s.client, ctx, fmt.Sprintf("/pipelines/%s/inputs/%s/", pipelineID, inputID))
}

func (s *ServiceImpl) GetLockedPipelineInput(ctx context.Context, pipelineID string, version int, inputID string) (*PipelineInput, error) {
	return Get[PipelineInput](s.client, ctx, fmt.Sprintf("/pipelines/%s/versions/%d/inputs/%s/", pipelineID, version, inputID))
}

func (s *ServiceImpl) UpdateDraftPipelineInput(ctx context.Context, pipelineID, inputID string, req *PipelineInputUpdateRequest) (*PipelineInput, error) {
	return Patch[PipelineInput](s.client, ctx, fmt.Sprintf("/pipelines/%s/inputs/%s/", pipelineID, inputID), req)
}

func (s *ServiceImpl) DeleteDraftPipelineInput(ctx context.Context, pipelineID, inputID string) error {
	return Delete(s.client, ctx, fmt.Sprintf("/pipelines/%s/inputs/%s/", pipelineID, inputID))
}

func (s *ServiceImpl) DeleteLockedPipelineInput(ctx context.Context, pipelineID string, version int, inputID string) error {
	return Delete(s.client, ctx, fmt.Sprintf("/pipelines/%s/versions/%d/inputs/%s/", pipelineID, version, inputID))
}

// PipelineSchedule types

type PipelineSchedule struct {
	ScheduleID     string                `json:"schedule_id"`
	PipelineID     string                `json:"pipeline_id"`
	Version        int                   `json:"version"`
	CronExpression string                `json:"cron_expression"`
	Timezone       string                `json:"timezone"`
	Status         PipelineScheduleStatus `json:"status"`
	CreatedAt      string                `json:"created_at"`
	UpdatedAt      string                `json:"updated_at"`
}

// PipelineInputID is intentionally absent from PipelineSchedule: the GET endpoint does not return it.
// The provider stores it in state as a replace-trigger attribute only.

type PipelineScheduleCreateRequest struct {
	CronExpression  string `json:"cron_expression"`
	PipelineInputID string `json:"pipeline_input_id"`
	Timezone        string `json:"timezone,omitempty"`
}

type PipelineScheduleUpdateRequest struct {
	CronExpression *string `json:"cron_expression,omitempty"`
	Timezone       *string `json:"timezone,omitempty"`
}

func (s *ServiceImpl) CreatePipelineSchedule(ctx context.Context, pipelineID string, version int, req *PipelineScheduleCreateRequest) (*PipelineSchedule, error) {
	return Post[PipelineSchedule](s.client, ctx, fmt.Sprintf("/pipelines/%s/versions/%d/schedules/", pipelineID, version), req)
}

func (s *ServiceImpl) GetPipelineSchedule(ctx context.Context, pipelineID string, version int, scheduleID string) (*PipelineSchedule, error) {
	return Get[PipelineSchedule](s.client, ctx, fmt.Sprintf("/pipelines/%s/versions/%d/schedules/%s/", pipelineID, version, scheduleID))
}

func (s *ServiceImpl) UpdatePipelineSchedule(ctx context.Context, pipelineID string, version int, scheduleID string, req *PipelineScheduleUpdateRequest) (*PipelineSchedule, error) {
	return Patch[PipelineSchedule](s.client, ctx, fmt.Sprintf("/pipelines/%s/versions/%d/schedules/%s/", pipelineID, version, scheduleID), req)
}

func (s *ServiceImpl) DeletePipelineSchedule(ctx context.Context, pipelineID string, version int, scheduleID string) error {
	return Delete(s.client, ctx, fmt.Sprintf("/pipelines/%s/versions/%d/schedules/%s/", pipelineID, version, scheduleID))
}

// PipelineEnvironment types

type PipelineEnvironmentVersion struct {
	Version     int                       `json:"version"`
	Packages    []string                  `json:"packages"`
	Status      PipelineEnvironmentStatus `json:"status"`
	ErrorDetail *string                   `json:"error_detail,omitempty"`
	CreatedAt   string                    `json:"created_at"`
	UpdatedAt   string                    `json:"updated_at"`
}

type PipelineEnvironment struct {
	EnvironmentID string                       `json:"environment_id"`
	Name          string                       `json:"name"`
	Description   *string                      `json:"description,omitempty"`
	LatestVersion int                          `json:"latest_version"`
	Versions      []PipelineEnvironmentVersion `json:"versions"`
	CreatedAt     string                       `json:"created_at"`
	UpdatedAt     string                       `json:"updated_at"`
}

type PipelineEnvironmentCreateRequest struct {
	Name        string   `json:"name"`
	Description *string  `json:"description,omitempty"`
	Packages    []string `json:"packages"`
}

type PipelineEnvironmentUpdateRequest struct {
	Packages []string `json:"packages"`
}

func (s *ServiceImpl) CreatePipelineEnvironment(ctx context.Context, req *PipelineEnvironmentCreateRequest) (*PipelineEnvironment, error) {
	return Post[PipelineEnvironment](s.client, ctx, "/pipelines/environments/", req)
}

func (s *ServiceImpl) GetPipelineEnvironment(ctx context.Context, id string) (*PipelineEnvironment, error) {
	return Get[PipelineEnvironment](s.client, ctx, "/pipelines/environments/"+id+"/")
}

func (s *ServiceImpl) UpdatePipelineEnvironment(ctx context.Context, id string, req *PipelineEnvironmentUpdateRequest) (*PipelineEnvironment, error) {
	return Patch[PipelineEnvironment](s.client, ctx, "/pipelines/environments/"+id+"/", req)
}

func (s *ServiceImpl) DeletePipelineEnvironment(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/pipelines/environments/"+id+"/")
}
