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

		first := data.Source.DirHash
		refreshArtifactSourceDirHash(data)
		if !data.Source.DirHash.Equal(first) {
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

func TestRollbackArtifactCreate(t *testing.T) {
	t.Parallel()

	repoID := "repo-123"

	t.Run("nil artifact is a no-op", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		resource := &ArtifactResource{provider: &Provider{service: mockService}}

		resource.rollbackArtifactCreate(context.Background(), nil, true)
	})

	t.Run("missing repository id is a no-op", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		resource := &ArtifactResource{provider: &Provider{service: mockService}}

		resource.rollbackArtifactCreate(context.Background(), &client.Artifact{ID: "artifact-1"}, true)
	})

	t.Run("skips delete when repository was user supplied", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		resource := &ArtifactResource{provider: &Provider{service: mockService}}

		resource.rollbackArtifactCreate(context.Background(), &client.Artifact{
			ID:                   "artifact-1",
			ArtifactRepositoryID: &repoID,
		}, false)
	})

	t.Run("deletes provisioned artifact repository", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockService := mock_client.NewMockService(ctrl)
		mockService.EXPECT().DeleteArtifactRepository(gomock.Any(), repoID).Return(nil)

		resource := &ArtifactResource{provider: &Provider{service: mockService}}
		resource.rollbackArtifactCreate(context.Background(), &client.Artifact{
			ID:                   "artifact-1",
			ArtifactRepositoryID: &repoID,
		}, true)
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
	container.ImageBuildConfig.CodeRef = codeRef
	return container
}

func testDraftSourceSpec(containers ...ArtifactContainerModel) *ArtifactSpecModel {
	return &ArtifactSpecModel{
		ContainerGroups: []ArtifactContainerGroupModel{{
			Containers: containers,
		}},
	}
}

func TestPrimaryCodeRefFromState(t *testing.T) {
	t.Parallel()

	stateCodeRef := &ArtifactCodeRefModel{
		CatalogID:        types.StringValue("aaaaaaaaaaaaaaaaaaaaaaaa"),
		CatalogVersionID: types.StringValue("bbbbbbbbbbbbbbbbbbbbbbbb"),
	}
	dir := t.TempDir()

	t.Run("finds primary container regardless of position", func(t *testing.T) {
		state := testSourcePlanModel(t, dir, testDraftSourceSpec(testSidecarWithBuildConfig(), testPrimaryWithCodeRef(stateCodeRef)), func(m *ArtifactResourceModel) {
			m.ArtifactID = types.StringValue("artifact-1")
		})
		got := primaryCodeRefFromState(state)
		if got == nil || got.CatalogID.ValueString() != stateCodeRef.CatalogID.ValueString() {
			t.Fatalf("primaryCodeRefFromState() = %#v, want primary code_ref", got)
		}
	})

	t.Run("nil state returns nil", func(t *testing.T) {
		if got := primaryCodeRefFromState(nil); got != nil {
			t.Fatalf("primaryCodeRefFromState(nil) = %#v, want nil", got)
		}
	})
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
			name: "locked spec change needs unknown",
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
			want:     true,
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
			name: "create sets unknown on primary",
			plan: testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithBuildConfig())),
			check: func(t *testing.T, plan *ArtifactResourceModel) {
				codeRef := plan.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig.CodeRef
				if codeRef == nil || !codeRef.CatalogID.IsUnknown() || !codeRef.CatalogVersionID.IsUnknown() {
					t.Fatalf("expected unknown code_ref on primary, got %#v", codeRef)
				}
			},
			isCreate: true,
		},
		{
			name: "create sets unknown on sole container without primary flag",
			plan: testSourcePlanModel(t, dir, testDraftSourceSpec(testSoleWithBuildConfig())),
			check: func(t *testing.T, plan *ArtifactResourceModel) {
				codeRef := plan.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig.CodeRef
				if codeRef == nil || !codeRef.CatalogID.IsUnknown() {
					t.Fatal("expected sole container to receive unknown code_ref")
				}
			},
			isCreate: true,
		},
		{
			name: "create skips non-primary container",
			plan: testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithBuildConfig(), testSidecarWithBuildConfig())),
			check: func(t *testing.T, plan *ArtifactResourceModel) {
				primaryCodeRef := plan.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig.CodeRef
				if primaryCodeRef == nil || !primaryCodeRef.CatalogID.IsUnknown() {
					t.Fatalf("expected unknown code_ref on primary, got %#v", primaryCodeRef)
				}
				sidecarCodeRef := plan.Spec.ContainerGroups[0].Containers[1].ImageBuildConfig.CodeRef
				if sidecarCodeRef != nil {
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
				primaryCodeRef := plan.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig.CodeRef
				if primaryCodeRef == nil || primaryCodeRef.CatalogID.ValueString() != stateCodeRef.CatalogID.ValueString() {
					t.Fatalf("expected primary code_ref copied from state, got %#v", primaryCodeRef)
				}
				sidecarCodeRef := plan.Spec.ContainerGroups[0].Containers[1].ImageBuildConfig.CodeRef
				if sidecarCodeRef != nil {
					t.Fatalf("expected non-primary container to remain without code_ref, got %#v", sidecarCodeRef)
				}
			},
		},
		{
			name: "update with reordered containers copies primary code_ref from state",
			plan: testSourcePlanModel(t, dir, testDraftSourceSpec(testSidecarWithBuildConfig(), testPrimaryWithBuildConfig()), func(m *ArtifactResourceModel) {
				m.ArtifactID = types.StringValue("artifact-1")
				m.Source.DirHash = dirHashA
			}),
			state: testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithCodeRef(stateCodeRef), testSidecarWithBuildConfig()), func(m *ArtifactResourceModel) {
				m.ArtifactID = types.StringValue("artifact-1")
				m.Source.DirHash = dirHashA
			}),
			check: func(t *testing.T, plan *ArtifactResourceModel) {
				primaryCodeRef := plan.Spec.ContainerGroups[0].Containers[1].ImageBuildConfig.CodeRef
				if primaryCodeRef == nil || primaryCodeRef.CatalogID.ValueString() != stateCodeRef.CatalogID.ValueString() {
					t.Fatalf("expected primary code_ref copied from state after reorder, got %#v", primaryCodeRef)
				}
				sidecarCodeRef := plan.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig.CodeRef
				if sidecarCodeRef != nil {
					t.Fatalf("expected non-primary container to remain without code_ref, got %#v", sidecarCodeRef)
				}
			},
		},
		{
			name: "update with changed dir_hash sets unknown on primary only",
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
				if primaryCodeRef == nil || !primaryCodeRef.CatalogID.IsUnknown() {
					t.Fatalf("expected unknown code_ref on primary after source change, got %#v", primaryCodeRef)
				}
				if plan.Spec.ContainerGroups[0].Containers[1].ImageBuildConfig.CodeRef != nil {
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
				if plan.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig.CodeRef != nil {
					t.Fatal("expected no code_ref changes without source")
				}
			},
		},
		{
			name: "no-op when manual code_ref is set",
			plan: testSourcePlanModel(t, dir, testDraftSourceSpec(testPrimaryWithCodeRef(stateCodeRef))),
			check: func(t *testing.T, plan *ArtifactResourceModel) {
				codeRef := plan.Spec.ContainerGroups[0].Containers[0].ImageBuildConfig.CodeRef
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
			name: "locked unchanged source ignores managed code_ref null in plan",
			plan: modelWithSource("locked", hashA),
			state: func() ArtifactResourceModel {
				m := modelWithSource("locked", hashA)
				spec := *m.Spec
				group := spec.ContainerGroups[0]
				container := group.Containers[0]
				container.ImageBuildConfig = &ArtifactImageBuildConfigModel{
					CodeRef: &ArtifactCodeRefModel{
						CatalogID:        types.StringValue("aaaaaaaaaaaaaaaaaaaaaaaa"),
						CatalogVersionID: types.StringValue("bbbbbbbbbbbbbbbbbbbbbbbb"),
					},
					Dockerfile: &ArtifactDockerfileModel{Source: types.StringValue("provided")},
				}
				group.Containers = []ArtifactContainerModel{container}
				spec.ContainerGroups = []ArtifactContainerGroupModel{group}
				m.Spec = &spec
				return m
			}(),
			want: false,
		},
		{
			name: "locked spec change with source needs clone",
			plan: func() ArtifactResourceModel {
				m := modelWithSource("locked", hashA)
				spec := *m.Spec
				group := spec.ContainerGroups[0]
				container := group.Containers[0]
				container.Port = types.Int64Value(9090)
				group.Containers = []ArtifactContainerModel{container}
				spec.ContainerGroups = []ArtifactContainerGroupModel{group}
				m.Spec = &spec
				return m
			}(),
			state: modelWithSource("locked", hashA),
			want:  true,
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
