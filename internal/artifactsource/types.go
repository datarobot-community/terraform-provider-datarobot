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
