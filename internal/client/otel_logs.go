package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-querystring/query"
)

const defaultOtelLogStreamPageSize = 100

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

func (s *ServiceImpl) pollNewOtelEntityLogs(
	ctx context.Context,
	entityType, entityID, buildID string,
	state *otelLogStreamState,
	onLine func(OtelLogEntry),
) {
	entries, err := s.getOtelEntityLogEntries(ctx, entityType, entityID, defaultOtelLogStreamPageSize, buildID)
	if err != nil {
		return
	}
	state.emitNew(entries, onLine)
}
