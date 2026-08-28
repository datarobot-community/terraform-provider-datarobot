// Package ignore decides which files artifact source upload excludes.
//
// CLI source: cli/internal/workload/ignore/doc.go
//
// The effective set is the union of hardcoded system excludes and patterns from
// <project-root>/.drignore (gitignore syntax). If .drignore is absent,
// .wapiignore is loaded for CLI compatibility, and Matcher.Notice says so. A
// missing ignore file is fine: only system excludes apply.
//
// System excludes are not overridable by negation patterns. They cover the CLI
// state directory under .datarobot/workload, the legacy .wapi, .git,
// .gitignore, .datarobot.yaml (never sent to the Files API), and Terraform's
// own .terraform and terraform.tfstate files. Every name but the state
// directory matches at any depth, and the comparison folds case.
//
// It also owns the filenames themselves, for the code that seeds a starter file
// rather than reading one: FileName is what to write, and Locate answers
// whether a project already has one under either name. Keeping both sides on
// this package's answer is what stops a project being given a name that a later
// upload does not look for.
package ignore
