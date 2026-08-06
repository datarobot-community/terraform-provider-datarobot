package artifactsource

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stageClientMock struct {
	mu sync.Mutex

	createCatalogCalls int
	createStageCalls   int
	applyStageCalls    int
	uploadPaths        []string

	createCatalogErr error
	createStageErr   error
	applyStageErr    error
	uploadErrForPath map[string]error
	uploadHook       func(name string) error

	activeUploads atomic.Int32
	maxConcurrent atomic.Int32

	catalogID string
	stageID   string
	version   int
}

func (m *stageClientMock) CreateCatalog(context.Context) (*filesapi.CatalogResp, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createCatalogCalls++
	if m.createCatalogErr != nil {
		return nil, m.createCatalogErr
	}
	if m.catalogID == "" {
		m.catalogID = "cat-test"
	}
	m.version++
	return &filesapi.CatalogResp{CatalogID: m.catalogID, CatalogVersionID: stageVersionID(m.version)}, nil
}

func (m *stageClientMock) CreateStage(_ context.Context, catalogID string) (*filesapi.StageResp, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createStageCalls++
	if m.createStageErr != nil {
		return nil, m.createStageErr
	}
	m.stageID = "stage-test"
	return &filesapi.StageResp{CatalogID: catalogID, StageID: m.stageID}, nil
}

func (m *stageClientMock) UploadToStage(_ context.Context, _, _, name string, _ int64, _ io.Reader) error {
	current := m.activeUploads.Add(1)
	defer m.activeUploads.Add(-1)

	for {
		prevMax := m.maxConcurrent.Load()
		if current <= prevMax {
			break
		}
		if m.maxConcurrent.CompareAndSwap(prevMax, current) {
			break
		}
	}

	if m.uploadHook != nil {
		if err := m.uploadHook(name); err != nil {
			return err
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if err, ok := m.uploadErrForPath[name]; ok {
		return err
	}
	m.uploadPaths = append(m.uploadPaths, name)
	return nil
}

func (m *stageClientMock) ApplyStage(_ context.Context, catalogID, _, _ string) (*filesapi.ApplyStageResp, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.applyStageCalls++
	if m.applyStageErr != nil {
		return nil, m.applyStageErr
	}
	m.version++
	return &filesapi.ApplyStageResp{CatalogID: catalogID, CatalogVersionID: stageVersionID(m.version)}, nil
}

func (m *stageClientMock) UploadFromZipNew(context.Context, string, int64, io.Reader) (*filesapi.FromFileResp, error) {
	panic("not used in stage tests")
}

func (m *stageClientMock) UploadFromZipExisting(context.Context, string, string, string, int64, io.Reader) (*filesapi.FromFileResp, error) {
	panic("not used in stage tests")
}

func (m *stageClientMock) PollStatus(context.Context, string) (*filesapi.StatusResp, error) {
	panic("not used in stage tests")
}

func (m *stageClientMock) AllFiles(context.Context, string, string) (map[string]filesapi.FileMeta, error) {
	panic("not used in stage tests")
}

func (m *stageClientMock) DownloadFile(context.Context, string, string, string, io.Writer) (string, int64, error) {
	panic("not used in stage tests")
}

func (m *stageClientMock) DeleteFiles(context.Context, string, []string) (*filesapi.DeleteFilesResp, error) {
	panic("not used in stage tests")
}

func (m *stageClientMock) ListVersions(context.Context, string, int) ([]filesapi.CatalogVersion, error) {
	panic("not used in stage tests")
}

func stageVersionID(n int) string {
	return fmt.Sprintf("ver-%d", n)
}

func writeStageTestFiles(t *testing.T, names ...string) []LocalFile {
	t.Helper()
	root := t.TempDir()
	files := make([]LocalFile, 0, len(names))
	for _, name := range names {
		abs := filepath.Join(root, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
		require.NoError(t, os.WriteFile(abs, []byte("x"), 0o644))
		files = append(files, LocalFile{RelPath: name, AbsPath: abs, Size: 1})
	}
	return files
}

func TestUploadFilesParallel_Empty(t *testing.T) {
	t.Parallel()

	mock := &stageClientMock{}
	err := uploadFilesParallel(context.Background(), mock, "cat", "stage", nil)
	require.NoError(t, err)
	assert.Empty(t, mock.uploadPaths)
}

func TestUploadFilesParallel_AllSucceed(t *testing.T) {
	t.Parallel()

	files := writeStageTestFiles(t, "a.txt", "b.txt", "c.txt")
	mock := &stageClientMock{}

	err := uploadFilesParallel(context.Background(), mock, "cat", "stage", files)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"a.txt", "b.txt", "c.txt"}, mock.uploadPaths)
}

func TestUploadFilesParallel_OneUploadFails(t *testing.T) {
	t.Parallel()

	files := writeStageTestFiles(t, "ok.txt", "bad.txt", "also-ok.txt")
	wantErr := errors.New("upload failed")
	mock := &stageClientMock{
		uploadErrForPath: map[string]error{"bad.txt": wantErr},
	}

	err := uploadFilesParallel(context.Background(), mock, "cat", "stage", files)
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
	assert.Contains(t, err.Error(), "bad.txt")
}

func TestUploadFilesParallel_ContextCancelled(t *testing.T) {
	t.Parallel()

	files := writeStageTestFiles(t, "slow-1.txt", "slow-2.txt", "slow-3.txt", "slow-4.txt", "slow-5.txt")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mock := &stageClientMock{
		uploadHook: func(string) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		},
	}

	err := uploadFilesParallel(ctx, mock, "cat", "stage", files)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestUploadFilesParallel_RespectsConcurrencyLimit(t *testing.T) {
	t.Parallel()

	names := make([]string, UploadConcurrency+2)
	for i := range names {
		names[i] = fmt.Sprintf("file-%02d.txt", i)
	}
	files := writeStageTestFiles(t, names...)

	started := make(chan struct{}, len(files))
	release := make(chan struct{})
	mock := &stageClientMock{
		uploadHook: func(string) error {
			started <- struct{}{}
			<-release
			return nil
		},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- uploadFilesParallel(context.Background(), mock, "cat", "stage", files)
	}()

	for range UploadConcurrency {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for concurrent uploads to start")
		}
	}

	select {
	case <-started:
		t.Fatalf("expected at most %d concurrent uploads", UploadConcurrency)
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	require.NoError(t, <-errCh)
	assert.Len(t, mock.uploadPaths, len(files))
}

func TestStageUploader_CreatesCatalogAndAppliesStage(t *testing.T) {
	t.Parallel()

	files := writeStageTestFiles(t, "only.txt")
	mock := &stageClientMock{}
	up := stageUploader{}

	catalogID, versionID, err := up.upload(context.Background(), mock, "", filesapi.OverwriteReplace, files)
	require.NoError(t, err)

	assert.Equal(t, "cat-test", catalogID)
	assert.Equal(t, "ver-2", versionID)
	assert.Equal(t, 1, mock.createCatalogCalls)
	assert.Equal(t, 1, mock.createStageCalls)
	assert.Equal(t, 1, mock.applyStageCalls)
	assert.Equal(t, []string{"only.txt"}, mock.uploadPaths)
}

func TestStageUploader_ExistingCatalogSkipsCreate(t *testing.T) {
	t.Parallel()

	files := writeStageTestFiles(t, "only.txt")
	mock := &stageClientMock{catalogID: "cat-existing"}
	up := stageUploader{}

	catalogID, _, err := up.upload(context.Background(), mock, "cat-existing", filesapi.OverwriteReplace, files)
	require.NoError(t, err)

	assert.Equal(t, "cat-existing", catalogID)
	assert.Zero(t, mock.createCatalogCalls)
	assert.Equal(t, 1, mock.createStageCalls)
}

func TestStageUploader_CreateCatalogFails(t *testing.T) {
	t.Parallel()

	files := writeStageTestFiles(t, "only.txt")
	mock := &stageClientMock{createCatalogErr: errors.New("create catalog failed")}
	up := stageUploader{}

	_, _, err := up.upload(context.Background(), mock, "", filesapi.OverwriteReplace, files)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create catalog")
	assert.Zero(t, mock.createStageCalls)
}

func TestStageUploader_CreateStageFails(t *testing.T) {
	t.Parallel()

	files := writeStageTestFiles(t, "only.txt")
	mock := &stageClientMock{
		catalogID:      "cat-existing",
		createStageErr: errors.New("create stage failed"),
	}
	up := stageUploader{}

	_, _, err := up.upload(context.Background(), mock, "cat-existing", filesapi.OverwriteReplace, files)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create stage")
	assert.Zero(t, mock.applyStageCalls)
}

func TestStageUploader_ParallelUploadFailsBeforeApply(t *testing.T) {
	t.Parallel()

	files := writeStageTestFiles(t, "good.txt", "bad.txt")
	mock := &stageClientMock{
		catalogID:        "cat-existing",
		uploadErrForPath: map[string]error{"bad.txt": errors.New("stage upload failed")},
	}
	up := stageUploader{}

	_, _, err := up.upload(context.Background(), mock, "cat-existing", filesapi.OverwriteReplace, files)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad.txt")
	assert.Zero(t, mock.applyStageCalls)
}

func TestStageUploader_ApplyStageFailsAfterUploads(t *testing.T) {
	t.Parallel()

	files := writeStageTestFiles(t, "a.txt", "b.txt")
	mock := &stageClientMock{
		catalogID:     "cat-existing",
		applyStageErr: errors.New("apply stage failed"),
	}
	up := stageUploader{}

	_, _, err := up.upload(context.Background(), mock, "cat-existing", filesapi.OverwriteReplace, files)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apply stage")
	assert.Equal(t, 1, mock.applyStageCalls)
	assert.ElementsMatch(t, []string{"a.txt", "b.txt"}, mock.uploadPaths)
}
