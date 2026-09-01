package artifactsource

import (
	"context"
	"errors"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
)

// PushDirectory walks a local directory and uploads files to the DataRobot
// Files API catalog. On first push (no CatalogID or CatalogVersionID) the full
// tree is uploaded. On subsequent pushes only added, modified, and deleted
// files are synced incrementally on top of the previous catalog version.
//
// Uploading is additive: a new catalog version carries the previous version's
// files alongside the ones just sent, and the overwrite mode only decides what
// happens to a path present in both. Removing a file therefore takes an
// explicit delete, which is why a push with no base to diff against leaves the
// catalog holding paths the local tree no longer has. See resolveBaseFiles.
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

	files, totalBytes, err := collectLocalFiles(opts.Dir, opts.Ignore, false)
	if err != nil {
		return nil, err
	}

	fileHashes := manifestFromFiles(files)
	sourceHash := directoryFingerprint(files)

	result := &Result{
		SourceHash: sourceHash,
		FileHashes: fileHashes,
		FileCount:  len(files),
		TotalBytes: totalBytes,
	}

	// resolved is tracked separately from len(base): a version that legitimately
	// holds no comparable file still gives a usable base, and treating it as no
	// base at all would quietly skip the deletes this diff exists to perform.
	base, resolved, err := resolveBaseFiles(ctx, client, opts)
	if err != nil {
		result.BaseUnavailable = err
	}

	if resolved {
		plan := DiffPushOnly(base, fileHashes)
		if plan.IsEmpty() {
			result.CatalogID = opts.CatalogID
			result.CatalogVersionID = opts.CatalogVersionID
			return result, nil
		}

		catalogID, versionID, err := applyIncrementalPush(ctx, client, opts.CatalogID, overwrite, plan, localFilesByPath(files))
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

// resolveBaseFiles reads back the catalog version the caller last pushed, to
// serve as the diff base. The bool reports whether a base was obtained at all,
// which is not the same question as whether it holds any entries.
//
// The base is fetched rather than remembered because the version id is already
// in Terraform state, which makes the server the one copy of that manifest.
// Storing per-file hashes in state as well would double the record, and writing
// them to disk would put sync bookkeeping inside the directory being uploaded.
//
// Reading the recorded version rather than the catalog's latest bounds what the
// diff may remove to the contents of that one version. A file another tool
// added after it is absent from the base, so it is left alone rather than read
// as a local deletion. The bound is only as good as the recorded version: a
// caller that adopts an artifact it did not upload is pointing this at a
// version somebody else wrote, and every path in it that the caller's tree does
// not have is a deletion.
//
// That bound is not enough on its own, because the base decides deletions and
// the local walk decides what is in the tree, and the two do not see the same
// set of paths. Anything the walk cannot produce would otherwise look exactly
// like a file the user removed. So the base is narrowed to what this push would
// itself upload:
//
//   - Ignored paths are dropped. The matcher covers files that were uploaded
//     before a rule existed to exclude them, so without this a first apply after
//     .drignore lands, or after a user adds one pattern, reads every newly
//     excluded path as a deletion and strips it from the catalog.
//   - Entries with no checksum are dropped. Nothing can be compared against
//     them, and the listing is not required to describe only regular files, so a
//     directory or placeholder row would otherwise be handed to DeleteFiles as
//     a path, taking its subtree with it. Dropping such an entry is safe in both
//     directions: a real file behind one is simply uploaded again.
//
// A failure is reported but not fatal. Falling back to a full upload is what
// this function's absence used to do on every push, so a catalog version that
// has expired or a request that failed in transit costs the deletes for one
// apply rather than the whole apply. The next push that changes the tree
// reconciles them, because the version it reads back still holds the leftovers.
func resolveBaseFiles(ctx context.Context, client filesapi.Client, opts Options) (Manifest, bool, error) {
	if opts.CatalogID == "" || opts.CatalogVersionID == "" {
		return nil, false, nil
	}

	remote, err := client.AllFiles(ctx, opts.CatalogID, opts.CatalogVersionID)
	if err != nil {
		return nil, false, fmt.Errorf("read catalog version %s: %w", opts.CatalogVersionID, err)
	}

	base := make(Manifest, len(remote))
	for path, meta := range remote {
		if meta.Hash == "" {
			continue
		}
		if opts.Ignore != nil && opts.Ignore(path, false) {
			continue
		}
		base[path] = FileMeta{Hash: meta.Hash, Size: meta.Size}
	}

	return base, true, nil
}

func collectLocalFiles(dir string, ignore IgnoreFunc, allowEmpty bool) ([]LocalFile, int64, error) {
	entries, err := walkDirectory(dir, ignore)
	if err != nil {
		return nil, 0, err
	}
	if len(entries) == 0 && !allowEmpty {
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
