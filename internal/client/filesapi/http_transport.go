// Provider-only file (no CLI equivalent).
//
// The CLI wires HTTP through the global drapi package (AuthorizeRequest,
// EndpointURL, PostJSON, GetJSON, etc.). The provider injects a transport
// interface instead to share auth/base URL with the Workload API client and
// to avoid an import cycle with internal/client (see filesapi_adapter.go).
package filesapi

import (
	"context"
	"net/http"
)

// HTTPTransport is the minimal HTTP surface the Files API client needs from the
// provider client. Implemented by client.filesapiTransport in the parent package
// to avoid an import cycle.
type HTTPTransport interface {
	APIEndpoint() string
	APIBaseURL() string
	HTTPClient() *http.Client
	PrepareAPIRequest(req *http.Request)
	Post(ctx context.Context, path string, body any, result any) error
}
