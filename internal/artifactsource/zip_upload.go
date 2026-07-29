package artifactsource

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
)

type zipUploader struct{}

func (zipUploader) upload(ctx context.Context, client filesapi.Client, catalogID, overwrite string, files []LocalFile) (string, string, error) {
	zipPath, err := buildZip(files)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = os.Remove(zipPath) }()

	zipFile, err := os.Open(zipPath)
	if err != nil {
		return "", "", fmt.Errorf("open built zip: %w", err)
	}
	defer func() { _ = zipFile.Close() }()

	stat, err := zipFile.Stat()
	if err != nil {
		return "", "", fmt.Errorf("stat built zip: %w", err)
	}

	var resp *filesapi.FromFileResp

	if catalogID != "" {
		resp, err = client.UploadFromZipExisting(ctx, catalogID, "artifact-push.zip", overwrite, stat.Size(), zipFile)
	} else {
		resp, err = client.UploadFromZipNew(ctx, "artifact-push.zip", stat.Size(), zipFile)
	}
	if err != nil {
		return "", "", fmt.Errorf("upload zip: %w", err)
	}

	if resp.StatusID != "" {
		if err := waitForCompletion(ctx, client, resp.StatusID); err != nil {
			return "", "", err
		}
	}

	return resp.CatalogID, resp.CatalogVersionID, nil
}

func waitForCompletion(ctx context.Context, client filesapi.Client, statusID string) error {
	deadline := time.Now().Add(filesapi.ZipPollTimeout)
	ticker := time.NewTicker(filesapi.ZipPollInterval)
	defer ticker.Stop()

	for {
		if time.Now().After(deadline) {
			return errors.New("timeout waiting for archive extract")
		}

		resp, err := client.PollStatus(ctx, statusID)
		if err != nil {
			return fmt.Errorf("poll status %s: %w", statusID, err)
		}

		if filesapi.IsTerminalStatus(resp.Status) {
			if filesapi.IsErrorStatus(resp.Status) {
				return fmt.Errorf("zip extraction failed: %s (%s)", resp.Status, resp.Message)
			}
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
