// Package wapi manages the local BASE sync state for source.dir (not stored in
// Terraform state).
//
// CLI source: cli/internal/workload/wapi/doc.go
//
// State lives at <projectDir>/.datarobot/workload, the location the CLI resolves,
// with a fallback to the legacy <projectDir>/.wapi while it is still there. Both
// sides therefore read and write one set of files in a mixed CLI/Terraform tree.
//
// Provider differences from CLI:
//   - No history.log.
//   - No validator/v10: only the manifest version is checked on load.
//   - Initialize does not write .drignore / .wapiignore (PR2 already does).
//   - No EnsureMigrated: the provider reads the legacy directory where it
//     stands but leaves relocating it to the CLI.
package wapi
