package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
				WaitForArtifactBuild(gomock.Any(), artifactID, buildID, waitOpts).
				Return(&client.ArtifactBuild{ID: buildID, Status: client.ArtifactBuildStatusCompleted}, nil),
			mockService.EXPECT().
				GetArtifact(gomock.Any(), artifactID).
				Return(draftWithImage, nil),
		)

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		artifact, gotBuildID, err := resource.syncArtifactBuild(context.Background(), artifactID, true, waitOpts)
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
		_, gotBuildID, err := resource.syncArtifactBuild(context.Background(), artifactID, false, nil)
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
		)

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		_, gotBuildID, err := resource.syncArtifactBuild(context.Background(), artifactID, true, nil)
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
		)

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		_, _, err := resource.syncArtifactBuild(context.Background(), artifactID, true, nil)
		if err == nil {
			t.Fatal("expected error for missing image_uri")
		}
	})
}

func boolPtr(v bool) *bool {
	return &v
}
