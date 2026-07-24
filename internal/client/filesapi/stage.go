// CLI source: cli/internal/drapi/filesapi/stage.go
//
// Provider differences from CLI:
//   - All methods take context.Context.
//   - CreateCatalog/CreateStage/ApplyStage use transport.Post instead of drapi.PostJSON.
//   - UploadToStage delegates response handling to doMultipartUpload (multipart.go);
//     CLI inlines the http.Client.Do + status check here.
package filesapi

import (
	"context"
	"fmt"
	"io"
	"net/url"
)

func (c *httpClient) CreateCatalog(ctx context.Context) (*CatalogResp, error) {
	var resp CatalogResp
	// CLI: drapi.EndpointURL + drapi.PostJSON(requestURL, "catalog", ...)
	if err := c.transport.Post(ctx, "/files/", struct{}{}, &resp); err != nil {
		return nil, fmt.Errorf("create catalog: %w", err)
	}
	return &resp, nil
}

func (c *httpClient) CreateStage(ctx context.Context, catalogID string) (*StageResp, error) {
	path := "/files/" + url.PathEscape(catalogID) + "/stages/"
	var resp StageResp
	// CLI: drapi.EndpointURL + drapi.PostJSON(requestURL, "stage", ...)
	if err := c.transport.Post(ctx, path, struct{}{}, &resp); err != nil {
		return nil, fmt.Errorf("create stage: %w", err)
	}
	return &resp, nil
}

func (c *httpClient) UploadToStage(ctx context.Context, catalogID, stageID, name string, size int64, body io.Reader) error {
	// CLI: drapi.EndpointURL(...) with separate error return
	requestURL := c.endpointURL(
		"/files/"+url.PathEscape(catalogID)+"/stages/"+url.PathEscape(stageID)+"/upload/",
		nil,
	)

	// CLI: newStreamingMultipartRequest(requestURL, ...) without transport/ctx
	req, err := newStreamingMultipartRequest(c.transport, ctx, requestURL, nil, name, size, body)
	if err != nil {
		return err
	}

	// CLI: inline client.Do + drapi.ErrFromResp here
	return doMultipartUpload(c, req, name)
}

func (c *httpClient) ApplyStage(ctx context.Context, catalogID, stageID, overwrite string) (*ApplyStageResp, error) {
	path := "/files/" + url.PathEscape(catalogID) + "/fromStage/"
	body := ApplyStageReq{StageID: stageID, Overwrite: overwrite}
	var resp ApplyStageResp
	// CLI: drapi.EndpointURL + drapi.PostJSON(requestURL, "apply-stage", ...)
	if err := c.transport.Post(ctx, path, body, &resp); err != nil {
		return nil, fmt.Errorf("apply stage: %w", err)
	}
	return &resp, nil
}
