// CLI source: cli/internal/drapi/filesapi/fromfile.go
//
// Provider differences from CLI:
//   - All methods take context.Context.
//   - URL building uses c.endpointURL instead of drapi.EndpointURL.
//   - uploadZipMultipart takes *httpClient + ctx; CLI version is a package-level helper.
//   - PollStatus inlines getAcceptingRedirect logic; CLI keeps it as a separate function
//     using drapi.AuthorizeRequest instead of transport.PrepareAPIRequest.
package filesapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func (c *httpClient) UploadFromZipNew(ctx context.Context, name string, size int64, body io.Reader) (*FromFileResp, error) {
	q := url.Values{}
	q.Set("useArchiveContents", "true")

	// CLI: drapi.EndpointURL("/files/fromFile/", q)
	requestURL := c.endpointURL("/files/fromFile/", q)
	return uploadZipMultipart(c, ctx, requestURL, name, size, body)
}

func (c *httpClient) UploadFromZipExisting(ctx context.Context, catalogID, name, overwrite string, size int64, body io.Reader) (*FromFileResp, error) {
	if overwrite == "" {
		overwrite = OverwriteReplace
	}

	q := url.Values{}
	q.Set("useArchiveContents", "true")
	q.Set("overwrite", overwrite)

	// CLI: drapi.EndpointURL("/files/"+catalogID+"/fromFile/", q)
	requestURL := c.endpointURL("/files/"+url.PathEscape(catalogID)+"/fromFile/", q)
	return uploadZipMultipart(c, ctx, requestURL, name, size, body)
}

// CLI: uploadZipMultipart(requestURL, name, size, body) — no client/ctx; uses drapi.ErrFromResp.
func uploadZipMultipart(c *httpClient, ctx context.Context, requestURL, name string, size int64, body io.Reader) (*FromFileResp, error) {
	req, err := newStreamingMultipartRequest(c.transport, ctx, requestURL, nil, name, size, body)
	if err != nil {
		return nil, err
	}

	// CLI: &http.Client{Timeout: uploadHTTPTimeout}
	client := c.httpClientWithTimeout(UploadHTTPTimeout)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("zip upload %s: %w", name, err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, errFromResp(resp, requestURL)
	}

	var out FromFileResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode fromFile response: %w", err)
	}

	return &out, nil
}

func (c *httpClient) PollStatus(ctx context.Context, statusID string) (*StatusResp, error) {
	// CLI: drapi.EndpointURL("/status/"+statusID+"/", nil)
	requestURL := c.endpointURL("/status/"+url.PathEscape(statusID)+"/", nil)

	// CLI: getAcceptingRedirect(requestURL) — separate helper using drapi.AuthorizeRequest
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build status request: %w", err)
	}

	c.transport.PrepareAPIRequest(req)

	client := &http.Client{
		Transport: c.transport.HTTPClient().Transport,
		Timeout:   StatusPollHTTPTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("status request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusSeeOther {
		return nil, errFromResp(resp, requestURL)
	}

	if resp.StatusCode == http.StatusSeeOther {
		return &StatusResp{Status: StatusCompleted, StatusID: statusID}, nil
	}

	var statusResp StatusResp
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return nil, fmt.Errorf("decode status response: %w", err)
	}

	return &statusResp, nil
}
