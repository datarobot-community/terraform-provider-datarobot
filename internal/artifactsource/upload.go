package artifactsource

import (
	"context"
	"errors"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
)

// UploadFiles pushes an explicit set of already-walked files into
// catalogID, creating a new catalog when catalogID is empty. It routes on
// the size of the change set: stage while it is at or below
// StageVsZipFileThreshold files and StageVsZipBytesThreshold bytes, zip
// above it.
//
// This is the package's only upload entry point, and it uploads exactly
// what it is handed. Which files those are is decided by
// internal/artifactsource/sync's Engine, from the Uploads rows of a
// three-way SyncPlan; paths that plan removes go out separately as Files
// API deletes. An empty file set is a no-op that reports no new catalog
// version.
func UploadFiles(ctx context.Context, client filesapi.Client, catalogID, overwrite string, files []LocalFile) (catalogIDOut, versionID string, err error) {
	if client == nil {
		return "", "", errors.New("files API client is required")
	}

	if len(files) == 0 {
		return catalogID, "", nil
	}

	if overwrite == "" {
		overwrite = filesapi.OverwriteReplace
	}

	return chooseUploader(files).upload(ctx, client, catalogID, overwrite, files)
}
