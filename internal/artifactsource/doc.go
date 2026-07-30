// Package artifactsource provides local directory upload orchestration over the
// DataRobot Files API client (internal/client/filesapi).
//
// PushDirectory walks a local directory, hashes files, and uploads via either
// the stage path (small change sets) or zip/fromFile path (large change sets).
// When CatalogID and BaseFiles (per-path hashes from Terraform state) are set,
// only added, modified, and deleted files are synced incrementally.
package artifactsource
