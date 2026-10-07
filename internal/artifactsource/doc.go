// Package artifactsource provides the local-directory primitives the
// DataRobot Files API client (internal/client/filesapi) is driven with.
//
// CollectLocalFiles walks and hashes a directory; FingerprintDirectory
// digests the result into the source.dir hash the resource keeps in state;
// UploadFiles pushes a set of those files through the stage path (small
// change sets) or the zip/fromFile path (large ones).
//
// Deciding what to upload, delete, or leave alone is not done here. That is
// internal/artifactsource/sync's three-way engine (BASE / LOCAL / REMOTE),
// which is the only thing that can express a deletion; this package just
// carries out what it asks for.
package artifactsource
