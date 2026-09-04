package wapi

import (
	"errors"
	"fmt"
)

// CLI source: cli/internal/workload/wapi/errors.go

// ErrAlreadyLinked is returned by Initialize when a state directory already
// exists, at either the current or the legacy location.
var ErrAlreadyLinked = errors.New("project already linked: state directory exists")

// ErrNotInitialized is returned by Load/Save when the state directory is missing.
var ErrNotInitialized = errors.New("state directory not found")

// CorruptedError wraps a read or parse failure for a file under the state directory.
type CorruptedError struct {
	Path string
	Err  error
}

func (e *CorruptedError) Error() string {
	return fmt.Sprintf("state file is corrupted at %s: %v", e.Path, e.Err)
}

func (e *CorruptedError) Unwrap() error {
	return e.Err
}
