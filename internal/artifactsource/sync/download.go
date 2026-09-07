package sync

// CLI source: cli/internal/workload/sync/download.go
//
// Provider differences from CLI:
//   - context.Context is threaded into DownloadFile and honored while a
//     worker waits for a concurrency slot, so a cancelled apply stops
//     pulling files (the CLI client has no ctx).
//   - No panic-recovery wrapper (CLI recoverWorkerPanic logs through
//     cli/internal/log): a panic here surfaces like any other provider
//     panic instead of being converted to an error.
//   - The worker pool is the same semaphore/done/errCh shape already used
//     by internal/artifactsource/stage.go, itself a port of this file.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
)

// downloadFiles pulls files in parallel, at most DownloadConcurrency at a
// time. Each download streams to disk and hashes in the same pass, so
// verification costs no extra read. The first error stops workers that
// have not started yet; the ones already running finish and are discarded.
func (e *Engine) downloadFiles(ctx context.Context, catalogID, versionID string, files []FileAction) error {
	if len(files) == 0 {
		return nil
	}

	done := make(chan struct{})

	var cancelOnce sync.Once

	cancel := func() { cancelOnce.Do(func() { close(done) }) }
	defer cancel()

	sem := make(chan struct{}, DownloadConcurrency)
	errCh := make(chan error, len(files))

	var wg sync.WaitGroup

	for _, fa := range files {
		wg.Add(1)

		go func() {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
			case <-done:
				return
			case <-ctx.Done():
				return
			}

			defer func() { <-sem }()

			if err := e.downloadOne(ctx, catalogID, versionID, fa); err != nil {
				select {
				case errCh <- err:
					cancel()
				default:
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)

	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	default:
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	return nil
}

// downloadOne streams one remote file to disk, hashing as it writes, and
// removes the partial file if the transfer, its size, or its checksum
// does not match what the plan promised. An empty RemoteHash skips
// checksum verification.
func (e *Engine) downloadOne(ctx context.Context, catalogID, versionID string, fa FileAction) error {
	// ExecuteLocal already rejected unsafe server paths up front via
	// validateServerPaths; re-check here so the per-call-site invariant
	// survives future refactors that bypass that entry point.
	if err := filesapi.SafeRelPath(fa.Path); err != nil {
		return fmt.Errorf("server returned unsafe download path %q: %w", fa.Path, err)
	}

	dst := filepath.Join(e.projectDir, filepath.FromSlash(fa.Path))

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("mkdir parent for %s: %w", fa.Path, err)
	}

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create %s: %w", fa.Path, err)
	}

	h := sha256.New()

	_, n, err := e.files.DownloadFile(ctx, catalogID, versionID, fa.Path, io.MultiWriter(out, h))

	closeErr := out.Close()

	if err != nil {
		_ = os.Remove(dst)
		return fmt.Errorf("download %s: %w", fa.Path, err)
	}

	if closeErr != nil {
		_ = os.Remove(dst)
		return fmt.Errorf("close %s: %w", fa.Path, closeErr)
	}

	if fa.RemoteSize > 0 && n != fa.RemoteSize {
		_ = os.Remove(dst)
		return fmt.Errorf("download size mismatch on %s: expected %d, got %d", fa.Path, fa.RemoteSize, n)
	}

	if fa.RemoteHash != "" {
		if got := hex.EncodeToString(h.Sum(nil)); got != fa.RemoteHash {
			_ = os.Remove(dst)
			return fmt.Errorf("checksum mismatch on %s: expected %s, got %s", fa.Path, fa.RemoteHash, got)
		}
	}

	return nil
}
