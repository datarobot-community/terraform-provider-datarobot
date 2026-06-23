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
type PipelineImageStatus string

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

	PipelineImageStatusCreating PipelineImageStatus = "CREATING"
	PipelineImageStatusReady    PipelineImageStatus = "READY"
	PipelineImageStatusError    PipelineImageStatus = "ERROR"
)

// Pipeline types

type PipelineVersion struct {
	Version       int                   `json:"version"`
	Status        PipelineVersionStatus `json:"status"`
	TaskNames     []string              `json:"taskNames,omitempty"`
	PythonVersion string                `json:"pythonVersion"`
	ErrorDetail   *string               `json:"errorDetail,omitempty"`
	CreatedAt     string                `json:"createdAt"`
}

// Pipeline maps to both PipelineCreateResponse and PipelineDetailResponse.
// VersionNumber is populated by Create/Lock responses (top-level "version" field).
// Versions is populated by Get responses (detail array).
type Pipeline struct {
	PipelineID    string            `json:"id"`
	Name          string            `json:"name"`
	Description   *string           `json:"description,omitempty"`
	Mode          PipelineMode      `json:"mode"`
	IsActive      bool              `json:"isActive"`
	TaskNames     []string          `json:"taskNames,omitempty"`
	PythonVersion *string           `json:"pythonVersion,omitempty"`
	VersionNumber *int              `json:"version,omitempty"`
	Versions      []PipelineVersion `json:"versions"`
	CreatedAt     string            `json:"createdAt"`
	UpdatedAt     string            `json:"updatedAt"`
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

func (s *ServiceImpl) UpdatePipelineDraft(ctx context.Context, id string, fileName string, content []byte, description *string) (*Pipeline, error) {
	extraFields := map[string]string{}
	if description != nil {
		extraFields["description"] = *description
	}
	return uploadFileFromBinary[Pipeline](s.client, ctx, "/pipelines/"+id+"/", http.MethodPatch, fileName, content, extraFields)
}

func (s *ServiceImpl) LockPipeline(ctx context.Context, id string) (*Pipeline, error) {
	return Patch[Pipeline](s.client, ctx, "/pipelines/"+id+"/mode/", &lockPipelineModeRequest{Mode: PipelineModeLocked})
}

func (s *ServiceImpl) DeletePipeline(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/pipelines/"+id+"/")
}

// PipelineInput types

type PipelineInput struct {
	InputID    string             `json:"id"`
	PipelineID string             `json:"pipelineId"`
	VersionID  *int               `json:"versionId,omitempty"`
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
	ScheduleID     string                 `json:"id"`
	PipelineID     string                 `json:"pipelineId"`
	Version        int                    `json:"version"`
	CronExpression string                 `json:"cronExpression"`
	Timezone       string                 `json:"timezone"`
	Status         PipelineScheduleStatus `json:"status"`
	CreatedAt      string                 `json:"createdAt"`
	UpdatedAt      string                 `json:"updatedAt"`
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

// PipelineImage types

type PipelineImageVersion struct {
	Version     int                 `json:"version"`
	Packages    []string            `json:"packages"`
	Status      PipelineImageStatus `json:"status"`
	ErrorDetail *string             `json:"errorDetail,omitempty"`
	CreatedAt   string              `json:"createdAt"`
	UpdatedAt   string              `json:"updatedAt"`
}

type PipelineImage struct {
	ImageID       string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   *string                `json:"description,omitempty"`
	LatestVersion int                    `json:"latestVersion"`
	Versions      []PipelineImageVersion `json:"versions"`
	CreatedAt     string                 `json:"createdAt"`
	UpdatedAt     string                 `json:"updatedAt"`
}

type PipelineImageCreateRequest struct {
	Name        string   `json:"name"`
	Description *string  `json:"description,omitempty"`
	Packages    []string `json:"packages"`
}

type PipelineImageUpdateRequest struct {
	Packages []string `json:"packages"`
}

func (s *ServiceImpl) CreatePipelineImage(ctx context.Context, req *PipelineImageCreateRequest) (*PipelineImage, error) {
	return Post[PipelineImage](s.client, ctx, "/pipelines/images/", req)
}

func (s *ServiceImpl) GetPipelineImage(ctx context.Context, id string) (*PipelineImage, error) {
	return Get[PipelineImage](s.client, ctx, "/pipelines/images/"+id+"/")
}

func (s *ServiceImpl) UpdatePipelineImage(ctx context.Context, id string, req *PipelineImageUpdateRequest) (*PipelineImage, error) {
	return Patch[PipelineImage](s.client, ctx, "/pipelines/images/"+id+"/", req)
}

func (s *ServiceImpl) DeletePipelineImage(ctx context.Context, id string) error {
	return Delete(s.client, ctx, "/pipelines/images/"+id+"/")
}
