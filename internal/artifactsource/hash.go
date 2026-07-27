package artifactsource

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	maxFileSizeBytes   int64 = 5 * 1024 * 1024 * 1024 // 5 GiB
	hashChunkSizeBytes       = 64 * 1024
)

// ErrFileTooLarge is returned when a file exceeds maxFileSizeBytes.
var ErrFileTooLarge = errors.New("file exceeds max size")

// hashFile returns the SHA-256 hex digest and size of a regular file.
func hashFile(path string) (string, int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", 0, fmt.Errorf("stat %s: %w", path, err)
	}

	if info.Size() > maxFileSizeBytes {
		return "", info.Size(), fmt.Errorf("%w: %s (%d bytes)", ErrFileTooLarge, path, info.Size())
	}

	f, err := os.Open(path)
	if err != nil {
		return "", info.Size(), fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	buf := make([]byte, hashChunkSizeBytes)

	if _, err := io.CopyBuffer(h, f, buf); err != nil {
		return "", info.Size(), fmt.Errorf("hash %s: %w", path, err)
	}

	return hex.EncodeToString(h.Sum(nil)), info.Size(), nil
}
