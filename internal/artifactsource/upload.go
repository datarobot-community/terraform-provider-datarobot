package artifactsource

import (
	"context"
	"errors"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
)

// UploadFiles pushes an explicit set of already-walked files into
// catalogID, creating a new catalog when catalogID is empty. Routing is the
// same one PushDirectory uses: stage while the change set is at or below
// StageVsZipFileThreshold files and StageVsZipBytesThreshold bytes, zip
// above it.
//
// Exported for internal/artifactsource/sync's Engine, which uploads the
// Uploads rows of a three-way SyncPlan rather than a whole directory walk
// and must not re-port the stage/zip split. An empty file set is a no-op
// that reports no new catalog version.
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
