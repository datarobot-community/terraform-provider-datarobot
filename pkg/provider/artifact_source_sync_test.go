package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/ignore"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/wapi"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestArtifactSourceConfigured(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	tests := []struct {
		name string
		data *ArtifactResourceModel
		want bool
	}{
		{
			name: "no source block",
			data: &ArtifactResourceModel{},
			want: false,
		},
		{
			name: "source without dir",
			data: &ArtifactResourceModel{
				Source: &ArtifactSourceModel{},
			},
			want: false,
		},
		{
			name: "null dir",
			data: &ArtifactResourceModel{
				Source: &ArtifactSourceModel{
					Dir: types.StringNull(),
				},
			},
			want: false,
		},
		{
			name: "unknown dir",
			data: &ArtifactResourceModel{
				Source: &ArtifactSourceModel{
					Dir: types.StringUnknown(),
				},
			},
			want: false,
		},
		{
			name: "known dir",
			data: &ArtifactResourceModel{
				Source: &ArtifactSourceModel{
					Dir: types.StringValue(dir),
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := artifactSourceConfigured(tt.data); got != tt.want {
				t.Fatalf("artifactSourceConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestArtifactSourceNeedsUpload(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	hashA := types.StringValue("hash-a")
	hashB := types.StringValue("hash-b")

	planWithSource := func(hash types.String) *ArtifactResourceModel {
		return &ArtifactResourceModel{
			Source: &ArtifactSourceModel{
				Dir:     types.StringValue(dir),
				DirHash: hash,
			},
		}
	}

	stateWithSource := func(hash types.String) *ArtifactResourceModel {
		return &ArtifactResourceModel{
			Source: &ArtifactSourceModel{
				Dir:     types.StringValue(dir),
				DirHash: hash,
			},
		}
	}

	tests := []struct {
		name            string
		plan            *ArtifactResourceModel
		state           *ArtifactResourceModel
		priorArtifactID string
		newArtifactID   string
		want            bool
	}{
		{
			name:            "create: no source on plan",
			plan:            &ArtifactResourceModel{},
			state:           nil,
			priorArtifactID: "",
			newArtifactID:   "artifact-new",
			want:            false,
		},
		{
			name:            "create: source configured with nil state",
			plan:            planWithSource(types.StringUnknown()),
			state:           nil,
			priorArtifactID: "",
			newArtifactID:   "artifact-new",
			want:            true,
		},
		{
			name:            "update: new artifact version always uploads",
			plan:            planWithSource(hashA),
			state:           stateWithSource(hashA),
			priorArtifactID: "artifact-old",
			newArtifactID:   "artifact-new",
			want:            true,
		},
		{
			name:            "update: nil state uploads",
			plan:            planWithSource(hashA),
			state:           nil,
			priorArtifactID: "artifact-1",
			newArtifactID:   "artifact-1",
			want:            true,
		},
		{
			name:            "update: source block newly added",
			plan:            planWithSource(hashA),
			state:           &ArtifactResourceModel{},
			priorArtifactID: "artifact-1",
			newArtifactID:   "artifact-1",
			want:            true,
		},
		{
			name:            "update: missing dir_hash in state uploads",
			plan:            planWithSource(hashA),
			state:           stateWithSource(types.StringNull()),
			priorArtifactID: "artifact-1",
			newArtifactID:   "artifact-1",
			want:            true,
		},
		{
			name:            "update: unknown dir_hash in state uploads",
			plan:            planWithSource(hashA),
			state:           stateWithSource(types.StringUnknown()),
			priorArtifactID: "artifact-1",
			newArtifactID:   "artifact-1",
			want:            true,
		},
		{
			name:            "update: unchanged hash skips upload",
			plan:            planWithSource(hashA),
			state:           stateWithSource(hashA),
			priorArtifactID: "artifact-1",
			newArtifactID:   "artifact-1",
			want:            false,
		},
		{
			name:            "update: changed hash triggers upload",
			plan:            planWithSource(hashB),
			state:           stateWithSource(hashA),
			priorArtifactID: "artifact-1",
			newArtifactID:   "artifact-1",
			want:            true,
		},
		{
			name:            "read refresh: matching hashes after prior apply",
			plan:            planWithSource(hashA),
			state:           stateWithSource(hashA),
			priorArtifactID: "artifact-1",
			newArtifactID:   "artifact-1",
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := artifactSourceNeedsUpload(tt.plan, tt.state, tt.priorArtifactID, tt.newArtifactID); got != tt.want {
				t.Fatalf("artifactSourceNeedsUpload() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCatalogIDFromModel(t *testing.T) {
	t.Parallel()

	catalogID := "aaaaaaaaaaaaaaaaaaaaaaaa"
	specWithCatalog := func(id types.String) *ArtifactSpecModel {
		return &ArtifactSpecModel{
			ContainerGroups: []ArtifactContainerGroupModel{{
				Containers: []ArtifactContainerModel{
					{ImageURI: types.StringValue("sidecar:latest")},
					{
						Primary: types.BoolValue(true),
						ImageBuildConfig: &ArtifactImageBuildConfigModel{
							CodeRef: artifactCodeRefObject(&ArtifactCodeRefModel{
								CatalogID:        id,
								CatalogVersionID: types.StringValue("bbbbbbbbbbbbbbbbbbbbbbbb"),
							}),
						},
					},
				},
			}},
		}
	}

	tests := []struct {
		name string
		data *ArtifactResourceModel
		want string
	}{
		{name: "nil model", data: nil, want: ""},
		{name: "nil spec", data: &ArtifactResourceModel{}, want: ""},
		{name: "no code_ref", data: &ArtifactResourceModel{Spec: &ArtifactSpecModel{}}, want: ""},
		{
			name: "null catalog id skipped",
			data: &ArtifactResourceModel{Spec: specWithCatalog(types.StringNull())},
			want: "",
		},
		{
			name: "known catalog id",
			data: &ArtifactResourceModel{Spec: specWithCatalog(types.StringValue(catalogID))},
			want: catalogID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := catalogIDFromModel(tt.data); got != tt.want {
				t.Fatalf("catalogIDFromModel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCatalogVersionIDFromModel(t *testing.T) {
	t.Parallel()

	versionID := "bbbbbbbbbbbbbbbbbbbbbbbb"
	specWithVersion := func(id types.String) *ArtifactSpecModel {
		return &ArtifactSpecModel{
			ContainerGroups: []ArtifactContainerGroupModel{{
				Containers: []ArtifactContainerModel{{
					ImageBuildConfig: &ArtifactImageBuildConfigModel{
						CodeRef: artifactCodeRefObject(&ArtifactCodeRefModel{
							CatalogID:        types.StringValue("aaaaaaaaaaaaaaaaaaaaaaaa"),
							CatalogVersionID: id,
						}),
					},
				}},
			}},
		}
	}

	tests := []struct {
		name string
		data *ArtifactResourceModel
		want string
	}{
		{name: "nil model", data: nil, want: ""},
		{name: "nil spec", data: &ArtifactResourceModel{}, want: ""},
		{
			name: "known catalog version id",
			data: &ArtifactResourceModel{Spec: specWithVersion(types.StringValue(versionID))},
			want: versionID,
		},
		{
			name: "unknown version id skipped",
			data: &ArtifactResourceModel{Spec: specWithVersion(types.StringUnknown())},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := catalogVersionIDFromModel(tt.data); got != tt.want {
				t.Fatalf("catalogVersionIDFromModel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRefreshArtifactSourceDirHash(t *testing.T) {
	t.Parallel()

	t.Run("no source block leaves model unchanged", func(t *testing.T) {
		data := &ArtifactResourceModel{}
		refreshArtifactSourceDirHash(data)
		if data.Source != nil {
			t.Fatal("expected source to remain nil")
		}
	})

	t.Run("computes hash for valid directory", func(t *testing.T) {
		dir := writeArtifactSourceTree(t, map[string]string{"main.py": "print('hi')"})
		data := &ArtifactResourceModel{
			Source: &ArtifactSourceModel{Dir: types.StringValue(dir)},
		}

		refreshArtifactSourceDirHash(data)
		if !IsKnown(data.Source.DirHash) {
			t.Fatal("expected computed dir_hash")
		}

		refreshArtifactSourceDirHash(data)
		if !data.Source.DirHash.Equal(data.Source.DirHash) {
			t.Fatal("expected stable hash on unchanged tree")
		}
	})

	t.Run("hash changes when file content changes", func(t *testing.T) {
		dir := writeArtifactSourceTree(t, map[string]string{"main.py": "v1"})
		data := &ArtifactResourceModel{
			Source: &ArtifactSourceModel{Dir: types.StringValue(dir)},
		}
		refreshArtifactSourceDirHash(data)
		first := data.Source.DirHash

		if err := os.WriteFile(filepath.Join(dir, "main.py"), []byte("v2"), 0o644); err != nil {
			t.Fatal(err)
		}
		refreshArtifactSourceDirHash(data)
		if data.Source.DirHash.Equal(first) {
			t.Fatal("expected dir_hash to change after file edit")
		}
	})

	t.Run("missing directory leaves dir_hash unset", func(t *testing.T) {
		data := &ArtifactResourceModel{
			Source: &ArtifactSourceModel{
				Dir:     types.StringValue(filepath.Join(t.TempDir(), "missing")),
				DirHash: types.StringNull(),
			},
		}
		refreshArtifactSourceDirHash(data)
		if IsKnown(data.Source.DirHash) {
			t.Fatal("expected dir_hash to remain unset when directory is missing")
		}
	})

	t.Run("ignores venv and datarobot yaml", func(t *testing.T) {
		dir := writeArtifactSourceTree(t, map[string]string{"main.py": "print('hi')"})
		data := &ArtifactResourceModel{
			Source: &ArtifactSourceModel{Dir: types.StringValue(dir)},
		}
		refreshArtifactSourceDirHash(data)
		base := data.Source.DirHash

		if err := os.Mkdir(filepath.Join(dir, ".venv"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".venv", "lib.py"), []byte("ignored"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".datarobot.yaml"), []byte("spec: x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		refreshArtifactSourceDirHash(data)
		if !data.Source.DirHash.Equal(base) {
			t.Fatal("expected dir_hash to ignore .venv and .datarobot.yaml")
		}
	})
}

func TestComputeArtifactSourceDirHash_PlanMatchesAfterWritingDrignore(t *testing.T) {
	t.Parallel()

	dir := writeArtifactSourceTree(t, map[string]string{
		"main.py":         "print('hi')",
		".venv/lib.py":    "ignored",
		".datarobot.yaml": "spec: x\n",
	})
	data := &ArtifactResourceModel{
		Source: &ArtifactSourceModel{Dir: types.StringValue(dir)},
	}

	before, err := computeArtifactSourceDirHash(data)
	if err != nil {
		t.Fatalf("plan-time hash: %v", err)
	}
	if !IsKnown(before) {
		t.Fatal("expected plan-time dir_hash")
	}

	wrote, err := ignore.WriteDefaultDrignoreIfMissing(dir)
	if err != nil {
		t.Fatalf("WriteDefaultDrignoreIfMissing: %v", err)
	}
	if !wrote {
		t.Fatal("expected default .drignore to be written")
	}

	after, err := computeArtifactSourceDirHash(data)
	if err != nil {
		t.Fatalf("after-write hash: %v", err)
	}
	if !before.Equal(after) {
		t.Fatalf("dir_hash churned: plan %q vs after write %q", before.ValueString(), after.ValueString())
	}
}

func TestSyncArtifactSource(t *testing.T) {
	t.Parallel()

	const (
		artifactID = "artifact-1"
		catalogID  = "aaaaaaaaaaaaaaaaaaaaaaaa"
		versionID  = "bbbbbbbbbbbbbbbbbbbbbbbb"
	)

	t.Run("skips upload when dir_hash unchanged", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		dir := writeArtifactSourceTree(t, map[string]string{"main.py": "stable"})
		hash, err := computeFolderHash(types.StringValue(dir))
		if err != nil {
			t.Fatal(err)
		}

		plan := &ArtifactResourceModel{
			Source: &ArtifactSourceModel{Dir: types.StringValue(dir), DirHash: hash},
		}
		state := &ArtifactResourceModel{
			Source: &ArtifactSourceModel{Dir: types.StringValue(dir), DirHash: hash},
		}
		artifact := &client.Artifact{ID: artifactID}

		got, uploaded, err := resource.syncArtifactSource(context.Background(), plan, state, artifact, artifactID)
		if err != nil {
			t.Fatalf("syncArtifactSource() error = %v", err)
		}
		if uploaded {
			t.Fatal("expected upload to be skipped when dir_hash unchanged")
		}
		if got != artifact {
			t.Fatal("expected same artifact pointer when upload is skipped")
		}
	})

	t.Run("uploads on create with nil state", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		filesAPI := newSyncTestFilesAPI()

		mockService.EXPECT().FilesAPI().Return(filesAPI)
		mockService.EXPECT().
			PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Eq("cat-new"), gomock.Any()).
			Return(&client.Artifact{ID: artifactID}, nil)

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		dir := writeArtifactSourceTree(t, map[string]string{"main.py": "create"})
		plan := &ArtifactResourceModel{
			Source: &ArtifactSourceModel{Dir: types.StringValue(dir)},
		}

		_, uploaded, err := resource.syncArtifactSource(context.Background(), plan, nil, &client.Artifact{ID: artifactID}, "")
		if err != nil {
			t.Fatalf("syncArtifactSource() error = %v", err)
		}
		if !uploaded {
			t.Fatal("expected upload during create")
		}
		if filesAPI.createCatalogCalls == 0 && filesAPI.uploadFromZipNewCalls == 0 {
			t.Fatal("expected Files API upload during create")
		}
	})

	t.Run("writes drignore and skips venv and datarobot yaml", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		filesAPI := newSyncTestFilesAPI()

		mockService.EXPECT().FilesAPI().Return(filesAPI)
		mockService.EXPECT().
			PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).
			Return(&client.Artifact{ID: artifactID}, nil)

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		dir := writeArtifactSourceTree(t, map[string]string{
			"main.py":         "print('hi')",
			".venv/lib.py":    "ignored",
			".datarobot.yaml": "spec: x\n",
		})
		plan := &ArtifactResourceModel{
			Source: &ArtifactSourceModel{Dir: types.StringValue(dir)},
		}

		if _, _, err := resource.syncArtifactSource(context.Background(), plan, nil, &client.Artifact{ID: artifactID}, ""); err != nil {
			t.Fatalf("syncArtifactSource() error = %v", err)
		}
		if _, err := os.Stat(filepath.Join(dir, ".drignore")); err != nil {
			t.Fatalf("expected .drignore to be written: %v", err)
		}
		for _, p := range filesAPI.uploadToStagePaths {
			if p == ".datarobot.yaml" || p == ".venv/lib.py" {
				t.Fatalf("uploaded ignored path %q", p)
			}
		}
		foundMain := false
		foundIgnore := false
		for _, p := range filesAPI.uploadToStagePaths {
			if p == "main.py" {
				foundMain = true
			}
			if p == ".drignore" {
				foundIgnore = true
			}
		}
		if !foundMain || !foundIgnore {
			t.Fatalf("uploaded paths = %v, want main.py and .drignore", filesAPI.uploadToStagePaths)
		}
	})

	t.Run("generate_ignore false does not write drignore", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		filesAPI := newSyncTestFilesAPI()

		mockService.EXPECT().FilesAPI().Return(filesAPI)
		mockService.EXPECT().
			PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).
			Return(&client.Artifact{ID: artifactID}, nil)

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		dir := writeArtifactSourceTree(t, map[string]string{"main.py": "print('hi')"})
		plan := &ArtifactResourceModel{
			Source: &ArtifactSourceModel{
				Dir:            types.StringValue(dir),
				GenerateIgnore: types.BoolValue(false),
			},
		}

		if _, _, err := resource.syncArtifactSource(context.Background(), plan, nil, &client.Artifact{ID: artifactID}, ""); err != nil {
			t.Fatalf("syncArtifactSource() error = %v", err)
		}
		if _, err := os.Stat(filepath.Join(dir, ".drignore")); !os.IsNotExist(err) {
			t.Fatal("expected .drignore not to be written when generate_ignore is false")
		}
	})

	t.Run("reuses catalog id from state", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		filesAPI := newSyncTestFilesAPI()
		filesAPI.catalogID = catalogID

		mockService.EXPECT().FilesAPI().Return(filesAPI)
		mockService.EXPECT().
			PatchArtifactCodeRef(gomock.Any(), artifactID, catalogID, gomock.Any()).
			Return(&client.Artifact{ID: artifactID}, nil)

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		dir := writeArtifactSourceTree(t, map[string]string{"main.py": "v2"})
		plan := &ArtifactResourceModel{
			Source: &ArtifactSourceModel{
				Dir:     types.StringValue(dir),
				DirHash: types.StringValue("new-hash"),
			},
			Spec: artifactSpecWithCodeRef(catalogID, versionID),
		}
		state := &ArtifactResourceModel{
			Source: &ArtifactSourceModel{
				Dir:     types.StringValue(dir),
				DirHash: types.StringValue("old-hash"),
			},
			Spec: artifactSpecWithCodeRef(catalogID, versionID),
		}

		if _, _, err := resource.syncArtifactSource(context.Background(), plan, state, &client.Artifact{ID: artifactID}, artifactID); err != nil {
			t.Fatalf("syncArtifactSource() error = %v", err)
		}
		if filesAPI.catalogID != catalogID {
			t.Fatalf("catalog id = %q, want %q", filesAPI.catalogID, catalogID)
		}
	})

	t.Run("falls back to catalog id from artifact when state lacks it", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		filesAPI := newSyncTestFilesAPI()
		filesAPI.catalogID = catalogID

		mockService.EXPECT().FilesAPI().Return(filesAPI)
		mockService.EXPECT().
			PatchArtifactCodeRef(gomock.Any(), artifactID, catalogID, gomock.Any()).
			Return(&client.Artifact{ID: artifactID}, nil)

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		dir := writeArtifactSourceTree(t, map[string]string{"main.py": "fallback"})
		plan := &ArtifactResourceModel{
			Source: &ArtifactSourceModel{
				Dir:     types.StringValue(dir),
				DirHash: types.StringValue("new-hash"),
			},
		}
		state := &ArtifactResourceModel{
			Source: &ArtifactSourceModel{
				Dir:     types.StringValue(dir),
				DirHash: types.StringValue("old-hash"),
			},
		}
		artifact := artifactWithCodeRef(artifactID, catalogID, versionID)

		if _, _, err := resource.syncArtifactSource(context.Background(), plan, state, artifact, artifactID); err != nil {
			t.Fatalf("syncArtifactSource() error = %v", err)
		}
	})

	t.Run("upload failure is returned without patching code ref", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		filesAPI := newSyncTestFilesAPI()
		filesAPI.uploadErr = fmt.Errorf("upload failed")

		mockService.EXPECT().FilesAPI().Return(filesAPI)

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		dir := writeArtifactSourceTree(t, map[string]string{"main.py": "fail"})
		plan := &ArtifactResourceModel{
			Source: &ArtifactSourceModel{
				Dir:     types.StringValue(dir),
				DirHash: types.StringValue("new-hash"),
			},
		}
		state := &ArtifactResourceModel{
			Source: &ArtifactSourceModel{
				Dir:     types.StringValue(dir),
				DirHash: types.StringValue("old-hash"),
			},
		}

		_, _, err := resource.syncArtifactSource(context.Background(), plan, state, &client.Artifact{ID: artifactID}, artifactID)
		if err == nil {
			t.Fatal("expected upload error")
		}
	})

	t.Run("patch failure is returned after upload", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		filesAPI := newSyncTestFilesAPI()

		mockService.EXPECT().FilesAPI().Return(filesAPI)
		mockService.EXPECT().
			PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).
			Return(nil, fmt.Errorf("patch failed"))

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		dir := writeArtifactSourceTree(t, map[string]string{"main.py": "patch-fail"})
		plan := &ArtifactResourceModel{
			Source: &ArtifactSourceModel{
				Dir:     types.StringValue(dir),
				DirHash: types.StringValue("new-hash"),
			},
		}
		state := &ArtifactResourceModel{
			Source: &ArtifactSourceModel{
				Dir:     types.StringValue(dir),
				DirHash: types.StringValue("old-hash"),
			},
		}

		_, _, err := resource.syncArtifactSource(context.Background(), plan, state, &client.Artifact{ID: artifactID}, artifactID)
		if err == nil {
			t.Fatal("expected patch error")
		}
	})
}

// TestSyncArtifactSourceThreeWay covers what wiring the three-way engine
// into the resource changed: local bookkeeping under source.dir/.wapi/, no
// network work for an unchanged tree, code_ref repointing across artifact
// versions, and remote changes landing in source.dir. The classification
// and diff rules themselves are tested in internal/artifactsource/sync.
func TestSyncArtifactSourceThreeWay(t *testing.T) {
	t.Parallel()

	const artifactID = "artifact-1"

	// syncOnce wires a resource around filesAPI and syncs dir once.
	syncOnce := func(
		t *testing.T,
		mockService *mock_client.MockService,
		dir string,
		artifact *client.Artifact,
		priorArtifactID string,
		state *ArtifactResourceModel,
	) (*client.Artifact, bool, error) {
		t.Helper()

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		plan := &ArtifactResourceModel{
			Source: &ArtifactSourceModel{
				Dir:     types.StringValue(dir),
				DirHash: types.StringValue("planned-hash"),
			},
		}

		return resource.syncArtifactSource(context.Background(), plan, state, artifact, priorArtifactID)
	}

	// syncedState is the state a previous apply would have left behind:
	// a dir_hash that no longer matches the plan, so the next sync runs.
	syncedState := func(dir string) *ArtifactResourceModel {
		return &ArtifactResourceModel{
			Source: &ArtifactSourceModel{
				Dir:     types.StringValue(dir),
				DirHash: types.StringValue("state-hash"),
			},
		}
	}

	t.Run("create uploads, patches code ref and records wapi state", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		filesAPI := newSyncTestFilesAPI()

		var patchedVersion string
		mockService.EXPECT().FilesAPI().Return(filesAPI)
		mockService.EXPECT().
			PatchArtifactCodeRef(gomock.Any(), artifactID, "cat-new", gomock.Any()).
			DoAndReturn(func(_ context.Context, id, catalogID, versionID string) (*client.Artifact, error) {
				patchedVersion = versionID
				return artifactWithCodeRef(id, catalogID, versionID), nil
			})

		dir := writeArtifactSourceTree(t, map[string]string{"main.py": "shared"})

		got, uploaded, err := syncOnce(t, mockService, dir, &client.Artifact{ID: artifactID}, "", nil)
		if err != nil {
			t.Fatalf("syncArtifactSource() error = %v", err)
		}
		if !uploaded {
			t.Fatal("expected the sync to report work on create")
		}
		if ref := client.ExtractCodeRef(got); ref == nil || ref.CatalogVersionID != patchedVersion {
			t.Fatalf("returned artifact does not carry the patched code_ref: %+v", ref)
		}

		cfg, err := wapi.LoadConfig(dir)
		if err != nil {
			t.Fatalf("load .wapi/config.json: %v", err)
		}
		if cfg.ArtifactID != artifactID {
			t.Fatalf("config artifactId = %q, want %q", cfg.ArtifactID, artifactID)
		}
		if cfg.CatalogID == nil || *cfg.CatalogID != "cat-new" {
			t.Fatalf("config catalogId = %v, want cat-new", cfg.CatalogID)
		}
		if cfg.LastSyncedVersionID == nil || *cfg.LastSyncedVersionID != patchedVersion {
			t.Fatalf("config lastSyncedVersionId = %v, want %q", cfg.LastSyncedVersionID, patchedVersion)
		}

		manifest, err := wapi.LoadManifest(dir)
		if err != nil {
			t.Fatalf("load .wapi/manifest.json: %v", err)
		}
		for _, want := range []string{"main.py", ignore.FileName} {
			if _, ok := manifest.Files[want]; !ok {
				t.Fatalf("BASE manifest is missing %q: %v", want, manifest.Files)
			}
		}

		// The rollback tree is committed, so the next apply cannot revert
		// this one. (.wapi/sync.lock stays on disk by design; the retry
		// case below covers that it is released.)
		if _, err := os.Stat(filepath.Join(dir, wapi.DirName, ".rollback")); err == nil {
			t.Fatal("expected .wapi/.rollback to be gone after a successful sync")
		}
	})

	t.Run("unchanged tree and matching catalog version does not upload", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		filesAPI := newSyncTestFilesAPI()

		mockService.EXPECT().FilesAPI().Return(filesAPI).Times(2)
		mockService.EXPECT().
			PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, id, catalogID, versionID string) (*client.Artifact, error) {
				return artifactWithCodeRef(id, catalogID, versionID), nil
			})

		dir := writeArtifactSourceTree(t, map[string]string{"main.py": "shared"})

		synced, _, err := syncOnce(t, mockService, dir, &client.Artifact{ID: artifactID}, "", nil)
		if err != nil {
			t.Fatalf("first syncArtifactSource() error = %v", err)
		}
		firstRunUploads := filesAPI.uploadCalls()

		// Same tree, same catalog version: the plan is empty, so the
		// second apply must not touch the catalog at all.
		if _, _, err := syncOnce(t, mockService, dir, synced, artifactID, syncedState(dir)); err != nil {
			t.Fatalf("second syncArtifactSource() error = %v", err)
		}
		if got := filesAPI.uploadCalls(); got != firstRunUploads {
			t.Fatalf("second sync issued %d upload call(s), want none", got-firstRunUploads)
		}
		if filesAPI.allFilesCalls != 0 {
			t.Fatalf("AllFiles called %d time(s) on an undrifted catalog, want 0", filesAPI.allFilesCalls)
		}
	})

	t.Run("new artifact version repoints code ref without re-uploading", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		filesAPI := newSyncTestFilesAPI()

		const clonedID = "artifact-2"

		var syncedVersion string
		mockService.EXPECT().FilesAPI().Return(filesAPI).Times(2)
		mockService.EXPECT().
			PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, id, catalogID, versionID string) (*client.Artifact, error) {
				syncedVersion = versionID
				return artifactWithCodeRef(id, catalogID, versionID), nil
			})

		dir := writeArtifactSourceTree(t, map[string]string{"main.py": "shared"})

		if _, _, err := syncOnce(t, mockService, dir, &client.Artifact{ID: artifactID}, "", nil); err != nil {
			t.Fatalf("first syncArtifactSource() error = %v", err)
		}
		firstRunUploads := filesAPI.uploadCalls()

		// A locked artifact whose source changes is cloned to a fresh
		// draft: new artifact ID, no code_ref of its own, same directory
		// and same catalog. The clone has to be pointed at the code the
		// directory already matches, without pushing it again.
		mockService.EXPECT().
			PatchArtifactCodeRef(gomock.Any(), clonedID, "cat-new", gomock.Any()).
			DoAndReturn(func(_ context.Context, id, catalogID, versionID string) (*client.Artifact, error) {
				if versionID != syncedVersion {
					t.Errorf("clone patched to version %q, want the last synced %q", versionID, syncedVersion)
				}
				return artifactWithCodeRef(id, catalogID, versionID), nil
			})

		if _, _, err := syncOnce(t, mockService, dir, &client.Artifact{ID: clonedID}, artifactID, syncedState(dir)); err != nil {
			t.Fatalf("clone syncArtifactSource() error = %v", err)
		}
		if got := filesAPI.uploadCalls(); got != firstRunUploads {
			t.Fatalf("clone issued %d upload call(s), want none", got-firstRunUploads)
		}

		cfg, err := wapi.LoadConfig(dir)
		if err != nil {
			t.Fatalf("load .wapi/config.json: %v", err)
		}
		if cfg.ArtifactID != clonedID {
			t.Fatalf("config artifactId = %q, want the cloned %q", cfg.ArtifactID, clonedID)
		}
	})

	t.Run("remote additions are downloaded into source dir", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		filesAPI := newSyncTestFilesAPI()

		mockService.EXPECT().FilesAPI().Return(filesAPI).Times(2)
		mockService.EXPECT().
			PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, id, catalogID, versionID string) (*client.Artifact, error) {
				return artifactWithCodeRef(id, catalogID, versionID), nil
			})

		dir := writeArtifactSourceTree(t, map[string]string{"main.py": "shared"})

		if _, _, err := syncOnce(t, mockService, dir, &client.Artifact{ID: artifactID}, "", nil); err != nil {
			t.Fatalf("first syncArtifactSource() error = %v", err)
		}
		firstRunUploads := filesAPI.uploadCalls()

		// Someone else pushed to the catalog: the artifact now points at
		// a version .wapi/ has never seen, so the engine fetches the real
		// remote manifest instead of trusting BASE.
		mirrorRemoteTree(t, filesAPI, dir, "main.py", ignore.FileName)
		filesAPI.remoteFile("helper.py", "from-remote")

		drifted := artifactWithCodeRef(artifactID, "cat-new", "ver-remote")
		if _, _, err := syncOnce(t, mockService, dir, drifted, artifactID, syncedState(dir)); err != nil {
			t.Fatalf("drifted syncArtifactSource() error = %v", err)
		}

		got, err := os.ReadFile(filepath.Join(dir, "helper.py"))
		if err != nil {
			t.Fatalf("expected the remote-only file to be downloaded: %v", err)
		}
		if string(got) != "from-remote" {
			t.Fatalf("helper.py = %q, want %q", got, "from-remote")
		}
		if filesAPI.allFilesCalls != 1 {
			t.Fatalf("AllFiles called %d time(s) on a drifted catalog, want 1", filesAPI.allFilesCalls)
		}
		if got := filesAPI.uploadCalls(); got != firstRunUploads {
			t.Fatalf("pull-only sync issued %d upload call(s), want none", got-firstRunUploads)
		}

		manifest, err := wapi.LoadManifest(dir)
		if err != nil {
			t.Fatalf("load .wapi/manifest.json: %v", err)
		}
		if _, ok := manifest.Files["helper.py"]; !ok {
			t.Fatalf("BASE manifest is missing the downloaded file: %v", manifest.Files)
		}
	})

	t.Run("locked artifact is refused instead of synced in place", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		filesAPI := newSyncTestFilesAPI()

		mockService.EXPECT().FilesAPI().Return(filesAPI)

		dir := writeArtifactSourceTree(t, map[string]string{"main.py": "shared"})
		locked := &client.Artifact{ID: artifactID, Status: client.ArtifactStatusLocked}

		_, _, err := syncOnce(t, mockService, dir, locked, artifactID, syncedState(dir))
		if err == nil {
			t.Fatal("expected a locked artifact to be refused")
		}
		if filesAPI.uploadCalls() != 0 {
			t.Fatalf("uploaded %d file(s) onto a locked artifact", filesAPI.uploadCalls())
		}
	})

	t.Run("failed sync releases the lock so the next apply retries", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		filesAPI := newSyncTestFilesAPI()
		filesAPI.uploadErr = errors.New("files API unavailable")

		mockService.EXPECT().FilesAPI().Return(filesAPI).Times(2)
		mockService.EXPECT().
			PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, id, catalogID, versionID string) (*client.Artifact, error) {
				return artifactWithCodeRef(id, catalogID, versionID), nil
			})

		dir := writeArtifactSourceTree(t, map[string]string{"main.py": "shared"})

		if _, _, err := syncOnce(t, mockService, dir, &client.Artifact{ID: artifactID}, "", nil); err == nil {
			t.Fatal("expected the upload failure to surface")
		}
		if _, err := os.Stat(filepath.Join(dir, wapi.DirName, ".rollback")); err == nil {
			t.Fatal("expected the rollback tree to be unwound after a failed sync")
		}

		filesAPI.uploadErr = nil
		if _, _, err := syncOnce(t, mockService, dir, &client.Artifact{ID: artifactID}, "", nil); err != nil {
			t.Fatalf("retry after a failed sync error = %v", err)
		}
	})
}

func TestSyncArtifactSourceAndBuild(t *testing.T) {
	t.Parallel()

	const artifactID = "artifact-1"

	t.Run("uploads source and waits for build on create", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		filesAPI := newSyncTestFilesAPI()

		draftArtifact := artifactFixtureDraftWithBuildConfig(artifactID, nil, "app")
		patchedArtifact := artifactSourcePatchedArtifact(draftArtifact, artifactSourceTestCatalogID, artifactSourceTestVersionID)

		mockService.EXPECT().FilesAPI().Return(filesAPI)
		mockService.EXPECT().
			PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).
			Return(patchedArtifact, nil)
		expectArtifactBuildAfterUpload(mockService, artifactID, artifactFixtureWithImageURI(patchedArtifact))

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		dir := writeArtifactSourceTree(t, map[string]string{"main.py": "create"})
		plan := testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithBuildConfig()))

		got, err := resource.syncArtifactSourceAndBuild(
			context.Background(),
			plan,
			nil,
			draftArtifact,
			"",
		)
		if err != nil {
			t.Fatalf("syncArtifactSourceAndBuild() error = %v", err)
		}
		if artifactPrimaryContainerImageURI(got) != artifactSourceTestImageURI {
			t.Fatalf("image_uri = %q, want %q", artifactPrimaryContainerImageURI(got), artifactSourceTestImageURI)
		}
	})

	t.Run("skips build when source upload is skipped", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		dir := writeArtifactSourceTree(t, map[string]string{"main.py": "stable"})
		hash, err := computeFolderHash(types.StringValue(dir))
		if err != nil {
			t.Fatal(err)
		}

		spec := testDraftSourceSpec(testPrimaryWithBuildConfig())
		plan := testSourcePlanModel(t, dir, spec, func(m *ArtifactResourceModel) {
			m.Source.DirHash = hash
		})
		state := testSourcePlanModel(t, dir, spec, func(m *ArtifactResourceModel) {
			m.Source.DirHash = hash
			m.ArtifactID = types.StringValue(artifactID)
		})
		artifact := artifactFixtureDraftWithBuildConfig(artifactID, nil, "app")

		got, err := resource.syncArtifactSourceAndBuild(
			context.Background(),
			plan,
			state,
			artifact,
			artifactID,
		)
		if err != nil {
			t.Fatalf("syncArtifactSourceAndBuild() error = %v", err)
		}
		if got != artifact {
			t.Fatal("expected same artifact when upload and build are skipped")
		}
	})

	t.Run("skips build when plan has no image build config", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		filesAPI := newSyncTestFilesAPI()

		mockService.EXPECT().FilesAPI().Return(filesAPI)
		mockService.EXPECT().
			PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).
			Return(&client.Artifact{ID: artifactID, Status: client.ArtifactStatusDraft}, nil)

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		dir := writeArtifactSourceTree(t, map[string]string{"main.py": "no-build"})
		plan := testSourcePlanModel(t, dir, testDraftSourceSpec(ArtifactContainerModel{
			Primary:  types.BoolValue(true),
			Port:     types.Int64Value(8080),
			ImageURI: types.StringValue("nginx:latest"),
		}))

		got, err := resource.syncArtifactSourceAndBuild(
			context.Background(),
			plan,
			nil,
			&client.Artifact{ID: artifactID, Status: client.ArtifactStatusDraft},
			"",
		)
		if err != nil {
			t.Fatalf("syncArtifactSourceAndBuild() error = %v", err)
		}
		if got.ID != artifactID {
			t.Fatalf("artifact_id = %q, want %q", got.ID, artifactID)
		}
	})

	t.Run("wait_for_build false triggers build without waiting", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		filesAPI := newSyncTestFilesAPI()

		draftArtifact := artifactFixtureDraftWithBuildConfig(artifactID, nil, "app")
		patchedArtifact := artifactSourcePatchedArtifact(draftArtifact, artifactSourceTestCatalogID, artifactSourceTestVersionID)
		builtArtifact := artifactFixtureWithImageURI(patchedArtifact)

		mockService.EXPECT().FilesAPI().Return(filesAPI)
		mockService.EXPECT().
			PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).
			Return(patchedArtifact, nil)
		expectArtifactBuildTriggerOnly(mockService, artifactID, builtArtifact)

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		dir := writeArtifactSourceTree(t, map[string]string{"main.py": "trigger-only"})
		plan := testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithBuildConfig()), func(m *ArtifactResourceModel) {
			m.Source.WaitForBuild = types.BoolValue(false)
		})

		got, err := resource.syncArtifactSourceAndBuild(
			context.Background(),
			plan,
			nil,
			draftArtifact,
			"",
		)
		if err != nil {
			t.Fatalf("syncArtifactSourceAndBuild() error = %v", err)
		}
		if artifactPrimaryContainerImageURI(got) != artifactSourceTestImageURI {
			t.Fatalf("image_uri = %q, want %q", artifactPrimaryContainerImageURI(got), artifactSourceTestImageURI)
		}
	})

	t.Run("upload failure is returned without build sync error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		filesAPI := newSyncTestFilesAPI()
		filesAPI.uploadErr = fmt.Errorf("upload failed")

		mockService.EXPECT().FilesAPI().Return(filesAPI)

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		dir := writeArtifactSourceTree(t, map[string]string{"main.py": "fail"})
		plan := testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithBuildConfig()), func(m *ArtifactResourceModel) {
			m.Source.DirHash = types.StringValue("new-hash")
		})
		state := testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithBuildConfig()), func(m *ArtifactResourceModel) {
			m.Source.DirHash = types.StringValue("old-hash")
			m.ArtifactID = types.StringValue(artifactID)
		})

		_, err := resource.syncArtifactSourceAndBuild(
			context.Background(),
			plan,
			state,
			artifactFixtureDraftWithBuildConfig(artifactID, nil, "app"),
			artifactID,
		)
		if err == nil {
			t.Fatal("expected upload error")
		}
		var buildErr *artifactBuildSyncError
		if errors.As(err, &buildErr) {
			t.Fatal("expected upload error, not artifactBuildSyncError")
		}
	})

	t.Run("build failure wraps artifactBuildSyncError", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		filesAPI := newSyncTestFilesAPI()

		draftArtifact := artifactFixtureDraftWithBuildConfig(artifactID, nil, "app")
		patchedArtifact := artifactSourcePatchedArtifact(draftArtifact, artifactSourceTestCatalogID, artifactSourceTestVersionID)
		buildErr := &client.ArtifactBuildFailedError{
			BuildID: artifactSourceTestBuildID,
			Status:  client.ArtifactBuildStatusFailed,
		}

		mockService.EXPECT().FilesAPI().Return(filesAPI)
		mockService.EXPECT().
			PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).
			Return(patchedArtifact, nil)
		gomock.InOrder(
			mockService.EXPECT().
				TriggerArtifactBuild(gomock.Any(), artifactID).
				Return(&client.ArtifactBuildTriggerResponse{BuildIDs: []string{artifactSourceTestBuildID}}, nil),
			mockService.EXPECT().
				WaitForArtifactBuild(gomock.Any(), artifactID, artifactSourceTestBuildID, gomock.Any()).
				Return(&client.ArtifactBuild{ID: artifactSourceTestBuildID, Status: client.ArtifactBuildStatusFailed}, buildErr),
			mockService.EXPECT().
				BaseURL().
				Return("https://app.datarobot.com"),
			mockService.EXPECT().
				GetArtifactBuildLogs(gomock.Any(), artifactID, artifactSourceTestBuildID).
				Return("", nil),
		)

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		dir := writeArtifactSourceTree(t, map[string]string{"main.py": "build-fail"})
		plan := testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithBuildConfig()))

		_, err := resource.syncArtifactSourceAndBuild(
			context.Background(),
			plan,
			nil,
			draftArtifact,
			"",
		)
		if err == nil {
			t.Fatal("expected build error")
		}
		var syncBuildErr *artifactBuildSyncError
		if !errors.As(err, &syncBuildErr) {
			t.Fatalf("expected artifactBuildSyncError, got %T: %v", err, err)
		}
		var failedErr *client.ArtifactBuildFailedError
		if !errors.As(syncBuildErr, &failedErr) {
			t.Fatalf("expected wrapped ArtifactBuildFailedError, got %T: %v", syncBuildErr.Unwrap(), syncBuildErr.Unwrap())
		}
		if failedErr.BuildID != artifactSourceTestBuildID {
			t.Fatalf("build_id = %q, want %q", failedErr.BuildID, artifactSourceTestBuildID)
		}
	})
}

func TestRollbackArtifactCreate(t *testing.T) {
	t.Parallel()

	repoID := "repo-123"

	t.Run("nil artifact is a no-op", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		resource := &ArtifactResource{provider: &Provider{service: mockService}}

		resource.rollbackArtifactCreate(context.Background(), nil)
	})

	t.Run("missing repository id is a no-op", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		resource := &ArtifactResource{provider: &Provider{service: mockService}}

		resource.rollbackArtifactCreate(context.Background(), &client.Artifact{ID: "artifact-1"})
	})

	t.Run("deletes artifact repository", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		mockService.EXPECT().DeleteArtifactRepository(gomock.Any(), repoID).Return(nil)

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		resource.rollbackArtifactCreate(context.Background(), &client.Artifact{
			ID:                   "artifact-1",
			ArtifactRepositoryID: &repoID,
		})
	})
}

func writeArtifactSourceTree(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// mirrorRemoteTree registers the on-disk content of paths as catalog
// content, so a drifted AllFiles call reports a remote that agrees with
// source.dir on everything except what the test adds on top.
func mirrorRemoteTree(t *testing.T, filesAPI *syncTestFilesAPI, dir string, paths ...string) {
	t.Helper()

	for _, rel := range paths {
		content, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			t.Fatal(err)
		}
		filesAPI.remoteFile(rel, string(content))
	}
}

func artifactSpecWithCodeRef(catalogID, versionID string) *ArtifactSpecModel {
	return &ArtifactSpecModel{
		ContainerGroups: []ArtifactContainerGroupModel{{
			Containers: []ArtifactContainerModel{{
				Primary: types.BoolValue(true),
				ImageBuildConfig: imageBuildConfigWithCodeRef(&ArtifactCodeRefModel{
					CatalogID:        types.StringValue(catalogID),
					CatalogVersionID: types.StringValue(versionID),
				}),
			}},
		}},
	}
}

func artifactWithCodeRef(artifactID, catalogID, versionID string) *client.Artifact {
	primary := true
	return &client.Artifact{
		ID: artifactID,
		Spec: client.ArtifactSpec{
			ContainerGroups: []client.ArtifactContainerGroup{{
				Containers: []client.ArtifactContainer{{
					Primary: &primary,
					ImageBuildConfig: &client.ArtifactImageBuildConfig{
						CodeRef: &client.ArtifactCodeRef{
							DataRobot: client.ArtifactDataRobotCodeRef{
								CatalogID:        catalogID,
								CatalogVersionID: versionID,
							},
						},
					},
				}},
			}},
		},
	}
}

type syncTestFilesAPI struct {
	catalogID string
	version   int
	uploadErr error

	// allFiles is the catalog manifest AllFiles serves once the engine
	// sees the catalog version drift; blobs is the content DownloadFile
	// streams for the same paths. Both are populated by remoteFile.
	allFiles map[string]filesapi.FileMeta
	blobs    map[string]string

	createCatalogCalls    int
	uploadFromZipNewCalls int
	uploadToStagePaths    []string
	allFilesCalls         int
	downloadedPaths       []string
	deletedPaths          []string
}

func newSyncTestFilesAPI() *syncTestFilesAPI {
	return &syncTestFilesAPI{}
}

// remoteFile makes the catalog hold content at path: AllFiles advertises
// its hash and size, and DownloadFile serves the bytes.
func (m *syncTestFilesAPI) remoteFile(path, content string) {
	if m.allFiles == nil {
		m.allFiles = map[string]filesapi.FileMeta{}
		m.blobs = map[string]string{}
	}

	sum := sha256.Sum256([]byte(content))
	m.allFiles[path] = filesapi.FileMeta{
		Hash: hex.EncodeToString(sum[:]),
		Size: int64(len(content)),
	}
	m.blobs[path] = content
}

// uploadCalls counts every code-bearing request the engine made, so a test
// can assert that an unchanged tree touched the Files API not at all.
func (m *syncTestFilesAPI) uploadCalls() int {
	return len(m.uploadToStagePaths) + m.uploadFromZipNewCalls
}

func (m *syncTestFilesAPI) CreateCatalog(context.Context) (*filesapi.CatalogResp, error) {
	if m.uploadErr != nil {
		return nil, m.uploadErr
	}
	m.createCatalogCalls++
	if m.catalogID == "" {
		m.catalogID = "cat-new"
	}
	m.version++
	return &filesapi.CatalogResp{
		CatalogID:        m.catalogID,
		CatalogVersionID: syncTestVersionID(m.version),
	}, nil
}

func (m *syncTestFilesAPI) CreateStage(context.Context, string) (*filesapi.StageResp, error) {
	if m.uploadErr != nil {
		return nil, m.uploadErr
	}
	return &filesapi.StageResp{CatalogID: m.catalogID, StageID: "stage-1"}, nil
}

func (m *syncTestFilesAPI) UploadToStage(_ context.Context, _, _, name string, _ int64, _ io.Reader) error {
	if m.uploadErr != nil {
		return m.uploadErr
	}
	m.uploadToStagePaths = append(m.uploadToStagePaths, name)
	return nil
}

func (m *syncTestFilesAPI) ApplyStage(context.Context, string, string, string) (*filesapi.ApplyStageResp, error) {
	if m.uploadErr != nil {
		return nil, m.uploadErr
	}
	m.version++
	return &filesapi.ApplyStageResp{
		CatalogID:        m.catalogID,
		CatalogVersionID: syncTestVersionID(m.version),
	}, nil
}

func (m *syncTestFilesAPI) UploadFromZipNew(context.Context, string, int64, io.Reader) (*filesapi.FromFileResp, error) {
	if m.uploadErr != nil {
		return nil, m.uploadErr
	}
	m.uploadFromZipNewCalls++
	if m.catalogID == "" {
		m.catalogID = "cat-zip-new"
	}
	m.version++
	return &filesapi.FromFileResp{
		CatalogID:        m.catalogID,
		CatalogVersionID: syncTestVersionID(m.version),
	}, nil
}

func (m *syncTestFilesAPI) UploadFromZipExisting(context.Context, string, string, string, int64, io.Reader) (*filesapi.FromFileResp, error) {
	if m.uploadErr != nil {
		return nil, m.uploadErr
	}
	m.version++
	return &filesapi.FromFileResp{
		CatalogID:        m.catalogID,
		CatalogVersionID: syncTestVersionID(m.version),
	}, nil
}

func (m *syncTestFilesAPI) PollStatus(context.Context, string) (*filesapi.StatusResp, error) {
	return &filesapi.StatusResp{Status: filesapi.StatusCompleted}, nil
}

func (m *syncTestFilesAPI) AllFiles(context.Context, string, string) (map[string]filesapi.FileMeta, error) {
	m.allFilesCalls++
	return m.allFiles, nil
}

func (m *syncTestFilesAPI) DownloadFile(_ context.Context, _, _, path string, w io.Writer) (string, int64, error) {
	content, ok := m.blobs[path]
	if !ok {
		return "", 0, fmt.Errorf("syncTestFilesAPI: no remote content registered for %q", path)
	}

	m.downloadedPaths = append(m.downloadedPaths, path)
	n, err := io.WriteString(w, content)

	return "", int64(n), err
}

func (m *syncTestFilesAPI) DeleteFiles(_ context.Context, _ string, paths []string) (*filesapi.DeleteFilesResp, error) {
	m.deletedPaths = append(m.deletedPaths, paths...)
	m.version++

	return &filesapi.DeleteFilesResp{
		CatalogID:        m.catalogID,
		CatalogVersionID: syncTestVersionID(m.version),
	}, nil
}

func (m *syncTestFilesAPI) ListVersions(context.Context, string, int) ([]filesapi.CatalogVersion, error) {
	return nil, nil
}

func syncTestVersionID(n int) string {
	return fmt.Sprintf("ver-%d", n)
}

func testSourcePlanModel(t *testing.T, dir string, spec *ArtifactSpecModel, opts ...func(*ArtifactResourceModel)) *ArtifactResourceModel {
	t.Helper()

	model := &ArtifactResourceModel{
		Status: types.StringValue("draft"),
		Source: &ArtifactSourceModel{Dir: types.StringValue(dir)},
		Spec:   spec,
	}
	for _, opt := range opts {
		opt(model)
	}
	return model
}

func testDraftImageBuildContainer(primary types.Bool, name string) ArtifactContainerModel {
	container := ArtifactContainerModel{
		Primary: primary,
		Port:    types.Int64Value(8080),
		ImageBuildConfig: &ArtifactImageBuildConfigModel{
			Dockerfile: &ArtifactDockerfileModel{Source: types.StringValue("provided")},
		},
	}
	if name != "" {
		container.Name = types.StringValue(name)
	}
	return container
}

func testPrimaryWithBuildConfig() ArtifactContainerModel {
	return testDraftImageBuildContainer(types.BoolValue(true), "primary")
}

func testSidecarWithBuildConfig() ArtifactContainerModel {
	return testDraftImageBuildContainer(types.BoolValue(false), "sidecar")
}

func testSoleWithBuildConfig() ArtifactContainerModel {
	return testDraftImageBuildContainer(types.BoolNull(), "")
}

func testPrimaryWithCodeRef(codeRef *ArtifactCodeRefModel) ArtifactContainerModel {
	container := testPrimaryWithBuildConfig()
	_ = setImageBuildConfigCodeRef(container.ImageBuildConfig, codeRef)
	return container
}

func testDraftSourceSpec(containers ...ArtifactContainerModel) *ArtifactSpecModel {
	return &ArtifactSpecModel{
		ContainerGroups: []ArtifactContainerGroupModel{{
			Containers: containers,
		}},
	}
}

func TestCodeRefManuallySet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ref  *ArtifactCodeRefModel
		want bool
	}{
		{name: "nil", ref: nil, want: false},
		{name: "empty block", ref: &ArtifactCodeRefModel{}, want: false},
		{
			name: "catalog id only",
			ref:  &ArtifactCodeRefModel{CatalogID: types.StringValue("aaaaaaaaaaaaaaaaaaaaaaaa")},
			want: true,
		},
		{
			name: "catalog version id only",
			ref:  &ArtifactCodeRefModel{CatalogVersionID: types.StringValue("bbbbbbbbbbbbbbbbbbbbbbbb")},
			want: true,
		},
		{
			name: "both ids",
			ref: &ArtifactCodeRefModel{
				CatalogID:        types.StringValue("aaaaaaaaaaaaaaaaaaaaaaaa"),
				CatalogVersionID: types.StringValue("bbbbbbbbbbbbbbbbbbbbbbbb"),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := codeRefManuallySet(tt.ref); got != tt.want {
				t.Fatalf("codeRefManuallySet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestArtifactContainerIsPrimary(t *testing.T) {
	t.Parallel()

	soleContainer := ArtifactContainerGroupModel{
		Containers: []ArtifactContainerModel{{Primary: types.BoolNull()}},
	}
	explicitPrimary := ArtifactContainerGroupModel{
		Containers: []ArtifactContainerModel{
			{Primary: types.BoolValue(true)},
			{Primary: types.BoolValue(false)},
		},
	}

	tests := []struct {
		name      string
		container ArtifactContainerModel
		group     ArtifactContainerGroupModel
		want      bool
	}{
		{name: "sole container without primary flag", container: soleContainer.Containers[0], group: soleContainer, want: true},
		{name: "explicit primary", container: explicitPrimary.Containers[0], group: explicitPrimary, want: true},
		{name: "explicit non-primary", container: explicitPrimary.Containers[1], group: explicitPrimary, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := artifactContainerIsPrimary(tt.container, tt.group); got != tt.want {
				t.Fatalf("artifactContainerIsPrimary() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSourceManagedCodeRefNeedsUnknown(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	hashA := types.StringValue("hash-a")
	hashB := types.StringValue("hash-b")

	draftSpec := &ArtifactSpecModel{
		ContainerGroups: []ArtifactContainerGroupModel{{
			Containers: []ArtifactContainerModel{{
				Primary: types.BoolValue(true),
				Port:    types.Int64Value(8080),
				ImageBuildConfig: &ArtifactImageBuildConfigModel{
					Dockerfile: &ArtifactDockerfileModel{Source: types.StringValue("provided")},
				},
			}},
		}},
	}

	withSource := func(status, name string, dirHash, artifactID types.String, spec *ArtifactSpecModel) *ArtifactResourceModel {
		model := &ArtifactResourceModel{
			Status: types.StringValue(status),
			Name:   types.StringValue(name),
			Source: &ArtifactSourceModel{
				Dir:     types.StringValue(dir),
				DirHash: dirHash,
			},
			Spec: spec,
		}
		if !artifactID.IsNull() && !artifactID.IsUnknown() {
			model.ArtifactID = artifactID
		} else if artifactID.IsUnknown() {
			model.ArtifactID = types.StringUnknown()
		}
		return model
	}

	artifactOne := types.StringValue("artifact-1")
	artifactTwo := types.StringValue("artifact-2")

	tests := []struct {
		name     string
		plan     *ArtifactResourceModel
		state    *ArtifactResourceModel
		isCreate bool
		want     bool
	}{
		{
			name:     "create always needs unknown code_ref",
			plan:     withSource("draft", "app", hashA, types.StringNull(), draftSpec),
			state:    nil,
			isCreate: true,
			want:     true,
		},
		{
			name:     "nil state on update needs unknown",
			plan:     withSource("draft", "app", hashA, artifactOne, draftSpec),
			state:    nil,
			isCreate: false,
			want:     true,
		},
		{
			name:     "draft update with unchanged source skips unknown",
			plan:     withSource("draft", "app", hashA, artifactOne, draftSpec),
			state:    withSource("draft", "app", hashA, artifactOne, draftSpec),
			isCreate: false,
			want:     false,
		},
		{
			name:     "draft update with changed dir_hash needs unknown",
			plan:     withSource("draft", "app", hashB, artifactOne, draftSpec),
			state:    withSource("draft", "app", hashA, artifactOne, draftSpec),
			isCreate: false,
			want:     true,
		},
		{
			name:     "plan artifact_id unknown needs unknown",
			plan:     withSource("draft", "app", hashA, types.StringUnknown(), draftSpec),
			state:    withSource("draft", "app", hashA, artifactOne, draftSpec),
			isCreate: false,
			want:     true,
		},
		{
			name:     "new artifact version id needs unknown",
			plan:     withSource("locked", "app", hashA, artifactTwo, draftSpec),
			state:    withSource("locked", "app", hashA, artifactOne, draftSpec),
			isCreate: false,
			want:     true,
		},
		{
			name:     "locked to draft transition needs unknown",
			plan:     withSource("draft", "app", hashA, artifactOne, draftSpec),
			state:    withSource("locked", "app", hashA, artifactOne, draftSpec),
			isCreate: false,
			want:     true,
		},
		{
			name: "locked spec change without source upload skips unknown code_ref",
			plan: withSource("locked", "app-v2", hashA, artifactOne, draftSpec),
			state: withSource("locked", "app", hashA, artifactOne, &ArtifactSpecModel{
				ContainerGroups: []ArtifactContainerGroupModel{{
					Containers: []ArtifactContainerModel{{
						Primary:  types.BoolValue(true),
						Port:     types.Int64Value(8080),
						ImageURI: types.StringValue("nginx:latest"),
					}},
				}},
			}),
			isCreate: false,
			want:     false,
		},
		{
			name:     "locked unchanged spec and source skips unknown",
			plan:     withSource("locked", "app", hashA, artifactOne, draftSpec),
			state:    withSource("locked", "app", hashA, artifactOne, draftSpec),
			isCreate: false,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := sourceManagedCodeRefNeedsUnknown(tt.plan, tt.state, tt.isCreate); got != tt.want {
				t.Fatalf("sourceManagedCodeRefNeedsUnknown() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplySourceManagedCodeRefsToPlan(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dirHashA := types.StringValue("hash-a")
	dirHashB := types.StringValue("hash-b")
	stateCodeRef := &ArtifactCodeRefModel{
		CatalogID:        types.StringValue("aaaaaaaaaaaaaaaaaaaaaaaa"),
		CatalogVersionID: types.StringValue("bbbbbbbbbbbbbbbbbbbbbbbb"),
	}

	tests := []struct {
		name     string
		plan     *ArtifactResourceModel
		state    *ArtifactResourceModel
		isCreate bool
		check    func(t *testing.T, plan *ArtifactResourceModel)
	}{
		{
			name: "create clears code_ref on primary until apply",
			plan: testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithBuildConfig())),
			check: func(t *testing.T, plan *ArtifactResourceModel) {
				codeRef := plan.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig.CodeRef
				if !codeRef.IsUnknown() {
					t.Fatalf("expected unknown code_ref on primary before apply, got %#v", codeRef)
				}
			},
			isCreate: true,
		},
		{
			name: "create clears code_ref on sole container without primary flag",
			plan: testSourcePlanModel(t, dir, testDraftSourceSpec(testSoleWithBuildConfig())),
			check: func(t *testing.T, plan *ArtifactResourceModel) {
				codeRef := plan.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig.CodeRef
				if !codeRef.IsUnknown() {
					t.Fatal("expected sole container to have unknown code_ref before apply")
				}
			},
			isCreate: true,
		},
		{
			name: "create skips non-primary container",
			plan: testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithBuildConfig(), testSidecarWithBuildConfig())),
			check: func(t *testing.T, plan *ArtifactResourceModel) {
				primaryCodeRef := plan.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig.CodeRef
				if !primaryCodeRef.IsUnknown() {
					t.Fatalf("expected unknown code_ref on primary before apply, got %#v", primaryCodeRef)
				}
				sidecarCodeRef := plan.Spec.ContainerGroups[0].Containers[1].ImageBuildConfig.CodeRef
				if !sidecarCodeRef.IsNull() {
					t.Fatalf("expected no code_ref on non-primary container, got %#v", sidecarCodeRef)
				}
			},
			isCreate: true,
		},
		{
			name: "update with unchanged source copies code_ref from state to primary only",
			plan: testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithBuildConfig(), testSidecarWithBuildConfig()), func(m *ArtifactResourceModel) {
				m.ArtifactID = types.StringValue("artifact-1")
				m.Source.DirHash = dirHashA
			}),
			state: testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithCodeRef(stateCodeRef), testSidecarWithBuildConfig()), func(m *ArtifactResourceModel) {
				m.ArtifactID = types.StringValue("artifact-1")
				m.Source.DirHash = dirHashA
			}),
			check: func(t *testing.T, plan *ArtifactResourceModel) {
				primaryCodeRef := imageBuildConfigCodeRef(plan.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig)
				if primaryCodeRef == nil || primaryCodeRef.CatalogID.ValueString() != stateCodeRef.CatalogID.ValueString() {
					t.Fatalf("expected primary code_ref copied from state, got %#v", primaryCodeRef)
				}
				sidecarCodeRef := imageBuildConfigCodeRef(plan.Spec.ContainerGroups[0].Containers[1].ImageBuildConfig)
				if sidecarCodeRef != nil {
					t.Fatalf("expected non-primary container to remain without code_ref, got %#v", sidecarCodeRef)
				}
			},
		},
		{
			name: "update with changed dir_hash clears code_ref on primary only",
			plan: testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithBuildConfig(), testSidecarWithBuildConfig()), func(m *ArtifactResourceModel) {
				m.ArtifactID = types.StringValue("artifact-1")
				m.Source.DirHash = dirHashB
			}),
			state: testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithCodeRef(stateCodeRef), testSidecarWithBuildConfig()), func(m *ArtifactResourceModel) {
				m.ArtifactID = types.StringValue("artifact-1")
				m.Source.DirHash = dirHashA
			}),
			check: func(t *testing.T, plan *ArtifactResourceModel) {
				primaryCodeRef := plan.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig.CodeRef
				if !primaryCodeRef.IsUnknown() {
					t.Fatalf("expected unknown code_ref on primary before re-upload, got %#v", primaryCodeRef)
				}
				if !plan.Spec.ContainerGroups[0].Containers[1].ImageBuildConfig.CodeRef.IsNull() {
					t.Fatal("expected non-primary container to remain without code_ref")
				}
			},
		},
		{
			name: "no-op without source block",
			plan: &ArtifactResourceModel{
				Spec: testDraftSourceSpec(testPrimaryWithBuildConfig()),
			},
			check: func(t *testing.T, plan *ArtifactResourceModel) {
				if imageBuildConfigCodeRef(plan.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig) != nil {
					t.Fatal("expected no code_ref changes without source")
				}
			},
		},
		{
			name: "no-op when manual code_ref is set",
			plan: testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithCodeRef(stateCodeRef))),
			check: func(t *testing.T, plan *ArtifactResourceModel) {
				codeRef := imageBuildConfigCodeRef(plan.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig)
				if codeRef == nil || codeRef.CatalogID.ValueString() != stateCodeRef.CatalogID.ValueString() {
					t.Fatalf("expected manual code_ref to be untouched, got %#v", codeRef)
				}
			},
			isCreate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plan := tt.plan
			applySourceManagedCodeRefsToPlan(plan, tt.state, tt.isCreate)
			tt.check(t, plan)
		})
	}
}

func TestArtifactLockedSourceCloneNeeded(t *testing.T) {
	t.Parallel()

	hashA := types.StringValue("hash-a")
	hashB := types.StringValue("hash-b")
	dir := t.TempDir()
	draftSpec := &ArtifactSpecModel{
		ContainerGroups: []ArtifactContainerGroupModel{{
			Containers: []ArtifactContainerModel{{
				Primary: types.BoolValue(true),
				Port:    types.Int64Value(8080),
				ImageBuildConfig: &ArtifactImageBuildConfigModel{
					Dockerfile: &ArtifactDockerfileModel{Source: types.StringValue("provided")},
				},
			}},
		}},
	}
	artifactOne := types.StringValue("artifact-1")

	modelWithSource := func(status string, hash types.String) ArtifactResourceModel {
		return ArtifactResourceModel{
			Status:     types.StringValue(status),
			Name:       types.StringValue("app"),
			ArtifactID: artifactOne,
			Source: &ArtifactSourceModel{
				Dir:     types.StringValue(dir),
				DirHash: hash,
			},
			Spec: draftSpec,
		}
	}

	tests := []struct {
		name  string
		plan  ArtifactResourceModel
		state ArtifactResourceModel
		want  bool
	}{
		{
			name:  "locked source change needs clone",
			plan:  modelWithSource("locked", hashB),
			state: modelWithSource("locked", hashA),
			want:  true,
		},
		{
			name:  "locked unchanged source skips clone",
			plan:  modelWithSource("locked", hashA),
			state: modelWithSource("locked", hashA),
			want:  false,
		},
		{
			name:  "draft source change does not use locked clone",
			plan:  modelWithSource("draft", hashB),
			state: modelWithSource("draft", hashA),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := artifactLockedSourceCloneNeeded(tt.plan, tt.state); got != tt.want {
				t.Fatalf("artifactLockedSourceCloneNeeded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestArtifactModifyPlanNeedsUnknownArtifactID(t *testing.T) {
	t.Parallel()

	hashA := types.StringValue("hash-a")
	hashB := types.StringValue("hash-b")
	dir := t.TempDir()
	draftSpec := &ArtifactSpecModel{
		ContainerGroups: []ArtifactContainerGroupModel{{
			Containers: []ArtifactContainerModel{{
				Primary: types.BoolValue(true),
				Port:    types.Int64Value(8080),
				ImageBuildConfig: &ArtifactImageBuildConfigModel{
					Dockerfile: &ArtifactDockerfileModel{Source: types.StringValue("provided")},
				},
			}},
		}},
	}
	artifactOne := types.StringValue("artifact-1")

	modelWithSource := func(status string, hash types.String) ArtifactResourceModel {
		return ArtifactResourceModel{
			Status:     types.StringValue(status),
			Name:       types.StringValue("app"),
			ArtifactID: artifactOne,
			Source: &ArtifactSourceModel{
				Dir:     types.StringValue(dir),
				DirHash: hash,
			},
			Spec: draftSpec,
		}
	}

	tests := []struct {
		name  string
		plan  ArtifactResourceModel
		state ArtifactResourceModel
		want  bool
	}{
		{
			name:  "locked source-only change needs unknown artifact_id",
			plan:  modelWithSource("locked", hashB),
			state: modelWithSource("locked", hashA),
			want:  true,
		},
		{
			name:  "locked unchanged source and spec skips unknown",
			plan:  modelWithSource("locked", hashA),
			state: modelWithSource("locked", hashA),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := artifactModifyPlanNeedsUnknownArtifactID(tt.plan, tt.state); got != tt.want {
				t.Fatalf("artifactModifyPlanNeedsUnknownArtifactID() = %v, want %v", got, tt.want)
			}
		})
	}
}
