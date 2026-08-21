// Package wapi manages .wapi/ — local BASE sync state for source.dir
// (not stored in Terraform state).
//
// CLI source: cli/internal/workload/wapi/doc.go
//
// Provider differences from CLI:
//   - No history.log.
//   - No validator/v10.
//   - Initialize does not write .drignore / .wapiignore (PR2 already does).
package wapi
