// Package ignore decides which files artifact source sync excludes.
//
// CLI source: cli/internal/workload/ignore/doc.go
//
// The effective set is the union of hardcoded system excludes and patterns
// from <project-root>/.drignore (gitignore syntax). If .drignore is absent,
// .wapiignore is loaded for CLI compatibility. A missing ignore file is
// fine — only system excludes apply.
//
// System excludes are not overridable by negation patterns. They include
// .wapi, .git, .gitignore, .datarobot.yaml (never sent to the Files API),
// .terraform, and terraform.tfstate / terraform.tfstate.*.
package ignore
