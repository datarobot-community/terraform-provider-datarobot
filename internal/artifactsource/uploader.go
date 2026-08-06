package artifactsource

import (
	"context"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
)

type uploader interface {
	upload(ctx context.Context, client filesapi.Client, catalogID, overwrite string, files []LocalFile) (catalogIDOut, versionID string, err error)
}

func chooseUploader(files []LocalFile) uploader {
	if len(files) == 0 {
		return stageUploader{}
	}

	var totalBytes int64
	for _, f := range files {
		totalBytes += f.Size
	}

	if len(files) <= StageVsZipFileThreshold && totalBytes <= StageVsZipBytesThreshold {
		return stageUploader{}
	}

	return zipUploader{}
}
