package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
)

const (
	defaultOtelLogStreamPageSize = 100
	// maxOtelLogStreamPages caps how many pages one poll will walk to catch up on
	// a burst of new records, so a runaway build can't make polling loop forever.
	maxOtelLogStreamPages = 20
)

type getOtelLogsRequest struct {
	Limit        int      `url:"limit,omitempty"`
	SearchKeys   []string `url:"searchKeys,omitempty"`
	SearchValues []string `url:"searchValues,omitempty"`
}

func otelLogsRequest(limit int, buildID string) *getOtelLogsRequest {
	req := &getOtelLogsRequest{Limit: limit}
	if buildID != "" {
		req.SearchKeys = []string{"build_id"}
		req.SearchValues = []string{buildID}
	}
	return req
}

func otelLogKey(entry OtelLogEntry) string {
	return entry.Timestamp + "\x00" + entry.Level + "\x00" + entry.Message
}

// FormatOtelLogEntry renders one OTEL log record for display.
func FormatOtelLogEntry(entry OtelLogEntry) string {
	line := fmt.Sprintf("[%s] %s: %s", entry.Timestamp, strings.ToUpper(entry.Level), entry.Message)
	if entry.StackTrace != "" {
		line += "\n" + entry.StackTrace
	}
	return line
}

func (s *ServiceImpl) getOtelEntityLogEntries(
	ctx context.Context,
	entityType, entityID string,
	limit int,
	buildID string,
) ([]OtelLogEntry, error) {
	pathValues, _ := query.Values(otelLogsRequest(limit, buildID))

	resp, err := Get[PaginatedResponse[OtelLogEntry]](s.client, ctx, "/otel/"+entityType+"/"+entityID+"/logs/?"+pathValues.Encode())
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

type otelLogStreamState struct {
	seen map[string]struct{}
}

func newOtelLogStreamState() *otelLogStreamState {
	return &otelLogStreamState{seen: make(map[string]struct{})}
}

func (st *otelLogStreamState) emitNew(entries []OtelLogEntry, onLine func(OtelLogEntry)) {
	if onLine == nil || len(entries) == 0 {
		return
	}

	// Server returns newest first; emit chronological.
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		key := otelLogKey(entry)
		if _, ok := st.seen[key]; ok {
			continue
		}
		st.seen[key] = struct{}{}
		onLine(entry)
	}
}

// pollNewOtelEntityLogs walks pages (server returns newest first) until it
// reaches an entry already emitted by a prior poll, or runs out of pages. A
// single fixed-size page only covers the newest defaultOtelLogStreamPageSize
// records; without following Next, a burst of more new records than that
// between two polls would silently drop everything past the first page
// instead of just being reported late.
func (s *ServiceImpl) pollNewOtelEntityLogs(
	ctx context.Context,
	entityType, entityID, buildID string,
	state *otelLogStreamState,
	onLine func(OtelLogEntry),
) {
	basePath := "/otel/" + entityType + "/" + entityID + "/logs/"
	pathValues, _ := query.Values(otelLogsRequest(defaultOtelLogStreamPageSize, buildID))
	requestPath := basePath + "?" + pathValues.Encode()

	var fresh []OtelLogEntry
	// True when the walk stopped on maxOtelLogStreamPages with pages still unread, so
	// records older than everything in `fresh` were never fetched and never will be.
	hitPageCap := false
	for page := 0; requestPath != ""; page++ {
		if page >= maxOtelLogStreamPages {
			hitPageCap = true
			break
		}

		resp, err := Get[PaginatedResponse[OtelLogEntry]](s.client, ctx, requestPath)
		if err != nil {
			break
		}

		caughtUp := false
		for _, entry := range resp.Data {
			if _, ok := state.seen[otelLogKey(entry)]; ok {
				caughtUp = true
				break
			}
			fresh = append(fresh, entry)
		}
		if caughtUp || resp.Next == "" {
			break
		}
		requestPath = otelLogsNextPath(basePath, resp.Next)
	}

	if hitPageCap {
		// Appended to the newest-first slice, so emitNew prints it before the records it
		// precedes - where the missing ones would have been.
		fresh = append(fresh, otelLogStreamGapEntry())
	}

	state.emitNew(fresh, onLine)
}

// otelLogStreamGapEntry is a provider-generated notice rather than a server record: it
// marks where a single poll fell further behind than maxOtelLogStreamPages of catch-up
// can cover. The cap keeps a runaway build from looping forever, but without this line
// the truncation is invisible and the stream reads as complete.
func otelLogStreamGapEntry() OtelLogEntry {
	return OtelLogEntry{
		Level: "warn",
		// Same shape the server uses, so it sorts and renders with the real records.
		Timestamp: time.Now().UTC().Format("2006-01-02 15:04:05.000000") + "+00:00",
		Message: fmt.Sprintf(
			"terraform-provider-datarobot: log stream fell behind by more than %d records "+
				"(%d pages x %d); older records from this interval were skipped",
			maxOtelLogStreamPages*defaultOtelLogStreamPageSize,
			maxOtelLogStreamPages,
			defaultOtelLogStreamPageSize,
		),
	}
}

// otelLogsNextPath turns a PaginatedResponse.Next value (an absolute URL) into
// a request path relative to the configured API endpoint, the same way
// GetAllPages handles the same field. The server echoes the searchKeys /
// searchValues filter back in Next, so following it keeps the build_id scope.
func otelLogsNextPath(basePath, next string) string {
	if idx := strings.Index(next, "?"); idx != -1 {
		return basePath + next[idx:]
	}
	return basePath
}
