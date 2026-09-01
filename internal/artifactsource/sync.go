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
		// A push that only removes files has no upload to report the resulting
		// version, so when the delete response does not name one it has to be
		// looked up. Returning the empty string would put it straight into the
		// artifact's code_ref, leaving the build pointing at no version and
		// unpinning the base every later push diffs against.
		if versionID == "" {
			resolved, err := latestCatalogVersion(ctx, client, catalogID)
			if err != nil {
				return "", "", err
			}
			versionID = resolved
		}
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

// latestCatalogVersion names the newest version of a catalog. It exists for the
// delete-only push, where the caller has advanced the catalog but has no upload
// response to learn the resulting version from.
func latestCatalogVersion(ctx context.Context, client filesapi.Client, catalogID string) (string, error) {
	versions, err := client.ListVersions(ctx, catalogID, 1)
	if err != nil {
		return "", fmt.Errorf("resolve catalog version after delete: %w", err)
	}
	if len(versions) == 0 || versions[0].ID == "" {
		return "", fmt.Errorf("resolve catalog version after delete: catalog %s reports no versions", catalogID)
	}

	return versions[0].ID, nil
}
