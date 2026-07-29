package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
)

type filesapiTransport struct {
	c *Client
}

func newFilesAPITransport(c *Client) filesapi.HTTPTransport {
	return &filesapiTransport{c: c}
}

func (t *filesapiTransport) APIEndpoint() string {
	return t.c.APIEndpoint()
}

func (t *filesapiTransport) APIBaseURL() string {
	return t.c.APIBaseURL()
}

func (t *filesapiTransport) HTTPClient() *http.Client {
	return t.c.HTTPClient()
}

func (t *filesapiTransport) PrepareAPIRequest(req *http.Request) {
	t.c.PrepareAPIRequest(req)
}

func (t *filesapiTransport) Post(ctx context.Context, path string, body any, result any) error {
	switch out := result.(type) {
	case *filesapi.CatalogResp:
		resp, err := Post[filesapi.CatalogResp](t.c, ctx, path, body)
		if err != nil {
			return err
		}
		*out = *resp
		return nil
	case *filesapi.StageResp:
		resp, err := Post[filesapi.StageResp](t.c, ctx, path, body)
		if err != nil {
			return err
		}
		*out = *resp
		return nil
	case *filesapi.ApplyStageResp:
		resp, err := Post[filesapi.ApplyStageResp](t.c, ctx, path, body)
		if err != nil {
			return err
		}
		*out = *resp
		return nil
	default:
		return fmt.Errorf("filesapi transport: unsupported POST result type %T", result)
	}
}
