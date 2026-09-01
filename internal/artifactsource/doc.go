// Package artifactsource provides local directory upload orchestration over the
// DataRobot Files API client (internal/client/filesapi).
//
// PushDirectory walks a local directory, hashes files, and uploads via either
// the stage path (small change sets) or zip/fromFile path (large change sets).
// When CatalogID and CatalogVersionID are set, that version is read back from
// the catalog to serve as the diff base, so only added, modified, and deleted
// files are synced rather than the whole tree.
package artifactsource
