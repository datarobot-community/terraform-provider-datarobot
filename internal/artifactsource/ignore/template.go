package ignore

import _ "embed"

// CLI source: cli/internal/workload/wapi/template.go (embeds the CLI's own
// drignore.tmpl). The provider's copy adds a Terraform section, because
// source.dir can be the directory holding the configuration that declares it.
//
// DefaultTemplate is the .drignore content later PRs drop at source.dir on
// first apply when generate_ignore is true. Embedded so we don't depend on
// file lookups at runtime.
//
//go:embed drignore.tmpl
var DefaultTemplate []byte
