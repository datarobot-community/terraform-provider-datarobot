package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

// capturedRequest holds the most recent request seen by the test server.
// Fields are populated by the handler after newTestPipelinesService returns,
// so callers must read through the pointer, not a value copied at setup time.
type capturedRequest struct {
	req  *http.Request
	body []byte
}

// newTestPipelinesService spins up an httptest server that records the last
// request's method/path/body and responds with a canned success payload, and
// returns a Service wired to talk to it. Pipelines API routes are registered
// without trailing slashes and the live app disables Starlette's automatic
// trailing-slash redirect (redirect_slashes=False), so a client that sends a
// trailing slash gets a bare 404 rather than a redirect.
func newTestPipelinesService(t *testing.T, body string) (Service, *capturedRequest) {
	t.Helper()

	captured := &capturedRequest{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.req = r
		if r.Body != nil {
			captured.body, _ = io.ReadAll(r.Body)
			r.Body.Close()
		}

		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	cfg := NewConfiguration("test-token")
	cfg.Endpoint = server.URL
	cfg.HTTPClient = server.Client()
	svc := NewService(NewClient(cfg))

	return svc, captured
}

// TestPipelinesServiceRequestPaths asserts every Pipelines API client method
// sends the exact route registered by pipelines-api (no trailing slash).
// This is a regression test: pipelines-api mounts all these routes without a
// trailing slash and disables redirect_slashes, so a mismatched trailing
// slash on the client side previously caused every call to 404.
func TestPipelinesServiceRequestPaths(t *testing.T) {
	const pipelineID = "pipe-1"
	const inputID = "input-1"
	const scheduleID = "sched-1"
	const imageID = "img-1"
	const version = 2

	cases := []struct {
		name         string
		method       string
		expectedPath string
		call         func(ctx context.Context, svc Service) error
	}{
		{"CreatePipeline", http.MethodPost, "/pipelines", func(ctx context.Context, svc Service) error {
			_, err := svc.CreatePipeline(ctx, "f.py", []byte("x"), nil)
			return err
		}},
		{"GetPipeline", http.MethodGet, "/pipelines/" + pipelineID, func(ctx context.Context, svc Service) error {
			_, err := svc.GetPipeline(ctx, pipelineID)
			return err
		}},
		{"UpdatePipelineDraft", http.MethodPatch, "/pipelines/" + pipelineID, func(ctx context.Context, svc Service) error {
			_, err := svc.UpdatePipelineDraft(ctx, pipelineID, "f.py", []byte("x"), nil)
			return err
		}},
		{"LockPipeline", http.MethodPatch, "/pipelines/" + pipelineID + "/mode", func(ctx context.Context, svc Service) error {
			_, err := svc.LockPipeline(ctx, pipelineID)
			return err
		}},
		{"DeletePipeline", http.MethodDelete, "/pipelines/" + pipelineID, func(ctx context.Context, svc Service) error {
			return svc.DeletePipeline(ctx, pipelineID)
		}},
		{"CreateDraftPipelineInput", http.MethodPost, "/pipelines/" + pipelineID + "/inputs", func(ctx context.Context, svc Service) error {
			_, err := svc.CreateDraftPipelineInput(ctx, pipelineID, &PipelineInputCreateRequest{})
			return err
		}},
		{"CreateLockedPipelineInput", http.MethodPost, "/pipelines/" + pipelineID + "/versions/2/inputs", func(ctx context.Context, svc Service) error {
			_, err := svc.CreateLockedPipelineInput(ctx, pipelineID, version, &PipelineInputCreateRequest{})
			return err
		}},
		{"GetDraftPipelineInput", http.MethodGet, "/pipelines/" + pipelineID + "/inputs/" + inputID, func(ctx context.Context, svc Service) error {
			_, err := svc.GetDraftPipelineInput(ctx, pipelineID, inputID)
			return err
		}},
		{"GetLockedPipelineInput", http.MethodGet, "/pipelines/" + pipelineID + "/versions/2/inputs/" + inputID, func(ctx context.Context, svc Service) error {
			_, err := svc.GetLockedPipelineInput(ctx, pipelineID, version, inputID)
			return err
		}},
		{"UpdateDraftPipelineInput", http.MethodPatch, "/pipelines/" + pipelineID + "/inputs/" + inputID, func(ctx context.Context, svc Service) error {
			_, err := svc.UpdateDraftPipelineInput(ctx, pipelineID, inputID, &PipelineInputUpdateRequest{})
			return err
		}},
		{"DeleteDraftPipelineInput", http.MethodDelete, "/pipelines/" + pipelineID + "/inputs/" + inputID, func(ctx context.Context, svc Service) error {
			return svc.DeleteDraftPipelineInput(ctx, pipelineID, inputID)
		}},
		{"DeleteLockedPipelineInput", http.MethodDelete, "/pipelines/" + pipelineID + "/versions/2/inputs/" + inputID, func(ctx context.Context, svc Service) error {
			return svc.DeleteLockedPipelineInput(ctx, pipelineID, version, inputID)
		}},
		{"CreatePipelineSchedule", http.MethodPost, "/pipelines/" + pipelineID + "/versions/2/schedules", func(ctx context.Context, svc Service) error {
			_, err := svc.CreatePipelineSchedule(ctx, pipelineID, version, &PipelineScheduleCreateRequest{})
			return err
		}},
		{"GetPipelineSchedule", http.MethodGet, "/pipelines/" + pipelineID + "/versions/2/schedules/" + scheduleID, func(ctx context.Context, svc Service) error {
			_, err := svc.GetPipelineSchedule(ctx, pipelineID, version, scheduleID)
			return err
		}},
		{"UpdatePipelineSchedule", http.MethodPatch, "/pipelines/" + pipelineID + "/versions/2/schedules/" + scheduleID, func(ctx context.Context, svc Service) error {
			_, err := svc.UpdatePipelineSchedule(ctx, pipelineID, version, scheduleID, &PipelineScheduleUpdateRequest{})
			return err
		}},
		{"DeletePipelineSchedule", http.MethodDelete, "/pipelines/" + pipelineID + "/versions/2/schedules/" + scheduleID, func(ctx context.Context, svc Service) error {
			return svc.DeletePipelineSchedule(ctx, pipelineID, version, scheduleID)
		}},
		{"CreatePipelineImage", http.MethodPost, "/pipelines/images", func(ctx context.Context, svc Service) error {
			_, err := svc.CreatePipelineImage(ctx, &PipelineImageCreateRequest{Name: "n", Packages: []string{"numpy"}})
			return err
		}},
		{"GetPipelineImage", http.MethodGet, "/pipelines/images/" + imageID, func(ctx context.Context, svc Service) error {
			_, err := svc.GetPipelineImage(ctx, imageID)
			return err
		}},
		{"UpdatePipelineImage", http.MethodPatch, "/pipelines/images/" + imageID, func(ctx context.Context, svc Service) error {
			_, err := svc.UpdatePipelineImage(ctx, imageID, &PipelineImageUpdateRequest{Name: "n", Packages: []string{"numpy"}})
			return err
		}},
		{"DeletePipelineImage", http.MethodDelete, "/pipelines/images/" + imageID, func(ctx context.Context, svc Service) error {
			return svc.DeletePipelineImage(ctx, imageID)
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, captured := newTestPipelinesService(t, "{}")

			if err := tc.call(context.Background(), svc); err != nil {
				t.Fatalf("%s: unexpected error: %v", tc.name, err)
			}
			if captured.req == nil {
				t.Fatalf("%s: no request was recorded", tc.name)
			}
			if captured.req.Method != tc.method {
				t.Errorf("%s: expected method %s, got %s", tc.name, tc.method, captured.req.Method)
			}
			if captured.req.URL.Path != tc.expectedPath {
				t.Errorf("%s: expected path %q, got %q", tc.name, tc.expectedPath, captured.req.URL.Path)
			}
		})
	}
}

// TestGetPipelineImageDecodesNestedDefinitionAndImageURI is a regression test
// for a bug where PipelineImageVersion.Packages was declared as a top-level
// field, but the API nests packages (and pythonBaseImage) under a "definition"
// object and reports the built image location under "imageUri". The flat
// field silently decoded to an empty slice on every real response.
func TestGetPipelineImageDecodesNestedDefinitionAndImageURI(t *testing.T) {
	baseImage := "covalent-runtime-image:latest"
	imageURI := "123456789.dkr.ecr.us-east-1.amazonaws.com/pipelines/images/img-1:v1"

	body := `{
		"id": "img-1",
		"name": "my-image",
		"description": null,
		"latestVersion": 1,
		"versions": [
			{
				"version": 1,
				"definition": {
					"name": "my-image",
					"packages": ["numpy==1.26.0", "pandas>=2.0"],
					"pythonBaseImage": "` + baseImage + `"
				},
				"status": "READY",
				"errorDetail": null,
				"imageUri": "` + imageURI + `",
				"createdAt": "2025-01-01T00:00:00Z",
				"updatedAt": "2025-01-01T00:00:00Z"
			}
		],
		"createdAt": "2025-01-01T00:00:00Z",
		"updatedAt": "2025-01-01T00:00:00Z"
	}`

	svc, _ := newTestPipelinesService(t, body)

	image, err := svc.GetPipelineImage(context.Background(), "img-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(image.Versions) != 1 {
		t.Fatalf("expected 1 version, got %d", len(image.Versions))
	}
	v := image.Versions[0]

	wantPkgs := []string{"numpy==1.26.0", "pandas>=2.0"}
	if !slices.Equal(v.Definition.Packages, wantPkgs) {
		t.Errorf("expected packages %v, got %v", wantPkgs, v.Definition.Packages)
	}
	if v.Definition.PythonBaseImage == nil || *v.Definition.PythonBaseImage != baseImage {
		t.Errorf("expected pythonBaseImage %q, got %v", baseImage, v.Definition.PythonBaseImage)
	}
	if v.ImageURI == nil || *v.ImageURI != imageURI {
		t.Errorf("expected imageUri %q, got %v", imageURI, v.ImageURI)
	}
}

// TestUpdatePipelineImageRequestMarshalsRequiredFields ensures the request
// struct's JSON tags actually produce "name" (required by the API on every
// PATCH) and "packages", since PipelineImageUpdateRequest previously omitted
// "name" entirely, which the API rejects with a 422.
func TestUpdatePipelineImageRequestMarshalsRequiredFields(t *testing.T) {
	base := "python:3.12"
	req := &PipelineImageUpdateRequest{
		Name:            "my-image",
		Packages:        []string{"numpy==1.26.0", "pandas>=2.0"},
		PythonBaseImage: &base,
	}
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected error marshaling request: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unexpected error decoding marshaled request: %v", err)
	}

	if decoded["name"] != "my-image" {
		t.Errorf("expected \"name\" field to be present and equal to %q, got %v", "my-image", decoded["name"])
	}
	if _, ok := decoded["packages"]; !ok {
		t.Errorf("expected \"packages\" field to be present")
	}
	if decoded["pythonBaseImage"] != base {
		t.Errorf("expected \"pythonBaseImage\" field to be present and equal to %q, got %v", base, decoded["pythonBaseImage"])
	}
}

// TestUpdatePipelineImageRequestOmitsNilPythonBaseImage confirms the optional
// pythonBaseImage field is omitted (not sent as null) when unset, matching
// the API's "anyOf[string, null]" schema which prefers omission on PATCH.
func TestUpdatePipelineImageRequestOmitsNilPythonBaseImage(t *testing.T) {
	req := &PipelineImageUpdateRequest{Name: "my-image", Packages: []string{"numpy"}}
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected error marshaling request: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unexpected error decoding marshaled request: %v", err)
	}
	if _, ok := decoded["pythonBaseImage"]; ok {
		t.Errorf("expected \"pythonBaseImage\" to be omitted when nil, got %v", decoded["pythonBaseImage"])
	}
}
