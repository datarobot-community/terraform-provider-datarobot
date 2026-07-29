// CLI source: cli/internal/drapi/filesapi/allfiles.go
//
// Provider differences from CLI:
//   - All methods take context.Context.
//   - Pagination/GET/DELETE use c.getJSON/c.deleteJSON instead of drapi.GetJSON/drapi.DeleteJSON.
//   - Path normalization uses local SafeRelPath/NormalizePath (paths.go) instead of
//     fileops.SafeRelPath/fileops.NormalizePath from cli/internal/workload/fileops.
//   - allFilesURL is a method on *httpClient using c.endpointURL; CLI uses drapi.EndpointURL.
package filesapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func (c *httpClient) AllFiles(ctx context.Context, catalogID, versionID string) (map[string]FileMeta, error) {
	out := make(map[string]FileMeta)

	pageURL := allFilesURL(c, catalogID, versionID)

	for pageURL != "" {
		var page AllFilesResp
		// CLI: drapi.GetJSON(pageURL, "files", &page)
		if err := c.getJSON(ctx, pageURL, &page); err != nil {
			return nil, err
		}

		for _, item := range page.Data {
			// CLI: fileops.NormalizePath / fileops.SafeRelPath
			key := NormalizePath(item.FileName)
			if err := SafeRelPath(key); err != nil {
				return nil, fmt.Errorf("remote manifest entry %q: %w", item.FileName, err)
			}

			out[key] = FileMeta{Hash: item.FileChecksum, Size: item.FileSize}
		}

		if page.Next == "" {
			break
		}

		// CLI: drapi.AssertNextOnSameHost(page.Next)
		if err := c.assertNextOnSameHost(page.Next); err != nil {
			return nil, err
		}

		pageURL = page.Next
	}

	return out, nil
}

// CLI: allFilesURL(catalogID, versionID string) using drapi.EndpointURL.
func allFilesURL(c *httpClient, catalogID, versionID string) string {
	if versionID != "" {
		return c.endpointURL(
			"/files/"+url.PathEscape(catalogID)+"/versions/"+url.PathEscape(versionID)+"/allFiles/",
			nil,
		)
	}

	return c.endpointURL("/files/"+url.PathEscape(catalogID)+"/allFiles/", nil)
}

func (c *httpClient) DownloadFile(ctx context.Context, catalogID, versionID, path string, w io.Writer) (string, int64, error) {
	// CLI: fileops.SafeRelPath(path)
	if err := SafeRelPath(path); err != nil {
		return "", 0, fmt.Errorf("download path %q: %w", path, err)
	}

	q := url.Values{}
	q.Set("fileName", path)

	// CLI: drapi.EndpointURL(..., q)
	requestURL := c.endpointURL(
		"/files/"+url.PathEscape(catalogID)+"/versions/"+url.PathEscape(versionID)+"/file/",
		q,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return "", 0, fmt.Errorf("build download request: %w", err)
	}

	c.transport.PrepareAPIRequest(req)

	// CLI: drapi.Get(requestURL, "file", int(downloadHTTPTimeout.Seconds()))
	resp, err := c.httpClientWithTimeout(DownloadHTTPTimeout).Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("download %s: %w", path, err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", 0, errFromResp(resp, requestURL)
	}

	n, err := io.Copy(w, resp.Body)
	if err != nil {
		return "", n, fmt.Errorf("write %s: %w", path, err)
	}

	return "", n, nil
}

func (c *httpClient) DeleteFiles(ctx context.Context, catalogID string, paths []string) (*DeleteFilesResp, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	// CLI: drapi.EndpointURL("/files/"+catalogID+"/allFiles/", nil)
	requestURL := c.endpointURL("/files/"+url.PathEscape(catalogID)+"/allFiles/", nil)

	var resp DeleteFilesResp
	// CLI: drapi.DeleteJSON(requestURL, "files", DeleteFilesReq{Paths: paths}, &resp)
	if err := c.deleteJSON(ctx, requestURL, DeleteFilesReq{Paths: paths}, &resp); err != nil {
		return nil, fmt.Errorf("delete files: %w", err)
	}

	return &resp, nil
}
