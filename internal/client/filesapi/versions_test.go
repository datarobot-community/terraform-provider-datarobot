// CLI source: cli/internal/drapi/filesapi/versions_test.go
//
// Provider differences from CLI:
// - Uses newTestClient from client_test.go instead of CLI's startServer + filesapi.New().
// - ListVersions calls pass context.Background(); CLI has no context parameter.
// - Test cases and assertions are otherwise ported from the CLI file.
// - Provider-only tests at the end cover the page size and the ErrNotFound/ErrGone mapping.
package filesapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client/filesapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListVersions_SinglePage(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/files/cid-1/versions/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "-created", r.URL.Query().Get("orderBy"))

		page := filesapi.CatalogVersionsResp{
			Data: []filesapi.CatalogVersion{
				{ID: "v3", CreatedAt: "2026-04-10T14:30:00Z", NumFiles: 47, TotalSize: 2412544},
				{ID: "v2", CreatedAt: "2026-04-10T10:15:00Z", NumFiles: 46, TotalSize: 2300000},
				{ID: "v1", CreatedAt: "2026-04-09T16:45:00Z", NumFiles: 45, TotalSize: 2100000},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(page))
	})

	c := newTestClient(t, mux)

	got, err := c.ListVersions(context.Background(), "cid-1", 0)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	assert.Equal(t, "v3", got[0].ID)
	assert.Equal(t, 47, got[0].NumFiles)
	assert.Equal(t, int64(2412544), got[0].TotalSize)
}

func TestListVersions_Pagination(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/files/cid-1/versions/", func(w http.ResponseWriter, r *http.Request) {
		offset := r.URL.Query().Get("offset")
		w.Header().Set("Content-Type", "application/json")

		if offset == "" {
			page1 := filesapi.CatalogVersionsResp{
				Data: []filesapi.CatalogVersion{
					{ID: "v5", CreatedAt: "2026-04-10T14:30:00Z", NumFiles: 5, TotalSize: 500},
					{ID: "v4", CreatedAt: "2026-04-10T13:30:00Z", NumFiles: 4, TotalSize: 400},
				},
			}
			page1.Next = "http://" + r.Host + r.URL.Path + "?offset=2"
			assert.NoError(t, json.NewEncoder(w).Encode(page1))
			return
		}

		page2 := filesapi.CatalogVersionsResp{
			Data: []filesapi.CatalogVersion{
				{ID: "v3", CreatedAt: "2026-04-10T12:30:00Z", NumFiles: 3, TotalSize: 300},
			},
		}
		assert.NoError(t, json.NewEncoder(w).Encode(page2))
	})

	c := newTestClient(t, mux)

	got, err := c.ListVersions(context.Background(), "cid-1", 0)
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, "v5", got[0].ID)
	assert.Equal(t, "v3", got[2].ID)
}

func TestListVersions_LimitTruncatesMidPage(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/files/cid-1/versions/", func(w http.ResponseWriter, _ *http.Request) {
		page := filesapi.CatalogVersionsResp{
			Data: []filesapi.CatalogVersion{
				{ID: "v5"}, {ID: "v4"}, {ID: "v3"}, {ID: "v2"}, {ID: "v1"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(page))
	})

	c := newTestClient(t, mux)

	got, err := c.ListVersions(context.Background(), "cid-1", 2)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "v5", got[0].ID)
	assert.Equal(t, "v4", got[1].ID)
}

func TestListVersions_RejectsCrossHostNext(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/files/cid-1/versions/", func(w http.ResponseWriter, _ *http.Request) {
		page := filesapi.CatalogVersionsResp{
			Data: []filesapi.CatalogVersion{{ID: "v1"}},
		}
		page.Next = "https://attacker.example/api/v2/files/cid-1/versions/?offset=1"

		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(page))
	})

	c := newTestClient(t, mux)

	got, err := c.ListVersions(context.Background(), "cid-1", 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "host")
	assert.Nil(t, got)
}

func TestListVersions_DecodesRealServerShape(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/files/cid-1/versions/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"count": 1,
			"totalCount": 1,
			"next": null,
			"previous": null,
			"data": [{
				"catalogId": "cid-1",
				"catalogVersionId": "ver-abc-1",
				"numFiles": 3,
				"creationDate": "2026-04-30T13:54:38.562000Z",
				"isLatest": true,
				"createdBy": "test",
				"isStage": false,
				"size": 615
			}]
		}`))
	})

	c := newTestClient(t, mux)

	got, err := c.ListVersions(context.Background(), "cid-1", 0)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "ver-abc-1", got[0].ID)
	assert.Equal(t, "2026-04-30T13:54:38.562000Z", got[0].CreatedAt)
	assert.Equal(t, 3, got[0].NumFiles)
	assert.Equal(t, int64(615), got[0].TotalSize)
}

func TestListVersions_EmptyResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/files/cid-1/versions/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(filesapi.CatalogVersionsResp{}))
	})

	c := newTestClient(t, mux)

	got, err := c.ListVersions(context.Background(), "cid-1", 0)
	require.NoError(t, err)
	assert.Empty(t, got)
}

// Provider-only: a caller asking for fewer versions than a page gets a page
// of that size, so a one-version read is one version on the server too.
func TestListVersions_PageSizeFollowsLimit(t *testing.T) {
	for _, tc := range []struct {
		name  string
		limit int
		want  string
	}{
		{name: "one version", limit: 1, want: "1"},
		{name: "no limit", limit: 0, want: "100"},
		{name: "more than a page", limit: 500, want: "100"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.URL.Query().Get("limit")
				w.Header().Set("Content-Type", "application/json")
				assert.NoError(t, json.NewEncoder(w).Encode(filesapi.CatalogVersionsResp{
					Data: []filesapi.CatalogVersion{{ID: "v2"}, {ID: "v1"}},
				}))
			}))

			_, err := c.ListVersions(context.Background(), "cid-1", tc.limit)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// Provider-only: the sync engine drops a catalog pin on ErrGone only, so a
// 404, which the Files API also returns for a catalog the caller cannot see,
// must not match it.
func TestListVersions_ErrGoneOnlyFor410(t *testing.T) {
	for _, tc := range []struct {
		status   int
		notFound bool
		gone     bool
	}{
		{http.StatusGone, true, true},
		{http.StatusNotFound, true, false},
		{http.StatusForbidden, false, false},
		{http.StatusInternalServerError, false, false},
	} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(`{"message":"Requested Files cid-1 was previously deleted."}`))
			}))

			got, err := c.ListVersions(context.Background(), "cid-1", 1)
			require.Error(t, err)
			assert.Nil(t, got)
			assert.Equal(t, tc.notFound, errors.Is(err, filesapi.ErrNotFound))
			assert.Equal(t, tc.gone, errors.Is(err, filesapi.ErrGone))
		})
	}
}
