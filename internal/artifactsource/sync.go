package artifactsource

import (
	"context"
	"fmt"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
)

func applyIncrementalPush(
	ctx context.Context,
	client filesapi.Client,
	catalogID, overwrite string,
	plan *PushPlan,
	byPath map[string]LocalFile,
) (string, string, error) {
	versionID := ""

	if len(plan.Deletes) > 0 {
		resp, err := client.DeleteFiles(ctx, catalogID, plan.Deletes)
		if err != nil {
			return "", "", fmt.Errorf("delete remote files: %w", err)
		}
		if resp != nil && resp.CatalogVersionID != "" {
			versionID = resp.CatalogVersionID
		}
	}

	if len(plan.Uploads) == 0 {
		return catalogID, versionID, nil
	}

	uploadFiles := localFilesForPaths(byPath, plan.Uploads)
	up := chooseUploader(uploadFiles)

	catalogIDOut, versionIDOut, err := up.upload(ctx, client, catalogID, overwrite, uploadFiles)
	if err != nil {
		return "", "", err
	}

	return catalogIDOut, versionIDOut, nil
}
