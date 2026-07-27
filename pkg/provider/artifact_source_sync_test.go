package provider

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

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
							CodeRef: &ArtifactCodeRefModel{
								CatalogID:        id,
								CatalogVersionID: types.StringValue("bbbbbbbbbbbbbbbbbbbbbbbb"),
							},
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
						CodeRef: &ArtifactCodeRefModel{
							CatalogID:        types.StringValue("aaaaaaaaaaaaaaaaaaaaaaaa"),
							CatalogVersionID: id,
						},
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

		got, err := resource.syncArtifactSource(context.Background(), plan, state, artifact, artifactID)
		if err != nil {
			t.Fatalf("syncArtifactSource() error = %v", err)
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

		if _, err := resource.syncArtifactSource(context.Background(), plan, nil, &client.Artifact{ID: artifactID}, ""); err != nil {
			t.Fatalf("syncArtifactSource() error = %v", err)
		}
		if filesAPI.createCatalogCalls == 0 && filesAPI.uploadFromZipNewCalls == 0 {
			t.Fatal("expected Files API upload during create")
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

		if _, err := resource.syncArtifactSource(context.Background(), plan, state, &client.Artifact{ID: artifactID}, artifactID); err != nil {
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

		if _, err := resource.syncArtifactSource(context.Background(), plan, state, artifact, artifactID); err != nil {
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

		_, err := resource.syncArtifactSource(context.Background(), plan, state, &client.Artifact{ID: artifactID}, artifactID)
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

		_, err := resource.syncArtifactSource(context.Background(), plan, state, &client.Artifact{ID: artifactID}, artifactID)
		if err == nil {
			t.Fatal("expected patch error")
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

func artifactSpecWithCodeRef(catalogID, versionID string) *ArtifactSpecModel {
	return &ArtifactSpecModel{
		ContainerGroups: []ArtifactContainerGroupModel{{
			Containers: []ArtifactContainerModel{{
				Primary: types.BoolValue(true),
				ImageBuildConfig: &ArtifactImageBuildConfigModel{
					CodeRef: &ArtifactCodeRefModel{
						CatalogID:        types.StringValue(catalogID),
						CatalogVersionID: types.StringValue(versionID),
					},
				},
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

	createCatalogCalls    int
	uploadFromZipNewCalls int
}

func newSyncTestFilesAPI() *syncTestFilesAPI {
	return &syncTestFilesAPI{}
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

func (m *syncTestFilesAPI) UploadToStage(context.Context, string, string, string, int64, io.Reader) error {
	if m.uploadErr != nil {
		return m.uploadErr
	}
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
	return nil, nil
}

func (m *syncTestFilesAPI) DownloadFile(context.Context, string, string, string, io.Writer) (string, int64, error) {
	return "", 0, nil
}

func (m *syncTestFilesAPI) DeleteFiles(context.Context, string, []string) (*filesapi.DeleteFilesResp, error) {
	return &filesapi.DeleteFilesResp{}, nil
}

func (m *syncTestFilesAPI) ListVersions(context.Context, string, int) ([]filesapi.CatalogVersion, error) {
	return nil, nil
}

func syncTestVersionID(n int) string {
	return fmt.Sprintf("ver-%d", n)
}
