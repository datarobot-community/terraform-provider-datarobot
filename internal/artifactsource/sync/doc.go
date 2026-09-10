// Package sync will host the CLI three-way artifact code sync engine
// (BASE / LOCAL / REMOTE). Classify maps a hash triple to an Action;
// Diff turns BASE / LOCAL / REMOTE manifests into a SyncPlan.
//
// CLI source: cli/internal/workload/sync/doc.go
package sync
