package artifactsource

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
)

type stageUploader struct{}

func (stageUploader) upload(ctx context.Context, client filesapi.Client, catalogID, overwrite string, files []LocalFile) (string, string, error) {
	if catalogID == "" {
		cat, err := client.CreateCatalog(ctx)
		if err != nil {
			return "", "", fmt.Errorf("create catalog: %w", err)
		}
		catalogID = cat.CatalogID
	}

	stage, err := client.CreateStage(ctx, catalogID)
	if err != nil {
		return "", "", fmt.Errorf("create stage: %w", err)
	}

	if err := uploadFilesParallel(ctx, client, catalogID, stage.StageID, files); err != nil {
		return "", "", err
	}

	apply, err := client.ApplyStage(ctx, catalogID, stage.StageID, overwrite)
	if err != nil {
		return "", "", fmt.Errorf("apply stage: %w", err)
	}

	return catalogID, apply.CatalogVersionID, nil
}

func uploadFilesParallel(ctx context.Context, client filesapi.Client, catalogID, stageID string, files []LocalFile) error {
	if len(files) == 0 {
		return nil
	}

	done := make(chan struct{})
	var cancelOnce sync.Once
	cancel := func() { cancelOnce.Do(func() { close(done) }) }
	defer cancel()

	sem := make(chan struct{}, UploadConcurrency)
	errCh := make(chan error, len(files))

	var wg sync.WaitGroup

	for _, f := range files {
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

			if err := uploadOneToStage(ctx, client, catalogID, stageID, f); err != nil {
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

func uploadOneToStage(ctx context.Context, client filesapi.Client, catalogID, stageID string, f LocalFile) error {
	file, err := os.Open(f.AbsPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", f.RelPath, err)
	}
	defer func() { _ = file.Close() }()

	if err := client.UploadToStage(ctx, catalogID, stageID, f.RelPath, f.Size, file); err != nil {
		return fmt.Errorf("upload %s: %w", f.RelPath, err)
	}

	return nil
}
