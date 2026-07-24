package artifactsource

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
)

// buildZip writes a zip archive of files to a temp file on disk.
func buildZip(files []LocalFile) (string, error) {
	tmp, err := os.CreateTemp("", "artifact-push-*.zip")
	if err != nil {
		return "", fmt.Errorf("create zip tempfile: %w", err)
	}

	defer func() { _ = tmp.Close() }()

	zw := zip.NewWriter(tmp)

	for _, f := range files {
		if err := addToZip(zw, f.AbsPath, f.RelPath); err != nil {
			_ = os.Remove(tmp.Name())
			return "", err
		}
	}

	if err := zw.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return "", fmt.Errorf("close zip writer: %w", err)
	}

	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return "", fmt.Errorf("close zip tempfile: %w", err)
	}

	return tmp.Name(), nil
}

func addToZip(zw *zip.Writer, src, archivePath string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s for zip: %w", src, err)
	}
	defer func() { _ = in.Close() }()

	hdr := &zip.FileHeader{Name: archivePath, Method: zip.Deflate}

	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return fmt.Errorf("zip header for %s: %w", archivePath, err)
	}

	if _, err := io.Copy(w, in); err != nil {
		return fmt.Errorf("copy %s into zip: %w", archivePath, err)
	}

	return nil
}
