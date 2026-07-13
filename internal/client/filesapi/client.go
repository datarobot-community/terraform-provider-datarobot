// CLI source: cli/internal/drapi/filesapi/client.go
//
// Provider differences from CLI:
//   - Client interface methods take context.Context (CLI has no ctx parameter).
//   - New(transport HTTPTransport) injects HTTP/auth instead of CLI's New() returning
//     a stateless httpClient backed by global drapi.
//   - httpClient holds a transport field; CLI's httpClient is an empty struct.
//   - endpointURL, assertNextOnSameHost, errFromResp, getJSON, deleteJSON, and
//     httpClientWithTimeout live here; CLI delegates equivalents to drapi.EndpointURL,
//     drapi.AssertNextOnSameHost, drapi.ErrFromResp, drapi.GetJSON, drapi.DeleteJSON,
//     and ad-hoc http.Client construction in each call site.
package filesapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client is a context-aware Files API client sharing auth with the provider HTTP client.
type Client interface {
	CreateCatalog(ctx context.Context) (*CatalogResp, error)
	CreateStage(ctx context.Context, catalogID string) (*StageResp, error)
	UploadToStage(ctx context.Context, catalogID, stageID, name string, size int64, body io.Reader) error
	ApplyStage(ctx context.Context, catalogID, stageID, overwrite string) (*ApplyStageResp, error)
	UploadFromZipNew(ctx context.Context, name string, size int64, body io.Reader) (*FromFileResp, error)
	UploadFromZipExisting(ctx context.Context, catalogID, name, overwrite string, size int64, body io.Reader) (*FromFileResp, error)
	PollStatus(ctx context.Context, statusID string) (*StatusResp, error)
	AllFiles(ctx context.Context, catalogID, versionID string) (map[string]FileMeta, error)
	DownloadFile(ctx context.Context, catalogID, versionID, path string, w io.Writer) (string, int64, error)
	DeleteFiles(ctx context.Context, catalogID string, paths []string) (*DeleteFilesResp, error)
	ListVersions(ctx context.Context, catalogID string, limit int) ([]CatalogVersion, error)
}

// New returns a Files API client backed by transport.
func New(transport HTTPTransport) Client {
	if transport == nil {
		panic("transport is required")
	}
	return &httpClient{transport: transport}
}

type httpClient struct {
	transport HTTPTransport
}

// CLI: drapi.EndpointURL(path, query).
func (c *httpClient) endpointURL(path string, query url.Values) string {
	endpoint := c.transport.APIEndpoint()
	if len(endpoint) > 0 && endpoint[len(endpoint)-1] == '/' {
		endpoint = endpoint[:len(endpoint)-1]
	}
	full := endpoint + path
	if len(query) > 0 {
		return full + "?" + query.Encode()
	}
	return full
}

// CLI: drapi.AssertNextOnSameHost(rawNextURL).
func (c *httpClient) assertNextOnSameHost(rawNextURL string) error {
	next, err := url.Parse(rawNextURL)
	if err != nil {
		return fmt.Errorf("pagination: parse Next URL: %w", err)
	}

	base, err := url.Parse(c.transport.APIBaseURL())
	if err != nil {
		return fmt.Errorf("pagination: parse API base URL: %w", err)
	}

	if next.Scheme != base.Scheme || next.Host != base.Host {
		return fmt.Errorf("pagination: Next URL host %q does not match API base host %q", next.Host, base.Host)
	}

	return nil
}

// CLI: drapi.ErrFromResp(resp, requestURL).
func errFromResp(resp *http.Response, requestURL string) error {
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	method := http.MethodGet
	if resp.Request != nil {
		method = resp.Request.Method
	}
	if len(body) > 0 {
		return fmt.Errorf("%s request %s : response %s %s", method, requestURL, resp.Status, string(body))
	}

	return fmt.Errorf("%s request %s : response %s", method, requestURL, resp.Status)
}

// CLI: drapi.GetJSON(pageURL, label, v) — provider uses context-aware GET with transport auth.
func (c *httpClient) getJSON(ctx context.Context, requestURL string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.transport.PrepareAPIRequest(req)

	resp, err := c.httpClientWithTimeout(DownloadHTTPTimeout).Do(req)
	if err != nil {
		return fmt.Errorf("GET request %s failed: %w", requestURL, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errFromResp(resp, requestURL)
	}

	defer func() { _ = resp.Body.Close() }()

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}

// CLI: drapi.DeleteJSON(requestURL, label, body, v) — provider uses context-aware DELETE with transport auth.
func (c *httpClient) deleteJSON(ctx context.Context, requestURL string, body any, v any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, requestURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.transport.PrepareAPIRequest(req)

	resp, err := c.httpClientWithTimeout(UploadHTTPTimeout).Do(req)
	if err != nil {
		return fmt.Errorf("DELETE request %s failed: %w", requestURL, err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusNoContent {
		return errFromResp(resp, requestURL)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNoContent || v == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}

// CLI: each call site builds &http.Client{Timeout: ...}; provider reuses transport.Transport with per-call timeout.
//
//nolint:unparam // UploadHTTPTimeout and DownloadHTTPTimeout share the same duration today.
func (c *httpClient) httpClientWithTimeout(timeout time.Duration) *http.Client {
	base := c.transport.HTTPClient()
	return &http.Client{
		Transport: base.Transport,
		Timeout:   timeout,
	}
}
