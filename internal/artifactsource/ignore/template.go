package ignore

import _ "embed"

// CLI source: cli/internal/workload/wapi/template.go
//
// DefaultTemplate is the .drignore content later PRs drop at source.dir on
// first apply when generate_ignore is true. Embedded so we don't depend on
// file lookups at runtime.
//
//go:embed drignore.tmpl
var DefaultTemplate []byte
