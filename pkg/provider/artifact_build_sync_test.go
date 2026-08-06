package provider

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestArtifactBuildWaitOptionsAddsOtelLogCallback(t *testing.T) {
	opts := artifactBuildWaitOptions(context.Background(), &client.WaitForArtifactBuildOptions{
		PollInterval: time.Millisecond,
	})
	if opts.OnOtelLogLine == nil {
		t.Fatal("expected OTEL log callback to be configured")
	}
	if opts.PollInterval != time.Millisecond {
		t.Fatalf("expected poll interval to be preserved, got %s", opts.PollInterval)
	}
}

func TestArtifactBuildNeededAfterUpload(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	plan := testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithBuildConfig()))
	draftArtifact := &client.Artifact{ID: "artifact-1", Status: client.ArtifactStatusDraft}
	lockedArtifact := &client.Artifact{ID: "artifact-1", Status: client.ArtifactStatusLocked}

	tests := []struct {
		name     string
		plan     *ArtifactResourceModel
		artifact *client.Artifact
		uploaded bool
		want     bool
	}{
		{
			name:     "uploaded draft with image build config",
			plan:     plan,
			artifact: draftArtifact,
			uploaded: true,
			want:     true,
		},
		{
			name:     "no upload",
			plan:     plan,
			artifact: draftArtifact,
			uploaded: false,
			want:     false,
		},
		{
			name:     "locked artifact",
			plan:     plan,
			artifact: lockedArtifact,
			uploaded: true,
			want:     false,
		},
		{
			name: "no image build config",
			plan: testSourcePlanModel(t, dir, testDraftSourceSpec(ArtifactContainerModel{
				Primary:  types.BoolValue(true),
				ImageURI: types.StringValue("nginx:latest"),
			})),
			artifact: draftArtifact,
			uploaded: true,
			want:     false,
		},
		{
			name:     "nil plan",
			plan:     nil,
			artifact: draftArtifact,
			uploaded: true,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := artifactBuildNeededAfterUpload(tt.plan, tt.artifact, tt.uploaded); got != tt.want {
				t.Fatalf("artifactBuildNeededAfterUpload() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestArtifactSourceWaitForBuild(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	tests := []struct {
		name string
		plan *ArtifactResourceModel
		want bool
	}{
		{
			name: "nil plan defaults true",
			plan: nil,
			want: true,
		},
		{
			name: "unset defaults true",
			plan: testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithBuildConfig())),
			want: true,
		},
		{
			name: "explicit true",
			plan: testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithBuildConfig()), func(m *ArtifactResourceModel) {
				m.Source.WaitForBuild = types.BoolValue(true)
			}),
			want: true,
		},
		{
			name: "explicit false",
			plan: testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithBuildConfig()), func(m *ArtifactResourceModel) {
				m.Source.WaitForBuild = types.BoolValue(false)
			}),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := artifactSourceWaitForBuild(tt.plan); got != tt.want {
				t.Fatalf("artifactSourceWaitForBuild() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestArtifactPrimaryContainerImageURI(t *testing.T) {
	t.Parallel()

	primaryURI := "registry.example/app:primary"
	sidecarURI := "registry.example/app:sidecar"
	fallbackURI := "registry.example/app:fallback"

	tests := []struct {
		name     string
		artifact *client.Artifact
		want     string
	}{
		{
			name:     "nil artifact",
			artifact: nil,
			want:     "",
		},
		{
			name: "explicit primary",
			artifact: &client.Artifact{
				Spec: client.ArtifactSpec{
					ContainerGroups: []client.ArtifactContainerGroup{{
						Containers: []client.ArtifactContainer{
							{ImageURI: sidecarURI, Primary: boolPtr(false)},
							{ImageURI: primaryURI, Primary: boolPtr(true)},
						},
					}},
				},
			},
			want: primaryURI,
		},
		{
			name: "fallback first container",
			artifact: &client.Artifact{
				Spec: client.ArtifactSpec{
					ContainerGroups: []client.ArtifactContainerGroup{{
						Containers: []client.ArtifactContainer{
							{ImageURI: fallbackURI},
							{ImageURI: "registry.example/app:other"},
						},
					}},
				},
			},
			want: fallbackURI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := artifactPrimaryContainerImageURI(tt.artifact); got != tt.want {
				t.Fatalf("artifactPrimaryContainerImageURI() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSyncArtifactBuild(t *testing.T) {
	t.Parallel()

	const (
		buildID  = "build-1"
		imageURI = "registry.example/app:build-1"
	)

	t.Run("waits for build and refreshes artifact", func(t *testing.T) {
		const artifactID = "artifact-wait"
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)

		draftWithImage := &client.Artifact{
			ID:     artifactID,
			Status: client.ArtifactStatusDraft,
			Spec: client.ArtifactSpec{
				ContainerGroups: []client.ArtifactContainerGroup{{
					Containers: []client.ArtifactContainer{{
						ImageURI: imageURI,
						Primary:  boolPtr(true),
					}},
				}},
			},
		}

		waitOpts := &client.WaitForArtifactBuildOptions{PollInterval: time.Millisecond}

		gomock.InOrder(
			mockService.EXPECT().
				TriggerArtifactBuild(gomock.Any(), artifactID).
				Return(&client.ArtifactBuildTriggerResponse{BuildIDs: []string{buildID}}, nil),
			mockService.EXPECT().
				WaitForArtifactBuild(gomock.Any(), artifactID, buildID, gomock.Any()).
				Return(&client.ArtifactBuild{ID: buildID, Status: client.ArtifactBuildStatusCompleted}, nil),
			mockService.EXPECT().
				GetArtifact(gomock.Any(), artifactID).
				Return(draftWithImage, nil),
		)

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		artifact, gotBuildID, err := resource.syncArtifactBuild(context.Background(), artifactID, "repo-1", true, waitOpts)
		if err != nil {
			t.Fatalf("syncArtifactBuild() error = %v", err)
		}
		if gotBuildID != buildID {
			t.Fatalf("buildID = %q, want %q", gotBuildID, buildID)
		}
		if artifactPrimaryContainerImageURI(artifact) != imageURI {
			t.Fatalf("image_uri = %q, want %q", artifactPrimaryContainerImageURI(artifact), imageURI)
		}
	})

	t.Run("trigger only without wait", func(t *testing.T) {
		const artifactID = "artifact-trigger-only"
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)

		draftWithImage := &client.Artifact{
			ID:     artifactID,
			Status: client.ArtifactStatusDraft,
			Spec: client.ArtifactSpec{
				ContainerGroups: []client.ArtifactContainerGroup{{
					Containers: []client.ArtifactContainer{{
						ImageURI: imageURI,
						Primary:  boolPtr(true),
					}},
				}},
			},
		}

		gomock.InOrder(
			mockService.EXPECT().
				TriggerArtifactBuild(gomock.Any(), artifactID).
				Return(&client.ArtifactBuildTriggerResponse{BuildIDs: []string{buildID}}, nil),
			mockService.EXPECT().
				GetArtifact(gomock.Any(), artifactID).
				Return(draftWithImage, nil),
		)

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		_, gotBuildID, err := resource.syncArtifactBuild(context.Background(), artifactID, "repo-1", false, nil)
		if err != nil {
			t.Fatalf("syncArtifactBuild() error = %v", err)
		}
		if gotBuildID != buildID {
			t.Fatalf("buildID = %q, want %q", gotBuildID, buildID)
		}
	})

	t.Run("wait failure surfaces build id", func(t *testing.T) {
		const artifactID = "artifact-failed"
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)

		buildErr := &client.ArtifactBuildFailedError{BuildID: buildID, Status: client.ArtifactBuildStatusFailed}

		gomock.InOrder(
			mockService.EXPECT().
				TriggerArtifactBuild(gomock.Any(), artifactID).
				Return(&client.ArtifactBuildTriggerResponse{BuildIDs: []string{buildID}}, nil),
			mockService.EXPECT().
				WaitForArtifactBuild(gomock.Any(), artifactID, buildID, gomock.Any()).
				Return(&client.ArtifactBuild{ID: buildID, Status: client.ArtifactBuildStatusFailed}, buildErr),
			mockService.EXPECT().
				BaseURL().
				Return("https://app.datarobot.com"),
			mockService.EXPECT().
				GetArtifactBuildLogs(gomock.Any(), artifactID, buildID).
				Return("[2026-06-09 10:00:00] ERROR: docker build failed", nil),
		)

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		_, gotBuildID, err := resource.syncArtifactBuild(context.Background(), artifactID, "repo-1", true, nil)
		if err == nil {
			t.Fatal("expected error")
		}
		if gotBuildID != buildID {
			t.Fatalf("buildID = %q, want %q", gotBuildID, buildID)
		}
		var failedErr *client.ArtifactBuildFailedError
		if !errors.As(err, &failedErr) {
			t.Fatalf("expected ArtifactBuildFailedError, got %T: %v", err, err)
		}
		if !strings.Contains(err.Error(), "docker build failed") {
			t.Fatalf("expected enriched logs in error, got: %v", err)
		}
		if !strings.Contains(err.Error(), "https://app.datarobot.com/registry/service-artifacts/repo-1/artifacts/"+artifactID+"/build-log") {
			t.Fatalf("expected build-log UI URL in error, got: %v", err)
		}
	})

	t.Run("completed build without image_uri fails", func(t *testing.T) {
		const artifactID = "artifact-no-image-uri"
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)

		gomock.InOrder(
			mockService.EXPECT().
				TriggerArtifactBuild(gomock.Any(), artifactID).
				Return(&client.ArtifactBuildTriggerResponse{BuildIDs: []string{buildID}}, nil),
			mockService.EXPECT().
				WaitForArtifactBuild(gomock.Any(), artifactID, buildID, gomock.Any()).
				Return(&client.ArtifactBuild{ID: buildID, Status: client.ArtifactBuildStatusCompleted}, nil),
			mockService.EXPECT().
				GetArtifact(gomock.Any(), artifactID).
				Return(&client.Artifact{
					ID:     artifactID,
					Status: client.ArtifactStatusDraft,
					Spec: client.ArtifactSpec{
						ContainerGroups: []client.ArtifactContainerGroup{{
							Containers: []client.ArtifactContainer{{
								Primary: boolPtr(true),
							}},
						}},
					},
				}, nil),
			mockService.EXPECT().
				BaseURL().
				Return("https://app.datarobot.com"),
			mockService.EXPECT().
				GetArtifactBuildLogs(gomock.Any(), artifactID, buildID).
				Return("", nil),
		)

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		_, _, err := resource.syncArtifactBuild(context.Background(), artifactID, "repo-1", true, nil)
		if err == nil {
			t.Fatal("expected error for missing image_uri")
		}
	})
}

func TestArtifactModifyPlanNeedsUnknownImageURI(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	spec := testDraftSourceSpec(testPrimaryWithBuildConfig())
	dirHash, err := computeFolderHash(types.StringValue(dir))
	if err != nil {
		t.Fatal(err)
	}
	draftPlan := testSourcePlanModel(t, dir, spec, func(m *ArtifactResourceModel) {
		m.Source.DirHash = dirHash
	})
	lockedPlan := testSourcePlanModel(t, dir, spec, func(m *ArtifactResourceModel) {
		m.Status = types.StringValue("locked")
		m.Source.DirHash = dirHash
	})

	draftState := func(hash types.String) *ArtifactResourceModel {
		state := testSourcePlanModel(t, dir, spec, func(m *ArtifactResourceModel) {
			m.Source.DirHash = hash
			m.ArtifactID = types.StringValue("artifact-1")
		})
		return state
	}

	tests := []struct {
		name     string
		plan     *ArtifactResourceModel
		state    *ArtifactResourceModel
		isCreate bool
		want     bool
	}{
		{
			name:     "create with source and build config",
			plan:     draftPlan,
			isCreate: true,
			want:     true,
		},
		{
			name:     "locked create with source",
			plan:     lockedPlan,
			isCreate: true,
			want:     true,
		},
		{
			name:     "no source",
			plan:     &ArtifactResourceModel{Spec: spec},
			isCreate: true,
			want:     false,
		},
		{
			name:  "draft update source unchanged",
			plan:  draftPlan,
			state: draftState(dirHash),
			want:  false,
		},
		{
			name: "draft update source changed",
			plan: testSourcePlanModel(t, dir, spec, func(m *ArtifactResourceModel) {
				m.Source.DirHash = types.StringValue("new-hash")
			}),
			state: draftState(dirHash),
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := artifactModifyPlanNeedsUnknownImageURI(tt.plan, tt.state, tt.isCreate); got != tt.want {
				t.Fatalf("artifactModifyPlanNeedsUnknownImageURI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplySourceManagedImageURIToPlan(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	spec := testDraftSourceSpec(testPrimaryWithBuildConfig())
	dirHash, err := computeFolderHash(types.StringValue(dir))
	if err != nil {
		t.Fatal(err)
	}

	setKnownImageURIs := func(plan *ArtifactResourceModel) {
		for gi := range plan.Spec.ContainerGroups {
			for ci := range plan.Spec.ContainerGroups[gi].Containers {
				plan.Spec.ContainerGroups[gi].Containers[ci].ImageURI = types.StringValue("nginx:latest")
			}
		}
	}

	draftPlan := testSourcePlanModel(t, dir, spec, func(m *ArtifactResourceModel) {
		m.Source.DirHash = dirHash
	})
	lockedPlan := testSourcePlanModel(t, dir, spec, func(m *ArtifactResourceModel) {
		m.Status = types.StringValue("locked")
		m.Source.DirHash = dirHash
	})

	draftState := func(hash types.String) *ArtifactResourceModel {
		return testSourcePlanModel(t, dir, spec, func(m *ArtifactResourceModel) {
			m.Source.DirHash = hash
			m.ArtifactID = types.StringValue("artifact-1")
		})
	}
	lockedState := func(hash types.String) *ArtifactResourceModel {
		return testSourcePlanModel(t, dir, spec, func(m *ArtifactResourceModel) {
			m.Status = types.StringValue("locked")
			m.Source.DirHash = hash
			m.ArtifactID = types.StringValue("artifact-1")
		})
	}

	tests := []struct {
		name               string
		plan               *ArtifactResourceModel
		state              *ArtifactResourceModel
		isCreate           bool
		wantPrimaryUnknown bool
	}{
		{
			name:               "create with source and build config",
			plan:               draftPlan,
			isCreate:           true,
			wantPrimaryUnknown: true,
		},
		{
			name:               "locked create with source",
			plan:               lockedPlan,
			isCreate:           true,
			wantPrimaryUnknown: true,
		},
		{
			name: "draft update source changed",
			plan: testSourcePlanModel(t, dir, spec, func(m *ArtifactResourceModel) {
				m.Source.DirHash = types.StringValue("new-hash")
			}),
			state:              draftState(dirHash),
			wantPrimaryUnknown: true,
		},
		{
			name: "locked to locked source changed",
			plan: testSourcePlanModel(t, dir, spec, func(m *ArtifactResourceModel) {
				m.Status = types.StringValue("locked")
				m.Source.DirHash = types.StringValue("new-hash")
			}),
			state:              lockedState(dirHash),
			wantPrimaryUnknown: true,
		},
		{
			name: "draft to locked source changed",
			plan: testSourcePlanModel(t, dir, spec, func(m *ArtifactResourceModel) {
				m.Status = types.StringValue("locked")
				m.Source.DirHash = types.StringValue("new-hash")
			}),
			state:              draftState(dirHash),
			wantPrimaryUnknown: true,
		},
		{
			name:               "no source",
			plan:               &ArtifactResourceModel{Spec: spec},
			isCreate:           true,
			wantPrimaryUnknown: false,
		},
		{
			name:               "draft update source unchanged",
			plan:               draftPlan,
			state:              draftState(dirHash),
			wantPrimaryUnknown: false,
		},
		{
			name:               "locked to locked source unchanged",
			plan:               lockedPlan,
			state:              lockedState(dirHash),
			wantPrimaryUnknown: false,
		},
		{
			name: "draft to locked source unchanged",
			plan: testSourcePlanModel(t, dir, spec, func(m *ArtifactResourceModel) {
				m.Status = types.StringValue("locked")
				m.Source.DirHash = dirHash
			}),
			state:              draftState(dirHash),
			wantPrimaryUnknown: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := cloneArtifactResourceModel(tt.plan)
			setKnownImageURIs(plan)

			applySourceManagedImageURIToPlan(plan, tt.state, tt.isCreate)

			got := primaryPlanImageURI(plan)
			if tt.wantPrimaryUnknown {
				if !got.IsUnknown() {
					t.Fatalf("primary image_uri = %v, want unknown", got)
				}
				return
			}
			if got.IsUnknown() {
				t.Fatalf("primary image_uri = unknown, want unchanged known value")
			}
			if got.ValueString() != "nginx:latest" {
				t.Fatalf("primary image_uri = %q, want nginx:latest", got.ValueString())
			}
		})
	}

	t.Run("non-primary container image_uri unchanged", func(t *testing.T) {
		multiSpec := testDraftSourceSpec(testPrimaryWithBuildConfig(), testSidecarWithBuildConfig())
		plan := testSourcePlanModel(t, dir, multiSpec, func(m *ArtifactResourceModel) {
			m.Source.DirHash = dirHash
		})
		setKnownImageURIs(plan)

		applySourceManagedImageURIToPlan(plan, nil, true)

		primary := primaryPlanImageURI(plan)
		if !primary.IsUnknown() {
			t.Fatalf("primary image_uri = %v, want unknown", primary)
		}

		sidecar := plan.Spec.ContainerGroups[0].Containers[1].ImageURI
		if sidecar.IsUnknown() || sidecar.ValueString() != "nginx:latest" {
			t.Fatalf("sidecar image_uri = %v, want nginx:latest", sidecar)
		}
	})
}

func TestApplyCompletedArtifactBuildToPrimaryContainer(t *testing.T) {
	t.Parallel()

	primary := true
	artifact := &client.Artifact{
		Spec: client.ArtifactSpec{
			ContainerGroups: []client.ArtifactContainerGroup{{
				Containers: []client.ArtifactContainer{{
					Primary: &primary,
					Build: &client.ArtifactContainerBuildInfo{
						ArtifactImageBuildID: "stale-build",
						Status:               client.ArtifactBuildStatusCompleted,
						CreatedAt:            "2026-01-01T00:00:00Z",
					},
				}},
			}},
		},
	}
	build := &client.ArtifactBuild{
		ID:        "fresh-build",
		Status:    client.ArtifactBuildStatusCompleted,
		CreatedAt: "2026-02-01T00:00:00Z",
	}

	applyCompletedArtifactBuildToPrimaryContainer(artifact, build)

	got := artifact.Spec.ContainerGroups[0].Containers[0].Build
	if got == nil || got.ArtifactImageBuildID != "fresh-build" {
		t.Fatalf("build id = %#v, want fresh-build", got)
	}
	if got.CreatedAt != "2026-02-01T00:00:00Z" {
		t.Fatalf("created_at = %q", got.CreatedAt)
	}
}

func cloneArtifactResourceModel(src *ArtifactResourceModel) *ArtifactResourceModel {
	if src == nil {
		return nil
	}
	dst := *src
	if src.Source != nil {
		source := *src.Source
		dst.Source = &source
	}
	if src.Spec != nil {
		spec := *src.Spec
		spec.ContainerGroups = make([]ArtifactContainerGroupModel, len(src.Spec.ContainerGroups))
		for gi, group := range src.Spec.ContainerGroups {
			spec.ContainerGroups[gi] = group
			spec.ContainerGroups[gi].Containers = make([]ArtifactContainerModel, len(group.Containers))
			copy(spec.ContainerGroups[gi].Containers, group.Containers)
		}
		dst.Spec = &spec
	}
	return &dst
}

func primaryPlanImageURI(plan *ArtifactResourceModel) types.String {
	if plan == nil || plan.Spec == nil {
		return types.StringNull()
	}
	for _, group := range plan.Spec.ContainerGroups {
		for _, container := range group.Containers {
			if artifactContainerIsPrimary(container, group) {
				return container.ImageURI
			}
		}
	}
	return types.StringNull()
}

func boolPtr(v bool) *bool {
	return &v
}
