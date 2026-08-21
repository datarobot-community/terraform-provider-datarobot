package wapi

import (
	"errors"
	"fmt"
)

// CLI source: cli/internal/workload/wapi/errors.go

// ErrAlreadyLinked is returned by Initialize when .wapi/ already exists.
var ErrAlreadyLinked = errors.New("project already linked: .wapi/ exists")

// ErrNotInitialized is returned by Load/Save when .wapi/ is missing.
var ErrNotInitialized = errors.New(".wapi/ not found")

// CorruptedError wraps a read or parse failure for a file under .wapi/.
type CorruptedError struct {
	Path string
	Err  error
}

func (e *CorruptedError) Error() string {
	return fmt.Sprintf(".wapi/ file is corrupted at %s: %v", e.Path, e.Err)
}

func (e *CorruptedError) Unwrap() error {
	return e.Err
}
