package artifactsource

import (
	"context"
	"io"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// uploadClientMock records which of the two upload backends UploadFiles
// routed a change set to. stageClientMock (stage_test.go) covers the stage
// path in depth; this one only has to tell the paths apart.
type uploadClientMock struct {
	stageClientMock

	zipNewCalls      int
	zipExistingCalls int
	pollStatusCalls  int
	zipStatusID      string
}

func (m *uploadClientMock) UploadFromZipNew(context.Context, string, int64, io.Reader) (*filesapi.FromFileResp, error) {
	m.zipNewCalls++
	return &filesapi.FromFileResp{CatalogID: "cat-zip-new", CatalogVersionID: "ver-zip", StatusID: m.zipStatusID}, nil
}

func (m *uploadClientMock) UploadFromZipExisting(_ context.Context, catalogID, _, _ string, _ int64, _ io.Reader) (*filesapi.FromFileResp, error) {
	m.zipExistingCalls++
	return &filesapi.FromFileResp{CatalogID: catalogID, CatalogVersionID: "ver-zip", StatusID: m.zipStatusID}, nil
}

func (m *uploadClientMock) PollStatus(context.Context, string) (*filesapi.StatusResp, error) {
	m.pollStatusCalls++
	return &filesapi.StatusResp{Status: filesapi.StatusCompleted}, nil
}

func TestChooseUploader_RoutesOnFileCountAndBytes(t *testing.T) {
	t.Parallel()

	atThreshold := make([]LocalFile, StageVsZipFileThreshold)
	assert.IsType(t, stageUploader{}, chooseUploader(atThreshold), "the threshold itself still stages")

	overCount := make([]LocalFile, StageVsZipFileThreshold+1)
	assert.IsType(t, zipUploader{}, chooseUploader(overCount), "one file too many switches to zip")

	overBytes := []LocalFile{{RelPath: "big.bin", Size: StageVsZipBytesThreshold + 1}}
	assert.IsType(t, zipUploader{}, chooseUploader(overBytes), "a single oversized file switches to zip")

	assert.IsType(t, stageUploader{}, chooseUploader(nil))
}

func TestUploadFiles_SmallChangeSetTakesTheStagePath(t *testing.T) {
	t.Parallel()

	files := writeStageTestFiles(t, "a.txt", "b.txt")
	mock := &uploadClientMock{}

	catalogID, versionID, err := UploadFiles(context.Background(), mock, "cat-1", "", files)
	require.NoError(t, err)

	assert.Equal(t, "cat-1", catalogID)
	assert.Equal(t, "ver-1", versionID)
	assert.Equal(t, 1, mock.createStageCalls)
	assert.Zero(t, mock.createCatalogCalls, "an existing catalog is reused")
	assert.Zero(t, mock.zipNewCalls)
	assert.Zero(t, mock.zipExistingCalls)
}

func TestUploadFiles_LargeChangeSetTakesTheZipPathAndPolls(t *testing.T) {
	t.Parallel()

	names := make([]string, StageVsZipFileThreshold+1)
	for i := range names {
		names[i] = "file-" + string(rune('a'+i%26)) + string(rune('0'+i/26)) + ".txt"
	}

	files := writeStageTestFiles(t, names...)
	mock := &uploadClientMock{zipStatusID: "status-1"}

	catalogID, versionID, err := UploadFiles(context.Background(), mock, "cat-1", "", files)
	require.NoError(t, err)

	assert.Equal(t, "cat-1", catalogID)
	assert.Equal(t, "ver-zip", versionID)
	assert.Equal(t, 1, mock.zipExistingCalls)
	assert.Equal(t, 1, mock.pollStatusCalls, "an async extract is waited on before the version is reported")
	assert.Zero(t, mock.createStageCalls)
}

func TestUploadFiles_EmptyChangeSetIsANoOp(t *testing.T) {
	t.Parallel()

	mock := &uploadClientMock{}

	catalogID, versionID, err := UploadFiles(context.Background(), mock, "cat-1", "", nil)
	require.NoError(t, err)

	assert.Equal(t, "cat-1", catalogID)
	assert.Empty(t, versionID, "nothing was pushed, so there is no new catalog version")
	assert.Zero(t, mock.createStageCalls)
	assert.Zero(t, mock.zipExistingCalls)
}

func TestUploadFiles_NilClient(t *testing.T) {
	t.Parallel()

	_, _, err := UploadFiles(context.Background(), nil, "cat-1", "", writeStageTestFiles(t, "a.txt"))
	assert.ErrorContains(t, err, "files API client is required")
}
