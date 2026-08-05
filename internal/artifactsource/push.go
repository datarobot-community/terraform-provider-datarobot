package artifactsource

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
)

// PushDirectory walks a local directory and uploads files to the DataRobot
// Files API catalog. On first push (no CatalogID or BaseFiles) the full tree
// is uploaded. On subsequent pushes only added, modified, and deleted files
// are synced incrementally on top of the previous catalog version.
func PushDirectory(ctx context.Context, client filesapi.Client, opts Options) (*Result, error) {
	if client == nil {
		return nil, errors.New("files API client is required")
	}
	if opts.Dir == "" {
		return nil, errors.New("directory path is required")
	}

	overwrite := opts.Overwrite
	if overwrite == "" {
		overwrite = filesapi.OverwriteReplace
	}

	files, totalBytes, err := scanLocalFiles(opts.Dir, opts.Ignore)
	if err != nil {
		return nil, err
	}

	fileHashes := manifestFromFiles(files)
	sourceHash := directoryFingerprint(files)
	byPath := localFilesByPath(files)

	result := &Result{
		SourceHash: sourceHash,
		FileHashes: fileHashes,
		FileCount:  len(files),
		TotalBytes: totalBytes,
	}

	if canIncrementalPush(opts) {
		plan := DiffPushOnly(opts.BaseFiles, fileHashes)
		if plan.IsEmpty() {
			result.CatalogID = opts.CatalogID
			result.CatalogVersionID = opts.CatalogVersionID
			return result, nil
		}

		catalogID, versionID, err := applyIncrementalPush(ctx, client, opts.CatalogID, overwrite, plan, byPath)
		if err != nil {
			return nil, err
		}

		result.CatalogID = catalogID
		result.CatalogVersionID = versionID
		result.Incremental = true
		return result, nil
	}

	up := chooseUploader(files)
	catalogID, versionID, err := up.upload(ctx, client, opts.CatalogID, overwrite, files)
	if err != nil {
		return nil, err
	}

	result.CatalogID = catalogID
	result.CatalogVersionID = versionID
	return result, nil
}

func canIncrementalPush(opts Options) bool {
	return opts.CatalogID != "" && len(opts.BaseFiles) > 0
}

func scanLocalFiles(dir string, ignore IgnoreFunc) ([]LocalFile, int64, error) {
	entries, err := walkDirectory(dir, ignore)
	if err != nil {
		return nil, 0, err
	}
	if len(entries) == 0 {
		return nil, 0, fmt.Errorf("directory %s contains no uploadable files", dir)
	}

	files := make([]LocalFile, 0, len(entries))
	var totalBytes int64

	for _, e := range entries {
		if err := filesapi.SafeRelPath(e.RelPath); err != nil {
			return nil, 0, fmt.Errorf("invalid path %q: %w", e.RelPath, err)
		}

		hash, size, err := hashFile(e.AbsPath)
		if err != nil {
			return nil, 0, err
		}

		files = append(files, LocalFile{
			RelPath: e.RelPath,
			AbsPath: e.AbsPath,
			Size:    size,
			Hash:    hash,
		})
		totalBytes += size
	}

	return files, totalBytes, nil
}
