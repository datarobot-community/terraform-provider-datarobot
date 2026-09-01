package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-querystring/query"
)

func TestOtelLogStreamStateEmitNew(t *testing.T) {
	state := newOtelLogStreamState()

	var lines []string
	emit := func(entry OtelLogEntry) {
		lines = append(lines, entry.Message)
	}

	// Server order is newest first.
	state.emitNew([]OtelLogEntry{
		{Timestamp: "t2", Level: "info", Message: "second"},
		{Timestamp: "t1", Level: "info", Message: "first"},
	}, emit)

	if len(lines) != 2 || lines[0] != "first" || lines[1] != "second" {
		t.Fatalf("expected chronological emission, got %#v", lines)
	}

	state.emitNew([]OtelLogEntry{
		{Timestamp: "t2", Level: "info", Message: "second"},
		{Timestamp: "t3", Level: "info", Message: "third"},
	}, emit)

	if len(lines) != 3 || lines[2] != "third" {
		t.Fatalf("expected only new line appended, got %#v", lines)
	}
}

// A burst of more new records than fit in one page must not be silently
// dropped: pollNewOtelEntityLogs should follow Next until it either catches up
// with a previously-seen record or runs out of pages.
func TestPollNewOtelEntityLogsFollowsPaginationForBurst(t *testing.T) {
	var page2URL string
	var requestPaths []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPaths = append(requestPaths, r.URL.String())
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("cursor") == "2" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"timestamp": "t2", "level": "info", "message": "m2"},
					{"timestamp": "t1", "level": "info", "message": "m1"},
				},
				"next": "",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"timestamp": "t4", "level": "info", "message": "m4"},
				{"timestamp": "t3", "level": "info", "message": "m3"},
			},
			"next": page2URL,
		})
	}))
	defer server.Close()
	page2URL = server.URL + "/otel/artifact/art-1/logs/?limit=2&cursor=2"

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc, ok := NewService(NewClient(cfg)).(*ServiceImpl)
	if !ok {
		t.Fatal("NewService did not return *ServiceImpl")
	}

	state := newOtelLogStreamState()
	var got []string
	svc.pollNewOtelEntityLogs(context.Background(), "artifact", "art-1", "", state, func(e OtelLogEntry) {
		got = append(got, e.Message)
	})

	want := []string{"m1", "m2", "m3", "m4"}
	if len(got) != len(want) {
		t.Fatalf("expected %d entries across 2 pages, got %#v", len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("entry %d = %q, want %q (got %#v)", i, got[i], w, got)
		}
	}
	if len(requestPaths) != 2 {
		t.Fatalf("expected 2 page fetches, got %d: %v", len(requestPaths), requestPaths)
	}

	// A follow-up poll with no new records should stop after the first page
	// instead of walking pagination all over again.
	requestPaths = nil
	got = nil
	svc.pollNewOtelEntityLogs(context.Background(), "artifact", "art-1", "", state, func(e OtelLogEntry) {
		got = append(got, e.Message)
	})
	if len(got) != 0 {
		t.Fatalf("expected no new entries on repeat poll, got %#v", got)
	}
	if len(requestPaths) != 1 {
		t.Fatalf("expected repeat poll to stop after 1 page once caught up, got %d: %v", len(requestPaths), requestPaths)
	}
}

// A burst deeper than maxOtelLogStreamPages cannot be fetched, but it must not pass for
// a complete stream: the poll emits a gap notice where the dropped records would have been.
func TestPollNewOtelEntityLogsWarnsWhenPageCapIsHit(t *testing.T) {
	var serverURL string
	pages := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages++
		w.Header().Set("Content-Type", "application/json")
		// Always a fresh record and always another page, so the walk can only stop on the cap.
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"timestamp": fmt.Sprintf("t%d", pages), "level": "info", "message": fmt.Sprintf("m%d", pages)},
			},
			"next": serverURL + fmt.Sprintf("/otel/artifact/art-1/logs/?limit=100&cursor=%d", pages+1),
		})
	}))
	defer server.Close()
	serverURL = server.URL

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc, ok := NewService(NewClient(cfg)).(*ServiceImpl)
	if !ok {
		t.Fatal("NewService did not return *ServiceImpl")
	}

	var got []OtelLogEntry
	svc.pollNewOtelEntityLogs(
		context.Background(), "artifact", "art-1", "build-1", newOtelLogStreamState(),
		func(e OtelLogEntry) { got = append(got, e) },
	)

	if pages != maxOtelLogStreamPages {
		t.Fatalf("expected the walk to stop at %d pages, got %d", maxOtelLogStreamPages, pages)
	}
	if len(got) != maxOtelLogStreamPages+1 {
		t.Fatalf("expected %d records plus 1 gap notice, got %d", maxOtelLogStreamPages, len(got))
	}
	// Oldest-first: the notice precedes the records it was appended ahead of.
	notice := got[0]
	if notice.Level != "warn" || !strings.Contains(notice.Message, "fell behind") {
		t.Fatalf("expected a gap notice first, got %+v", notice)
	}
	if notice.Timestamp == "" {
		t.Error("gap notice needs a timestamp to render and dedupe like a real record")
	}
	for _, entry := range got[1:] {
		if entry.Level == "warn" {
			t.Errorf("expected exactly one gap notice, also got %+v", entry)
		}
	}
}

func TestOtelLogsRequestIncludesBuildIDFilter(t *testing.T) {
	values, err := query.Values(otelLogsRequest(100, "build-abc"))
	if err != nil {
		t.Fatalf("query.Values returned error: %v", err)
	}
	if got := values["searchKeys"]; len(got) != 1 || got[0] != "build_id" {
		t.Fatalf("expected searchKeys=build_id, got %v", got)
	}
	if got := values["searchValues"]; len(got) != 1 || got[0] != "build-abc" {
		t.Fatalf("expected searchValues=build-abc, got %v", got)
	}

	empty, err := query.Values(otelLogsRequest(100, ""))
	if err != nil {
		t.Fatalf("query.Values returned error: %v", err)
	}
	if len(empty["searchKeys"]) != 0 || len(empty["searchValues"]) != 0 {
		t.Fatalf("expected no search filters without build ID, got %v", empty)
	}
}

func TestFormatOtelLogEntryIncludesStackTrace(t *testing.T) {
	got := FormatOtelLogEntry(OtelLogEntry{
		Timestamp:  "2026-07-09T16:14:50Z",
		Level:      "error",
		Message:    "boom",
		StackTrace: "traceback",
	})
	if got != "[2026-07-09T16:14:50Z] ERROR: boom\ntraceback" {
		t.Fatalf("unexpected format: %q", got)
	}
}
