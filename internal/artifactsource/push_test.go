package artifactsource_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockFilesAPI struct {
	mu sync.Mutex

	createCatalogCalls      int
	createStageCalls        int
	uploadToStagePaths      []string
	applyStageCalls         int
	uploadFromZipNewCalls   int
	uploadFromZipExistCalls int
	pollStatusCalls         int
	deletePaths             []string
	deleteCalls             int

	catalogID string
	stageID   string
	version   int
}

func (m *mockFilesAPI) CreateCatalog(_ context.Context) (*filesapi.CatalogResp, error) {
	m.createCatalogCalls++
	if m.catalogID == "" {
		m.catalogID = "cat-new"
	}
	m.version++
	return &filesapi.CatalogResp{CatalogID: m.catalogID, CatalogVersionID: versionID(m.version)}, nil
}

func (m *mockFilesAPI) CreateStage(_ context.Context, catalogID string) (*filesapi.StageResp, error) {
	m.createStageCalls++
	m.stageID = "stage-1"
	return &filesapi.StageResp{CatalogID: catalogID, StageID: m.stageID}, nil
}

func (m *mockFilesAPI) UploadToStage(_ context.Context, _, _, name string, _ int64, _ io.Reader) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.uploadToStagePaths = append(m.uploadToStagePaths, name)
	return nil
}

func (m *mockFilesAPI) ApplyStage(_ context.Context, catalogID, _, _ string) (*filesapi.ApplyStageResp, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.applyStageCalls++
	m.version++
	return &filesapi.ApplyStageResp{CatalogID: catalogID, CatalogVersionID: versionID(m.version), NumFiles: len(m.uploadToStagePaths)}, nil
}

func (m *mockFilesAPI) UploadFromZipNew(_ context.Context, _ string, _ int64, _ io.Reader) (*filesapi.FromFileResp, error) {
	m.uploadFromZipNewCalls++
	if m.catalogID == "" {
		m.catalogID = "cat-zip-new"
	}
	m.version++
	return &filesapi.FromFileResp{CatalogID: m.catalogID, CatalogVersionID: versionID(m.version)}, nil
}

func (m *mockFilesAPI) UploadFromZipExisting(_ context.Context, catalogID, _, _ string, _ int64, _ io.Reader) (*filesapi.FromFileResp, error) {
	m.uploadFromZipExistCalls++
	m.version++
	return &filesapi.FromFileResp{CatalogID: catalogID, CatalogVersionID: versionID(m.version), StatusID: "status-1"}, nil
}

func (m *mockFilesAPI) PollStatus(_ context.Context, _ string) (*filesapi.StatusResp, error) {
	m.pollStatusCalls++
	return &filesapi.StatusResp{Status: filesapi.StatusCompleted}, nil
}

func (m *mockFilesAPI) AllFiles(context.Context, string, string) (map[string]filesapi.FileMeta, error) {
	panic("not used")
}

func (m *mockFilesAPI) DownloadFile(context.Context, string, string, string, io.Writer) (string, int64, error) {
	panic("not used")
}

func (m *mockFilesAPI) DeleteFiles(_ context.Context, _ string, paths []string) (*filesapi.DeleteFilesResp, error) {
	m.deleteCalls++
	m.deletePaths = append(m.deletePaths, paths...)
	m.version++
	return &filesapi.DeleteFilesResp{CatalogVersionID: versionID(m.version)}, nil
}

func (m *mockFilesAPI) ListVersions(context.Context, string, int) ([]filesapi.CatalogVersion, error) {
	panic("not used")
}

func versionID(n int) string {
	return fmt.Sprintf("ver-%d", n)
}

func TestPushDirectory_StagePathSmallTree(t *testing.T) {
	t.Parallel()

	root := writeSmallTree(t)
	mock := &mockFilesAPI{}

	result, err := artifactsource.PushDirectory(context.Background(), mock, artifactsource.Options{Dir: root})
	require.NoError(t, err)

	assert.Equal(t, "cat-new", result.CatalogID)
	assert.Equal(t, "ver-2", result.CatalogVersionID)
	assert.Equal(t, 2, result.FileCount)
	assert.NotEmpty(t, result.SourceHash)
	assert.Len(t, result.FileHashes, 2)
	assert.False(t, result.Incremental)

	assert.Equal(t, 1, mock.createCatalogCalls)
	assert.Equal(t, 1, mock.createStageCalls)
	assert.Equal(t, 2, len(mock.uploadToStagePaths))
	assert.Equal(t, 1, mock.applyStageCalls)
	assert.Zero(t, mock.uploadFromZipNewCalls)
}

func TestPushDirectory_ExistingCatalogStagePath(t *testing.T) {
	t.Parallel()

	root := writeSmallTree(t)
	mock := &mockFilesAPI{catalogID: "cat-existing"}

	result, err := artifactsource.PushDirectory(context.Background(), mock, artifactsource.Options{
		Dir:       root,
		CatalogID: "cat-existing",
	})
	require.NoError(t, err)

	assert.Equal(t, "cat-existing", result.CatalogID)
	assert.Zero(t, mock.createCatalogCalls)
	assert.Equal(t, 1, mock.createStageCalls)
}

func TestPushDirectory_ZipPathLargeTree(t *testing.T) {
	t.Parallel()

	root := writeLargeTree(t, artifactsource.StageVsZipFileThreshold+1)
	mock := &mockFilesAPI{}

	result, err := artifactsource.PushDirectory(context.Background(), mock, artifactsource.Options{Dir: root})
	require.NoError(t, err)

	assert.Equal(t, "cat-zip-new", result.CatalogID)
	assert.Equal(t, artifactsource.StageVsZipFileThreshold+1, result.FileCount)
	assert.Len(t, result.FileHashes, artifactsource.StageVsZipFileThreshold+1)
	assert.Equal(t, 1, mock.uploadFromZipNewCalls)
	assert.Zero(t, mock.createStageCalls)
}

func TestPushDirectory_ZipPathExistingCatalogPolls(t *testing.T) {
	t.Parallel()

	root := writeLargeTree(t, artifactsource.StageVsZipFileThreshold+1)
	mock := &mockFilesAPI{catalogID: "cat-zip-existing"}

	result, err := artifactsource.PushDirectory(context.Background(), mock, artifactsource.Options{
		Dir:       root,
		CatalogID: "cat-zip-existing",
	})
	require.NoError(t, err)

	assert.Equal(t, "cat-zip-existing", result.CatalogID)
	assert.Equal(t, 1, mock.uploadFromZipExistCalls)
	assert.Equal(t, 1, mock.pollStatusCalls)
}

func TestPushDirectory_IncrementalNoChanges(t *testing.T) {
	t.Parallel()

	root := writeSmallTree(t)
	mock := &mockFilesAPI{catalogID: "cat-existing", version: 5}

	first, err := artifactsource.PushDirectory(context.Background(), mock, artifactsource.Options{Dir: root})
	require.NoError(t, err)

	mock2 := &mockFilesAPI{catalogID: "cat-existing", version: 5}
	second, err := artifactsource.PushDirectory(context.Background(), mock2, artifactsource.Options{
		Dir:              root,
		CatalogID:        first.CatalogID,
		CatalogVersionID: first.CatalogVersionID,
		BaseFiles:        first.FileHashes,
	})
	require.NoError(t, err)

	assert.Equal(t, first.CatalogVersionID, second.CatalogVersionID)
	assert.Equal(t, first.FileHashes, second.FileHashes)
	assert.Zero(t, mock2.createStageCalls)
	assert.Zero(t, mock2.deleteCalls)
}

func TestPushDirectory_IncrementalUploadsOnlyChangedFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.txt"), []byte("aaa"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "b.txt"), []byte("bbb"), 0o644))

	mock := &mockFilesAPI{}
	first, err := artifactsource.PushDirectory(context.Background(), mock, artifactsource.Options{Dir: root})
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(root, "b.txt"), []byte("changed"), 0o644))

	mock2 := &mockFilesAPI{catalogID: first.CatalogID, version: 2}
	second, err := artifactsource.PushDirectory(context.Background(), mock2, artifactsource.Options{
		Dir:              root,
		CatalogID:        first.CatalogID,
		CatalogVersionID: first.CatalogVersionID,
		BaseFiles:        first.FileHashes,
	})
	require.NoError(t, err)

	assert.True(t, second.Incremental)
	assert.Equal(t, []string{"b.txt"}, mock2.uploadToStagePaths)
	assert.Equal(t, 1, mock2.createStageCalls)
	assert.Zero(t, mock2.deleteCalls)
	assert.NotEqual(t, first.SourceHash, second.SourceHash)
	assert.NotEqual(t, first.FileHashes["b.txt"], second.FileHashes["b.txt"])
}

func TestPushDirectory_IncrementalDeletesRemovedFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.txt"), []byte("aaa"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "b.txt"), []byte("bbb"), 0o644))

	mock := &mockFilesAPI{}
	first, err := artifactsource.PushDirectory(context.Background(), mock, artifactsource.Options{Dir: root})
	require.NoError(t, err)

	require.NoError(t, os.Remove(filepath.Join(root, "b.txt")))

	mock2 := &mockFilesAPI{catalogID: first.CatalogID, version: 2}
	second, err := artifactsource.PushDirectory(context.Background(), mock2, artifactsource.Options{
		Dir:              root,
		CatalogID:        first.CatalogID,
		CatalogVersionID: first.CatalogVersionID,
		BaseFiles:        first.FileHashes,
	})
	require.NoError(t, err)

	assert.True(t, second.Incremental)
	assert.Equal(t, []string{"b.txt"}, mock2.deletePaths)
	assert.Equal(t, 1, mock2.deleteCalls)
	assert.Empty(t, mock2.uploadToStagePaths)
	assert.Len(t, second.FileHashes, 1)
}

func TestPushDirectory_EmptyDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mock := &mockFilesAPI{}

	_, err := artifactsource.PushDirectory(context.Background(), mock, artifactsource.Options{Dir: root})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no uploadable files")
}

func TestPushDirectory_NilClient(t *testing.T) {
	t.Parallel()

	_, err := artifactsource.PushDirectory(context.Background(), nil, artifactsource.Options{Dir: t.TempDir()})
	require.Error(t, err)
}

func writeSmallTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.txt"), []byte("aaa"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "b.txt"), []byte("bbb"), 0o644))
	return root
}

func writeLargeTree(t *testing.T, count int) string {
	t.Helper()
	root := t.TempDir()
	for i := 0; i < count; i++ {
		name := filepath.Join(root, "file-"+padIndex(i)+".txt")
		require.NoError(t, os.WriteFile(name, []byte("x"), 0o644))
	}
	return root
}

func padIndex(i int) string {
	return fmt.Sprintf("%02d", i)
}
