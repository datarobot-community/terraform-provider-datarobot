package sync

// CLI source: cli/internal/workload/sync/engine.go (Engine.Sync).
//
// Provider difference from CLI: the CLI's Sync interleaves the display
// package between phases (progress bars, the conflict menu, a dry-run
// exit). terraform apply has none of that, so Run is just the three
// phases back to back.

import "context"

// Run is the whole pipeline in one call: Plan, then the local half, then
// the remote half. It is what datarobot_artifact uses; callers that need
// to inspect the plan before it is applied (or to apply only one half)
// still call Plan / ExecuteLocal / ExecuteRemote directly.
//
// Run does not release the sync lock — the caller owns Close, so a
// failure mid-pipeline still frees .wapi/sync.lock:
//
//	engine, err := sync.New(dir, artifactID, files, store)
//	...
//	defer engine.Close()
//	result, err := engine.Run(ctx)
//
// An empty plan still runs both halves: they no-op on the network and on
// disk, but phase 6 records the observed catalog version in .wapi/, which
// is what keeps the next Plan on the non-drifted fast path.
func (e *Engine) Run(ctx context.Context) (*Result, error) {
	if _, err := e.Plan(ctx); err != nil {
		return nil, err
	}

	if err := e.ExecuteLocal(ctx); err != nil {
		return nil, err
	}

	return e.ExecuteRemote(ctx)
}
