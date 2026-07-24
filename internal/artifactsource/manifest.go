package artifactsource

// FileMeta is the per-file manifest entry stored in Terraform state.
// Hash is SHA-256 hex (64 chars) as produced by hashFile.
type FileMeta struct {
	Hash string
	Size int64
}

// Manifest maps normalized relative paths to per-file metadata.
type Manifest map[string]FileMeta

func manifestFromFiles(files []LocalFile) Manifest {
	out := make(Manifest, len(files))
	for _, f := range files {
		out[f.RelPath] = FileMeta{Hash: f.Hash, Size: f.Size}
	}
	return out
}

func localFilesByPath(files []LocalFile) map[string]LocalFile {
	out := make(map[string]LocalFile, len(files))
	for _, f := range files {
		out[f.RelPath] = f
	}
	return out
}

func localFilesForPaths(byPath map[string]LocalFile, paths []string) []LocalFile {
	out := make([]LocalFile, 0, len(paths))
	for _, path := range paths {
		if f, ok := byPath[path]; ok {
			out = append(out, f)
		}
	}
	return out
}
