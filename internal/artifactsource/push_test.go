package artifactsource_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	allFilesCalls           int
	allFilesArgs            []string
	listVersionsCalls       int

	// allFiles is the catalog version listing AllFiles serves, and allFilesErr
	// the failure it reports instead.
	allFiles     map[string]filesapi.FileMeta
	allFilesErr  error
	listVersions []filesapi.CatalogVersion

	// deleteOmitsVersion makes DeleteFiles answer without naming the version it
	// produced, which is what an API that replies 204 No Content looks like
	// here, since the body is never decoded.
	deleteOmitsVersion bool

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

func (m *mockFilesAPI) AllFiles(_ context.Context, catalogID, versionID string) (map[string]filesapi.FileMeta, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.allFilesCalls++
	m.allFilesArgs = append(m.allFilesArgs, catalogID+"@"+versionID)
	if m.allFilesErr != nil {
		return nil, m.allFilesErr
	}
	return m.allFiles, nil
}

func (m *mockFilesAPI) DownloadFile(context.Context, string, string, string, io.Writer) (string, int64, error) {
	panic("not used")
}

func (m *mockFilesAPI) DeleteFiles(_ context.Context, _ string, paths []string) (*filesapi.DeleteFilesResp, error) {
	m.deleteCalls++
	m.deletePaths = append(m.deletePaths, paths...)
	m.version++
	if m.deleteOmitsVersion {
		return &filesapi.DeleteFilesResp{}, nil
	}
	return &filesapi.DeleteFilesResp{CatalogVersionID: versionID(m.version)}, nil
}

func (m *mockFilesAPI) ListVersions(context.Context, string, int) ([]filesapi.CatalogVersion, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listVersionsCalls++
	if m.listVersions == nil {
		return nil, nil
	}
	return m.listVersions, nil
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

// remoteFromManifest describes a catalog version holding exactly what a previous
// push uploaded. Deriving it from that push's own hashes keeps the two sides of
// the comparison honest: the test never restates how a file is hashed.
func remoteFromManifest(m artifactsource.Manifest) map[string]filesapi.FileMeta {
	out := make(map[string]filesapi.FileMeta, len(m))
	for path, meta := range m {
		out[path] = filesapi.FileMeta{Hash: meta.Hash, Size: meta.Size}
	}
	return out
}

func TestPushDirectory_IncrementalNoChanges(t *testing.T) {
	t.Parallel()

	root := writeSmallTree(t)
	mock := &mockFilesAPI{catalogID: "cat-existing", version: 5}

	first, err := artifactsource.PushDirectory(context.Background(), mock, artifactsource.Options{Dir: root})
	require.NoError(t, err)

	mock2 := &mockFilesAPI{
		catalogID: "cat-existing",
		version:   5,
		allFiles:  remoteFromManifest(first.FileHashes),
	}
	second, err := artifactsource.PushDirectory(context.Background(), mock2, artifactsource.Options{
		Dir:              root,
		CatalogID:        first.CatalogID,
		CatalogVersionID: first.CatalogVersionID,
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

	mock2 := &mockFilesAPI{
		catalogID: first.CatalogID,
		version:   2,
		allFiles:  remoteFromManifest(first.FileHashes),
	}
	second, err := artifactsource.PushDirectory(context.Background(), mock2, artifactsource.Options{
		Dir:              root,
		CatalogID:        first.CatalogID,
		CatalogVersionID: first.CatalogVersionID,
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

	mock2 := &mockFilesAPI{
		catalogID: first.CatalogID,
		version:   2,
		allFiles:  remoteFromManifest(first.FileHashes),
	}
	second, err := artifactsource.PushDirectory(context.Background(), mock2, artifactsource.Options{
		Dir:              root,
		CatalogID:        first.CatalogID,
		CatalogVersionID: first.CatalogVersionID,
	})
	require.NoError(t, err)

	assert.True(t, second.Incremental)
	assert.Equal(t, []string{"b.txt"}, mock2.deletePaths)
	assert.Equal(t, 1, mock2.deleteCalls)
	assert.Empty(t, mock2.uploadToStagePaths)
	assert.Len(t, second.FileHashes, 1)

	// A delete-only push still has to name the version it produced: this value
	// is written into the artifact's code_ref, and an empty one would leave the
	// build with no code and unpin the base for every later push.
	assert.NotEmpty(t, second.CatalogVersionID)
	assert.NotEqual(t, first.CatalogVersionID, second.CatalogVersionID)
}

// The ordinary shape of a real edit: some files changed, others removed.
func TestPushDirectory_IncrementalUploadsAndDeletesTogether(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "keep.txt"), []byte("keep"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "edit.txt"), []byte("before"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "drop.txt"), []byte("drop"), 0o644))

	mock := &mockFilesAPI{}
	first, err := artifactsource.PushDirectory(context.Background(), mock, artifactsource.Options{Dir: root})
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(root, "edit.txt"), []byte("after"), 0o644))
	require.NoError(t, os.Remove(filepath.Join(root, "drop.txt")))
	require.NoError(t, os.WriteFile(filepath.Join(root, "add.txt"), []byte("add"), 0o644))

	mock2 := &mockFilesAPI{
		catalogID: first.CatalogID,
		version:   2,
		allFiles:  remoteFromManifest(first.FileHashes),
	}
	second, err := artifactsource.PushDirectory(context.Background(), mock2, artifactsource.Options{
		Dir:              root,
		CatalogID:        first.CatalogID,
		CatalogVersionID: first.CatalogVersionID,
	})
	require.NoError(t, err)

	assert.True(t, second.Incremental)
	assert.ElementsMatch(t, []string{"edit.txt", "add.txt"}, mock2.uploadToStagePaths)
	assert.Equal(t, []string{"drop.txt"}, mock2.deletePaths)
	// The upload runs after the delete, so its version is the one that survives.
	assert.NotEmpty(t, second.CatalogVersionID)
	assert.Zero(t, mock2.listVersionsCalls)
}

// A delete-only push against an API that answers without naming the new version
// has to look it up rather than report an empty one.
func TestPushDirectory_DeleteOnlyResolvesVersionWhenResponseOmitsIt(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.txt"), []byte("aaa"), 0o644))

	mock := &mockFilesAPI{}
	first, err := artifactsource.PushDirectory(context.Background(), mock, artifactsource.Options{Dir: root})
	require.NoError(t, err)

	remote := remoteFromManifest(first.FileHashes)
	remote["gone.txt"] = filesapi.FileMeta{Hash: "whatever", Size: 3}

	mock2 := &mockFilesAPI{
		catalogID:          first.CatalogID,
		version:            2,
		allFiles:           remote,
		deleteOmitsVersion: true,
		listVersions:       []filesapi.CatalogVersion{{ID: "ver-after-delete"}},
	}
	second, err := artifactsource.PushDirectory(context.Background(), mock2, artifactsource.Options{
		Dir:              root,
		CatalogID:        first.CatalogID,
		CatalogVersionID: first.CatalogVersionID,
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"gone.txt"}, mock2.deletePaths)
	assert.Equal(t, "ver-after-delete", second.CatalogVersionID)
	assert.Equal(t, 1, mock2.listVersionsCalls)
}

func TestPushDirectory_DeleteOnlyFailsWhenVersionCannotBeResolved(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.txt"), []byte("aaa"), 0o644))

	mock := &mockFilesAPI{}
	first, err := artifactsource.PushDirectory(context.Background(), mock, artifactsource.Options{Dir: root})
	require.NoError(t, err)

	remote := remoteFromManifest(first.FileHashes)
	remote["gone.txt"] = filesapi.FileMeta{Hash: "whatever", Size: 3}

	mock2 := &mockFilesAPI{
		catalogID:          first.CatalogID,
		version:            2,
		allFiles:           remote,
		deleteOmitsVersion: true,
	}
	_, err = artifactsource.PushDirectory(context.Background(), mock2, artifactsource.Options{
		Dir:              root,
		CatalogID:        first.CatalogID,
		CatalogVersionID: first.CatalogVersionID,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve catalog version after delete")
}

// The base decides deletions, so it must never hold a path the walk could not
// have produced. An ignored path is one the push would refuse to upload, and
// deleting it would strip files from the catalog that the user never touched.
func TestPushDirectory_BaseExcludesIgnoredPaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "app.py"), []byte("app"), 0o644))

	mock := &mockFilesAPI{}
	first, err := artifactsource.PushDirectory(context.Background(), mock, artifactsource.Options{Dir: root})
	require.NoError(t, err)

	// The catalog still holds what earlier pushes sent, before a rule excluded it.
	remote := remoteFromManifest(first.FileHashes)
	remote[".venv/lib/python3.11/site-packages/x.py"] = filesapi.FileMeta{Hash: "aa", Size: 1}
	remote["secrets.tfvars"] = filesapi.FileMeta{Hash: "bb", Size: 1}

	ignoreVenvAndTfvars := func(relPath string, _ bool) bool {
		return strings.HasPrefix(relPath, ".venv") || strings.HasSuffix(relPath, ".tfvars")
	}

	mock2 := &mockFilesAPI{catalogID: first.CatalogID, version: 2, allFiles: remote}
	_, err = artifactsource.PushDirectory(context.Background(), mock2, artifactsource.Options{
		Dir:              root,
		CatalogID:        first.CatalogID,
		CatalogVersionID: first.CatalogVersionID,
		Ignore:           ignoreVenvAndTfvars,
	})
	require.NoError(t, err)

	assert.Empty(t, mock2.deletePaths)
	assert.Zero(t, mock2.deleteCalls)
}

// An entry with no checksum cannot take part in a content comparison, and the
// listing is not promised to describe only regular files, so handing such a
// path to the delete call could take a whole subtree with it.
func TestPushDirectory_BaseSkipsEntriesWithoutChecksum(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "app.py"), []byte("app"), 0o644))

	mock := &mockFilesAPI{}
	first, err := artifactsource.PushDirectory(context.Background(), mock, artifactsource.Options{Dir: root})
	require.NoError(t, err)

	remote := remoteFromManifest(first.FileHashes)
	remote["src"] = filesapi.FileMeta{Hash: "", Size: 0}

	mock2 := &mockFilesAPI{catalogID: first.CatalogID, version: 2, allFiles: remote}
	_, err = artifactsource.PushDirectory(context.Background(), mock2, artifactsource.Options{
		Dir:              root,
		CatalogID:        first.CatalogID,
		CatalogVersionID: first.CatalogVersionID,
	})
	require.NoError(t, err)

	assert.Empty(t, mock2.deletePaths)
}

// A version that lists nothing is still a base: everything local is new, and
// there is nothing to remove. Treating it as no base at all would quietly skip
// the deletes on every later push.
func TestPushDirectory_EmptyBaseUploadsEverythingWithoutWarning(t *testing.T) {
	t.Parallel()

	root := writeSmallTree(t)
	mock := &mockFilesAPI{
		catalogID: "cat-1",
		version:   3,
		allFiles:  map[string]filesapi.FileMeta{},
	}

	result, err := artifactsource.PushDirectory(context.Background(), mock, artifactsource.Options{
		Dir:              root,
		CatalogID:        "cat-1",
		CatalogVersionID: "ver-3",
	})
	require.NoError(t, err)

	assert.NoError(t, result.BaseUnavailable)
	assert.True(t, result.Incremental)
	assert.Len(t, mock.uploadToStagePaths, 2)
	assert.Zero(t, mock.deleteCalls)
}

// remoteManifest turns a path -> contents map into the listing AllFiles would
// return for a catalog version holding exactly those files, so a test can state
// the remote side the way it states the local one.
func TestPushDirectory_BaseFailureFallsBackToFullUpload(t *testing.T) {
	t.Parallel()

	root := writeSmallTree(t)
	mock := &mockFilesAPI{
		catalogID:   "cat-1",
		version:     2,
		allFilesErr: errors.New("catalog version expired"),
	}

	result, err := artifactsource.PushDirectory(context.Background(), mock, artifactsource.Options{
		Dir:              root,
		CatalogID:        "cat-1",
		CatalogVersionID: "ver-2",
	})
	require.NoError(t, err)

	require.Error(t, result.BaseUnavailable)
	assert.Contains(t, result.BaseUnavailable.Error(), "catalog version expired")
	assert.Contains(t, result.BaseUnavailable.Error(), "ver-2")

	assert.False(t, result.Incremental)
	assert.Len(t, mock.uploadToStagePaths, 2)
	// No base means no way to tell a deletion from a file that was never there.
	assert.Zero(t, mock.deleteCalls)
}

// A first push has no version to diff against and must not go looking for one.
func TestPushDirectory_NoPinnedVersionSkipsBaseLookup(t *testing.T) {
	t.Parallel()

	root := writeSmallTree(t)
	mock := &mockFilesAPI{catalogID: "cat-1"}

	result, err := artifactsource.PushDirectory(context.Background(), mock, artifactsource.Options{
		Dir:       root,
		CatalogID: "cat-1",
	})
	require.NoError(t, err)

	assert.Zero(t, mock.allFilesCalls)
	assert.NoError(t, result.BaseUnavailable)
	assert.False(t, result.Incremental)
	assert.Len(t, mock.uploadToStagePaths, 2)
}

// An explicitly supplied base is used as given, without a lookup.
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
