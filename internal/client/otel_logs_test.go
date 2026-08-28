package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
	svc.pollNewOtelEntityLogs(context.Background(), "artifact", "art-1", state, func(e OtelLogEntry) {
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
	svc.pollNewOtelEntityLogs(context.Background(), "artifact", "art-1", state, func(e OtelLogEntry) {
		got = append(got, e.Message)
	})
	if len(got) != 0 {
		t.Fatalf("expected no new entries on repeat poll, got %#v", got)
	}
	if len(requestPaths) != 1 {
		t.Fatalf("expected repeat poll to stop after 1 page once caught up, got %d: %v", len(requestPaths), requestPaths)
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
