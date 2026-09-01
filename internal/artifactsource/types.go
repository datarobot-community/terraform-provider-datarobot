package artifactsource

// IgnoreFunc reports whether a normalized relative path should be excluded.
// Directories are queried so subtrees can be pruned at the root.
type IgnoreFunc func(relPath string, isDir bool) bool

// LocalFile is a regular file discovered under the push root.
type LocalFile struct {
	RelPath string
	AbsPath string
	Size    int64
	Hash    string // SHA-256 hex of file contents
}

// Options configure a PushDirectory call.
type Options struct {
	// Dir is the local directory to upload (absolute or relative).
	Dir string
	// CatalogID is an optional existing catalog; empty creates a new catalog on first push.
	CatalogID string
	// CatalogVersionID is the catalog version from the last successful push.
	// When set together with CatalogID, that version is read back and used as
	// the diff base, so only added and modified files are uploaded and files
	// gone from the tree are removed remotely. When the tree is unchanged, no
	// API upload runs at all.
	CatalogVersionID string
	// Overwrite is the Files API overwrite mode; defaults to REPLACE when empty.
	Overwrite string
	// Ignore optionally excludes paths; nil uploads all regular files.
	Ignore IgnoreFunc
}

// Result is returned after a successful PushDirectory.
type Result struct {
	CatalogID        string
	CatalogVersionID string
	SourceHash       string
	FileHashes       Manifest
	FileCount        int
	TotalBytes       int64
	// Incremental is true when the push diffed against a resolved base and sent
	// only what changed, rather than uploading the full tree.
	Incremental bool
	// BaseUnavailable is set when a pinned catalog version could not be read
	// back to diff against. The push still succeeded, as a full upload, but it
	// could not tell which paths had gone away, so anything deleted locally
	// since that version is still in the catalog and will reach the next image
	// build. Uploads are unaffected.
	BaseUnavailable error
}
