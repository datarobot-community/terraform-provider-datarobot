package ignore

// CLI source: cli/internal/workload/ignore/matcher.go
//
// Provider differences from CLI:
//   - System excludes are gitignore patterns compiled by the same engine as
//     the user file, rather than matched by string prefix. Unanchored names
//     therefore match at any depth and terraform.tfstate.* needs no special
//     case.
//   - Extra system excludes for Terraform's own working files.
//   - Locate uses a local fileExists; the CLI has an fsutil package.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

const (
	// FileName is the canonical user-editable ignore file at the project root.
	// It is exported so the package that writes the starter template names the
	// same file this one reads, rather than keeping a second copy of the string.
	FileName = ".drignore"

	// LegacyFileName is the name FileName replaced. It is read only when
	// FileName is absent: a project set up before the rename has a committed
	// one, and ignoring it would upload the .venv, node_modules and .env it was
	// written to keep out.
	LegacyFileName = ".wapiignore"
)

// systemPatterns are always-ignored paths, not overridable by the user file.
// Ordering mirrors the CLI's systemExcludes so the two lists diff cleanly.
//
// The CLI state directory is anchored to the root with a leading slash rather
// than listed bare: an unanchored entry would also exclude tool state under
// .datarobot/cli/, which the CLI syncs today. The manifest under it is
// rewritten by every deploy, so uploading it would have the next run find a
// changed file, rebuild, and roll a workload nobody had touched, for ever.
//
// The remaining names are unanchored, so they match at any depth the way a
// bare gitignore pattern does: a vendored checkout's sub/.git is as unwanted in
// an upload as the one at the root.
//
// The Terraform entries are provider-specific. source.dir can be the same
// directory as the configuration that declares it, and a state file uploaded
// into an artifact is both useless to the container and a credential leak.
//
// Patterns must be lowercase. Match folds the path it is given, because macOS
// and Windows preserve case without distinguishing it: a project holding a
// differently-cased .DataRobot.yaml would otherwise upload it. The cost is that
// a case-sensitive filesystem also excludes a genuine .Git, which is not a
// directory anyone keeps beside the real one.
var systemPatterns = []string{
	"/.datarobot/workload",
	".wapi",
	".git",
	".gitignore",
	".datarobot.yaml",
	".terraform",
	"terraform.tfstate",
	"terraform.tfstate.*",
}

// system holds systemPatterns compiled once. Matching only reads, so one
// instance shared across Matchers is safe for concurrent use.
var system = gitignore.CompileIgnoreLines(systemPatterns...)

// Matcher decides whether a path is excluded from upload. Match is safe for
// concurrent use after New or FromLines.
type Matcher struct {
	user *gitignore.GitIgnore // nil when the user has no ignore file

	// source is the base filename the patterns came from, empty when none was
	// loaded. Kept so callers can name the file the user actually has instead
	// of assuming the current name.
	source string

	// shadowed is a candidate that lost to source, empty when only one was
	// present. Its patterns are not applied and the user is told so.
	shadowed string
}

// Locate returns the path New would take its patterns from, or "" when the
// project has neither name. The code that writes the starter template asks this
// rather than testing for a filename itself, so the two cannot drift.
//
// It uses the same file test as New, which means it agrees on the awkward
// cases: a directory at one of these names is not an ignore file, and neither
// is a symlink whose target is missing. Answering "present" for either would
// have the writer skip the template while New found nothing to read, leaving
// the project with no patterns at all.
func Locate(projectDir string) string {
	for _, name := range []string{FileName, LegacyFileName} {
		path := filepath.Join(projectDir, name)
		if fileExists(path) {
			return path
		}
	}

	return ""
}

// New loads the project's ignore file, preferring FileName and falling back to
// LegacyFileName when it is absent. A project with neither is fine: only the
// hardcoded system excludes apply.
//
// The newer name wins outright when both are present. Merging two files would
// make the effective pattern set depend on an ordering nobody wrote down, and a
// project that has both is mid-rename, where the new file is the one being
// edited. ShadowWarning reports the loser, because a file at the project root
// that looks like it is filtering the upload and is not needs saying out loud.
//
// An unreadable file fails rather than falling through to the older name.
// Quietly uploading under a different set of patterns than the one the user is
// looking at is how a .env reaches the remote; a failed apply they can retry is
// the better end of that trade. The loser is only ever stat'd, so a leftover
// this run would not have applied cannot fail it either.
func New(projectDir string) (*Matcher, error) {
	gi, err := compileIfPresent(filepath.Join(projectDir, FileName))
	if err != nil {
		return nil, err
	}

	if gi != nil {
		return &Matcher{user: gi, source: FileName, shadowed: shadowedBy(projectDir)}, nil
	}

	gi, err = compileIfPresent(filepath.Join(projectDir, LegacyFileName))
	if err != nil {
		return nil, err
	}

	if gi == nil {
		return &Matcher{}, nil
	}

	return &Matcher{user: gi, source: LegacyFileName}, nil
}

// shadowedBy names the older file sitting beside the one in effect, empty when
// there is none. Existence is all the warning needs, so this does not read it:
// compiling a file whose patterns are inert would be work thrown away, and its
// read errors would fail an apply that was never going to use it.
func shadowedBy(projectDir string) string {
	if fileExists(filepath.Join(projectDir, LegacyFileName)) {
		return LegacyFileName
	}

	return ""
}

// compileIfPresent returns nil, nil when path does not exist, so New can try
// the next candidate without treating absence as failure.
func compileIfPresent(path string) (*gitignore.GitIgnore, error) {
	gi, err := gitignore.CompileIgnoreFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, err
	}

	return gi, nil
}

// fileExists reports whether path is a regular file. A directory at an ignore
// file's name, or a symlink whose target is missing, is not one.
//
// CLI source: cli/internal/fsutil/fsutil.go (FileExists).
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

// Source is the base filename the patterns were read from, empty when the
// Matcher has none. Callers writing a message about a pattern use it so the
// advice points at the file that exists on disk.
func (m *Matcher) Source() string { return m.source }

// Notice is a one-line account of the legacy filename still being in use, empty
// in every other case. The file keeps working either way, so this is the only
// signal a user gets that the old name is on its way out.
//
// It carries the coordination caveat because the ignore file is part of the
// uploaded set: renaming it deletes the old name from the remote, and a
// collaborator on a CLI from before this change reads only that name.
//
// Surfacing it is the caller's job, which keeps this package doing no I/O
// beyond finding the ignore file. Same for ShadowWarning. The two are never
// both set: the legacy name is only ever in effect when it is the only one.
func (m *Matcher) Notice() string {
	if m.source != LegacyFileName {
		return ""
	}

	return fmt.Sprintf(
		"Using %s for upload ignore patterns. That name is deprecated: rename the file to %s. "+
			"It is uploaded with your code, so agree the rename with anyone else deploying this project.",
		LegacyFileName, FileName)
}

// ShadowWarning reports a second ignore file whose patterns are not being
// applied, empty when the project has at most one.
//
// It is separate from Notice, and a warning rather than housekeeping, because
// the two states differ in what they cost. Using the old name costs nothing;
// the patterns still apply. Having both means a set of patterns the user wrote
// is silently inert, and the ways to end up there do not look like mistakes: a
// half-finished rename, or a `touch .drignore` in response to the deprecation
// line. The ignore file is itself part of the uploaded set, so a machine that
// links to an artifact carrying the other name downloads it right next to the
// one already on disk.
//
// Nothing else in an apply would mention it. The first sign otherwise is a
// .venv on the remote.
func (m *Matcher) ShadowWarning() string {
	if m.shadowed == "" {
		return ""
	}

	return fmt.Sprintf(
		"Both %s and %s are present. %s is the one in effect, and the patterns in %s are not applied. "+
			"Merge them into %s and delete %s.",
		FileName, m.shadowed, FileName, m.shadowed, FileName, m.shadowed)
}

// FromLines builds a Matcher from in-memory pattern lines. Empty/nil means
// "system excludes only".
func FromLines(lines []string) *Matcher {
	if len(lines) == 0 {
		return &Matcher{}
	}

	return &Matcher{user: gitignore.CompileIgnoreLines(lines...)}
}

// Match reports whether relPath should be excluded. isDir lets directory-only
// patterns ("build/") prune subtrees.
func (m *Matcher) Match(relPath string, isDir bool) bool {
	if relPath == "" {
		return false
	}

	// A directory is probed with a trailing slash and a file without one, which
	// is the single question git asks. Directory-only patterns match nothing
	// else, and a bare name matches with the slash or without it, so one probe
	// covers both shapes.
	//
	// Asking twice, unslashed first, would make a directory negation
	// unreachable: given "*" then "!build/", the "*" rule answers yes for
	// "build" and the rule written to bring it back is never consulted.
	probe := relPath
	if isDir {
		probe += "/"
	}

	if system.MatchesPath(strings.ToLower(probe)) {
		return true
	}

	if m.user == nil {
		return false
	}

	return m.user.MatchesPath(probe)
}
