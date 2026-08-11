package client

import (
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
