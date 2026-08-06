// CLI source: cli/internal/drapi/filesapi/doc.go
//
// Provider differences from CLI:
//   - Documents HTTPTransport injection via New(transport) instead of the CLI's
//     parameterless New() backed by global drapi/config.
//   - Notes ZipPollInterval/ZipPollTimeout sourced from cli/internal/workload/sync/limits.go.
//
// Package filesapi is a typed Go client for the DataRobot Files API.
//
// It exposes two upload workflows ported from the DR CLI
// (cli/internal/drapi/filesapi/): a synchronous stage workflow
// (CreateStage → UploadToStage → ApplyStage) for small change sets, and an
// async zip workflow (UploadFromZip* → PollStatus) for larger ones.
//
// Construct a client with New, passing the provider's shared HTTP client so
// auth and base URL match the Workload API:
//
//	filesAPI := filesapi.New(serviceClient)
//
// Async zip uploads use ZipPollInterval and ZipPollTimeout when polling
// PollStatus (same defaults as cli/internal/workload/sync/limits.go).
package filesapi
