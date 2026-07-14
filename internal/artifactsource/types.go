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
	FileCount        int
	TotalBytes       int64
}
