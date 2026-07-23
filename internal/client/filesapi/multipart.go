// CLI source: cli/internal/drapi/filesapi/multipart.go
//
// Provider differences from CLI:
//   - newStreamingMultipartRequest takes HTTPTransport + context.Context; CLI uses
//     drapi.AuthorizeRequest(req) instead of transport.PrepareAPIRequest(req).
//   - doMultipartUpload is extracted here from stage.go's inline upload logic.
package filesapi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
)

const multipartFormField = "file"

// CLI: newStreamingMultipartRequest(requestURL, query, filename, size, body) — no transport/ctx.
func newStreamingMultipartRequest(
	transport HTTPTransport,
	ctx context.Context,
	requestURL string,
	query url.Values,
	filename string,
	size int64,
	body io.Reader,
) (*http.Request, error) {
	if len(query) > 0 {
		requestURL += "?" + query.Encode()
	}

	contentType, prologue, epilogue, err := multipartFraming(filename)
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()

	go streamMultipartBody(pw, prologue, body, epilogue)

	// CLI: http.NewRequest(http.MethodPost, ...) without context
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, pr)
	if err != nil {
		_ = pr.Close()
		return nil, fmt.Errorf("build multipart request: %w", err)
	}

	if size >= 0 {
		req.ContentLength = int64(len(prologue)) + size + int64(len(epilogue))
	}

	// CLI: drapi.AuthorizeRequest(req)
	transport.PrepareAPIRequest(req)
	req.Header.Set("Content-Type", contentType)

	return req, nil
}

// Identical logic to CLI multipartFraming.
func multipartFraming(filename string) (string, []byte, []byte, error) {
	var head bytes.Buffer

	w := multipart.NewWriter(&head)

	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, multipartFormField, filename))
	hdr.Set("Content-Type", "application/octet-stream")

	if _, err := w.CreatePart(hdr); err != nil {
		return "", nil, nil, fmt.Errorf("create multipart part: %w", err)
	}

	contentType := w.FormDataContentType()
	headEnd := head.Len()

	if err := w.Close(); err != nil {
		return "", nil, nil, fmt.Errorf("close multipart writer: %w", err)
	}

	buf := head.Bytes()

	return contentType, buf[:headEnd], buf[headEnd:], nil
}

// Identical logic to CLI streamMultipartBody.
func streamMultipartBody(pw *io.PipeWriter, prologue []byte, body io.Reader, epilogue []byte) {
	defer pw.Close()

	if _, err := pw.Write(prologue); err != nil {
		_ = pw.CloseWithError(err)
		return
	}

	if _, err := io.Copy(pw, body); err != nil {
		_ = pw.CloseWithError(fmt.Errorf("stream upload body: %w", err))
		return
	}

	if _, err := pw.Write(epilogue); err != nil {
		_ = pw.CloseWithError(err)
	}
}

// CLI: upload response handling was inline in UploadToStage (stage.go).
func doMultipartUpload(c *httpClient, req *http.Request, name string) error {
	client := c.httpClientWithTimeout(UploadHTTPTimeout)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("upload %s: %w", name, err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return errFromResp(resp, req.URL.String())
	}

	return nil
}
