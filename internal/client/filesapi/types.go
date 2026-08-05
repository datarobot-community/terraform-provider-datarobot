// CLI source: cli/internal/drapi/filesapi/types.go
//
// Provider differences from CLI:
// - Types and helpers are identical; only package-level doc comments were trimmed.
// - CLI references wapi.FileMeta shape compatibility; provider omits that cross-package note.
package filesapi

// FileMeta is the per-file entry in a manifest: SHA-256 hex + byte size.
type FileMeta struct {
	Hash string
	Size int64
}

// Overwrite modes for the stage and zip endpoints.
const (
	OverwriteReplace = "REPLACE"
	OverwriteSkip    = "SKIP"
	OverwriteRename  = "RENAME"
	OverwriteError   = "ERROR"
)

type CatalogResp struct {
	CatalogID        string `json:"catalogId"`
	CatalogVersionID string `json:"catalogVersionId"`
}

type StageResp struct {
	CatalogID string `json:"catalogId"`
	StageID   string `json:"stageId"`
}

type ApplyStageReq struct {
	StageID   string `json:"stageId"`
	Overwrite string `json:"overwrite"`
}

type ApplyStageResp struct {
	CatalogID        string `json:"catalogId"`
	CatalogVersionID string `json:"catalogVersionId"`
	NumFiles         int    `json:"numFiles"`
}

type FromFileResp struct {
	CatalogID        string `json:"catalogId"`
	CatalogVersionID string `json:"catalogVersionId"`
	StatusID         string `json:"statusId"`
}

type StatusResp struct {
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
	StatusID string `json:"statusId,omitempty"`
}

const (
	StatusInitialized      = "INITIALIZED"
	StatusRunningToWorkers = "RUNNING_TO_WORKERS"
	StatusStartedOnWorker  = "STARTED_ON_WORKER"
	StatusCompleted        = "COMPLETED"
	StatusError            = "ERROR"
	StatusAborted          = "ABORTED"
	StatusExpired          = "EXPIRED"
)

// IsTerminalStatus reports whether the async job has finished.
func IsTerminalStatus(s string) bool {
	switch s {
	case StatusCompleted, StatusError, StatusAborted, StatusExpired:
		return true
	}
	return false
}

// IsErrorStatus reports whether the job ended in failure.
func IsErrorStatus(s string) bool {
	switch s {
	case StatusError, StatusAborted, StatusExpired:
		return true
	}
	return false
}

type AllFilesResp struct {
	Data       []AllFilesItem `json:"data"`
	Count      int            `json:"count"`
	TotalCount int            `json:"totalCount"`
	Next       string         `json:"next"`
	Previous   string         `json:"previous"`
}

type AllFilesItem struct {
	FileName     string `json:"fileName"`
	FileType     string `json:"fileType,omitempty"`
	FileSize     int64  `json:"fileSize"`
	FileChecksum string `json:"fileChecksum"`
}

type DeleteFilesReq struct {
	Paths []string `json:"paths"`
}

type DeleteFilesResp struct {
	CatalogID        string              `json:"catalogId"`
	CatalogVersionID string              `json:"catalogVersionId"`
	NumFiles         int                 `json:"numFiles"`
	Results          []DeleteFilesResult `json:"results"`
}

type DeleteFilesResult struct {
	Path            string `json:"path"`
	NumFilesDeleted int    `json:"numFilesDeleted"`
}

type CatalogVersionsResp struct {
	Data       []CatalogVersion `json:"data"`
	Count      int              `json:"count"`
	TotalCount int              `json:"totalCount"`
	Next       string           `json:"next"`
	Previous   string           `json:"previous"`
}

// CatalogVersion is a single entry in the version-history listing for a catalog.
type CatalogVersion struct {
	ID        string `json:"catalogVersionId"`
	CreatedAt string `json:"creationDate"`
	NumFiles  int    `json:"numFiles"`
	TotalSize int64  `json:"size"`
}
