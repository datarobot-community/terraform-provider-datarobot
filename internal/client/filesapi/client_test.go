// CLI source: cli/internal/drapi/filesapi/client_test.go
//
// Provider differences from CLI:
//   - Test harness uses newTestClient + testTransport (provider Client + HTTPTransport)
//     instead of CLI's startServer wiring viper/config.GetEndpointURL and filesapi.New().
//   - All client calls pass context.Background(); CLI calls have no context parameter.
//   - Test cases and assertions are otherwise ported from the CLI file.
package filesapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	drclient "github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CLI: startServer(t, handler) + filesapi.New() — wires global viper URL/API key and /version/ stub.
func newTestClient(t *testing.T, handler http.Handler) filesapi.Client {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	cfg := drclient.NewConfiguration("test-token")
	cfg.Endpoint = srv.URL + "/api/v2"

	return filesapi.New(&testTransport{c: drclient.NewClient(cfg)})
}

// Provider-only: implements filesapi.HTTPTransport for tests (mirrors filesapi_adapter.go).
type testTransport struct {
	c *drclient.Client
}

func (t *testTransport) APIEndpoint() string      { return t.c.APIEndpoint() }
func (t *testTransport) APIBaseURL() string       { return t.c.APIBaseURL() }
func (t *testTransport) HTTPClient() *http.Client { return t.c.HTTPClient() }
func (t *testTransport) PrepareAPIRequest(req *http.Request) {
	t.c.PrepareAPIRequest(req)
}

func (t *testTransport) Post(ctx context.Context, path string, body any, result any) error {
	switch out := result.(type) {
	case *filesapi.CatalogResp:
		resp, err := drclient.Post[filesapi.CatalogResp](t.c, ctx, path, body)
		if err != nil {
			return err
		}
		*out = *resp
		return nil
	case *filesapi.StageResp:
		resp, err := drclient.Post[filesapi.StageResp](t.c, ctx, path, body)
		if err != nil {
			return err
		}
		*out = *resp
		return nil
	case *filesapi.ApplyStageResp:
		resp, err := drclient.Post[filesapi.ApplyStageResp](t.c, ctx, path, body)
		if err != nil {
			return err
		}
		*out = *resp
		return nil
	default:
		return fmt.Errorf("unsupported POST result type %T", result)
	}
}

func TestCreateCatalog(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v2/files/", r.URL.Path)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"catalogId":"cid-1","catalogVersionId":"v0"}`))
	}))

	got, err := c.CreateCatalog(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "cid-1", got.CatalogID)
	assert.Equal(t, "v0", got.CatalogVersionID)
}

func TestCreateStage_ApplyStage(t *testing.T) {
	// This test does not call UploadToStage. It only checks the create-stage → apply-stage API wiring (paths, JSON body, response parsing).
	// The full stage workflow is: CreateStage → UploadToStage (one or more files) → ApplyStage. Upload is covered separately in TestUploadToStage_Multipart.
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/files/cid-1/stages/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"catalogId":"cid-1","stageId":"st-1"}`))
	})
	mux.HandleFunc("/api/v2/files/cid-1/fromStage/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)

		var req filesapi.ApplyStageReq
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "st-1", req.StageID)
		assert.Equal(t, filesapi.OverwriteReplace, req.Overwrite)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"catalogId":"cid-1","catalogVersionId":"v1","numFiles":3}`))
	})

	c := newTestClient(t, mux)

	stage, err := c.CreateStage(context.Background(), "cid-1")
	require.NoError(t, err)
	assert.Equal(t, "st-1", stage.StageID)

	apply, err := c.ApplyStage(context.Background(), "cid-1", "st-1", filesapi.OverwriteReplace)
	require.NoError(t, err)
	assert.Equal(t, "v1", apply.CatalogVersionID)
	assert.Equal(t, 3, apply.NumFiles)
}

func TestUploadToStage_Multipart(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/files/cid-1/stages/st-1/upload/", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		mr, err := r.MultipartReader()
		if !assert.NoError(t, err) {
			return
		}

		part, err := mr.NextPart()
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, "file", part.FormName())
		assert.Equal(t, "agent.py", part.FileName())

		body, err := io.ReadAll(part)
		assert.NoError(t, err)
		assert.Equal(t, "print('hi')\n", string(body))

		_, err = mr.NextPart()
		assert.ErrorIs(t, err, io.EOF)

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"catalogId":"cid-1","stageId":"st-1"}`))
	}))

	body := strings.NewReader("print('hi')\n")
	err := c.UploadToStage(context.Background(), "cid-1", "st-1", "agent.py", int64(body.Len()), body)
	require.NoError(t, err)
}

func TestUploadToStage_AdvertisesContentLength(t *testing.T) {
	const payload = "hello-world"

	var (
		gotContentLength int64
		gotBodyLen       int
	)

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentLength = r.ContentLength

		raw, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		gotBodyLen = len(raw)

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"catalogId":"cid-1","stageId":"st-1"}`))
	}))

	body := strings.NewReader(payload)
	require.NoError(t, c.UploadToStage(context.Background(), "cid-1", "st-1", "test.txt", int64(body.Len()), body))

	assert.Greater(t, gotContentLength, int64(len(payload))) // envelope headers' bytes are counted
	assert.Equal(t, int64(gotBodyLen), gotContentLength)     // the advertised size of body mathes what server gets
}

func TestAllFiles_Pagination(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/files/cid-1/versions/v1/allFiles/", func(w http.ResponseWriter, r *http.Request) {
		offset := r.URL.Query().Get("offset")
		w.Header().Set("Content-Type", "application/json")

		if offset == "" {
			page1 := filesapi.AllFilesResp{
				Data: []filesapi.AllFilesItem{
					{FileName: "a.py", FileSize: 10, FileChecksum: "aaa"},
					{FileName: "b.py", FileSize: 20, FileChecksum: "bbb"},
				},
			}
			page1.Next = "http://" + r.Host + r.URL.Path + "?offset=1"
			assert.NoError(t, json.NewEncoder(w).Encode(page1))
			return
		}

		page2 := filesapi.AllFilesResp{
			Data: []filesapi.AllFilesItem{
				{FileName: "café.py", FileSize: 30, FileChecksum: "ccc"},
			},
		}
		assert.NoError(t, json.NewEncoder(w).Encode(page2))
	})

	c := newTestClient(t, mux)

	got, err := c.AllFiles(context.Background(), "cid-1", "v1")
	require.NoError(t, err)
	assert.Len(t, got, 3)
	assert.Equal(t, filesapi.FileMeta{Hash: "aaa", Size: 10}, got["a.py"])
	assert.Equal(t, filesapi.FileMeta{Hash: "ccc", Size: 30}, got["café.py"])
}

func TestAllFiles_RejectsCrossHostNext(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/files/cid-1/versions/v1/allFiles/", func(w http.ResponseWriter, _ *http.Request) {
		page := filesapi.AllFilesResp{
			Data: []filesapi.AllFilesItem{
				{FileName: "ok.py", FileSize: 1, FileChecksum: "aa"},
			},
		}
		page.Next = "https://attacker.example/api/v2/files/cid-1/versions/v1/allFiles/?offset=1"
		assert.NoError(t, json.NewEncoder(w).Encode(page))
	})

	c := newTestClient(t, mux)

	got, err := c.AllFiles(context.Background(), "cid-1", "v1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "host")
	assert.Nil(t, got)
}

func TestAllFiles_RejectsHostilePath(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/files/cid-1/versions/v1/allFiles/", func(w http.ResponseWriter, _ *http.Request) {
		page := filesapi.AllFilesResp{
			Data: []filesapi.AllFilesItem{
				{FileName: "ok.py", FileSize: 1, FileChecksum: "aa"},
				{FileName: "../../etc/passwd", FileSize: 1, FileChecksum: "bb"},
			},
		}
		assert.NoError(t, json.NewEncoder(w).Encode(page))
	})

	c := newTestClient(t, mux)

	got, err := c.AllFiles(context.Background(), "cid-1", "v1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "escapes project root")
	assert.Nil(t, got)
}

func TestDownloadFile_RejectsHostilePath(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("DownloadFile should reject hostile paths before reaching the server")
	}))

	cases := []struct {
		name    string
		path    string
		wantSub string
	}{
		{"DotDotEscape", "../escape", "escapes project root"},
		{"BackslashTraversal", `..\escape`, "backslash"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			_, _, err := c.DownloadFile(context.Background(), "cid-1", "v1", tc.path, &buf)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantSub)
		})
	}
}

func TestDeleteFiles(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v2/files/cid-1/allFiles/", r.URL.Path)

		var req filesapi.DeleteFilesReq
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, []string{"old.py"}, req.Paths)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"catalogId":"cid-1","catalogVersionId":"v2","numFiles":1,"results":[{"path":"old.py","numFilesDeleted":1}]}`))
	}))

	got, err := c.DeleteFiles(context.Background(), "cid-1", []string{"old.py"})
	require.NoError(t, err)
	assert.Equal(t, "v2", got.CatalogVersionID)
	assert.Equal(t, 1, got.NumFiles)
}

func TestDeleteFiles_EmptyIsNoop(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("DeleteFiles with empty paths should not call the server")
	}))

	got, err := c.DeleteFiles(context.Background(), "cid-1", nil)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestPollStatus_RunningJSON(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/status/sid-1/", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"RUNNING_TO_WORKERS","statusId":"sid-1"}`))
	}))

	resp, err := c.PollStatus(context.Background(), "sid-1")
	require.NoError(t, err)
	assert.Equal(t, filesapi.StatusRunningToWorkers, resp.Status)
	assert.False(t, filesapi.IsTerminalStatus(resp.Status))
}

func TestPollStatus_CompletedRedirect(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/status/sid-2/", r.URL.Path)
		w.Header().Set("Location", "/api/v2/catalogItems/cat-x/")
		w.WriteHeader(http.StatusSeeOther)
	}))

	resp, err := c.PollStatus(context.Background(), "sid-2")
	require.NoError(t, err)
	assert.Equal(t, filesapi.StatusCompleted, resp.Status)
	assert.True(t, filesapi.IsTerminalStatus(resp.Status))
}

func TestUploadFromZipExisting(t *testing.T) {
	// updates an existing catalog uploading zip
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/files/cid-1/fromFile/", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("useArchiveContents"))
		assert.Equal(t, "REPLACE", r.URL.Query().Get("overwrite"))
		assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")

		mr, err := r.MultipartReader()
		if !assert.NoError(t, err) {
			return
		}

		part, err := mr.NextPart()
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, "file", part.FormName())
		assert.Equal(t, "changes.zip", part.FileName())

		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"catalogId":"cid-1","catalogVersionId":"v9","statusId":"sid-9"}`))
	}))

	zipBody := bytes.NewReader([]byte("PK\x03\x04fake-zip"))
	resp, err := c.UploadFromZipExisting(context.Background(), "cid-1", "changes.zip", "", int64(zipBody.Len()), zipBody)
	require.NoError(t, err)
	assert.Equal(t, "v9", resp.CatalogVersionID)
	assert.Equal(t, "sid-9", resp.StatusID)
}

func TestUploadFromZipNew_HitsFromFileEndpoint(t *testing.T) {
	// creates a new catalog + uploads zip
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/files/fromFile/", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("useArchiveContents"))
		assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")

		mr, err := r.MultipartReader()
		if !assert.NoError(t, err) {
			return
		}

		part, err := mr.NextPart()
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, "file", part.FormName())
		assert.Equal(t, "wapi-sync.zip", part.FileName())

		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"catalogId":"new-cid","catalogVersionId":"new-ver","statusId":"sid-new"}`))
	}))

	zipBody := bytes.NewReader([]byte("PK\x03\x04fake-zip"))
	resp, err := c.UploadFromZipNew(context.Background(), "wapi-sync.zip", int64(zipBody.Len()), zipBody)
	require.NoError(t, err)
	assert.Equal(t, "new-cid", resp.CatalogID)
	assert.Equal(t, "new-ver", resp.CatalogVersionID)
	assert.Equal(t, "sid-new", resp.StatusID)
}

var _ = multipart.ErrMessageTooLarge
