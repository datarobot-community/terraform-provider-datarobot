package ignore

import (
	"strings"

	_ "embed"
)

// CLI source: cli/internal/workload/wapi/template.go
//
// DefaultTemplate is the .drignore content Ensure writes at source.dir when
// generate_ignore is true. Embedded so we don't depend on file lookups at runtime.
//
//go:embed drignore.tmpl
var DefaultTemplate []byte

// FromDefaultTemplate builds a Matcher from DefaultTemplate lines (plan-time
// hashing when the file is not on disk yet).
func FromDefaultTemplate() *Matcher {
	return FromLines(strings.Split(string(DefaultTemplate), "\n"))
}
