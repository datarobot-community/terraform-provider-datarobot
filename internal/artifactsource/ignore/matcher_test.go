package ignore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemExcludes_AlwaysApply(t *testing.T) {
	t.Parallel()

	m := FromLines(nil)

	cases := []struct {
		path string
		want bool
	}{
		{".wapi", true},
		{".wapi/config.json", true},
		{".git", true},
		{".git/HEAD", true},
		{".gitignore", true},
		{".datarobot.yaml", true},
		{".datarobot/workload", true},
		{".datarobot/workload/manifest.json", true},
		{".terraform", true},
		{".terraform/providers/lock.json", true},
		{"terraform.tfstate", true},
		{"terraform.tfstate.backup", true},
		{"terraform.tfstate.d/env/terraform.tfstate", true},
		{"terraform.tfvars", true},
		{"prod.auto.tfvars", true},
		{"terraform.tfvars.json", true},
		{"main.tf", false},
		// Nested occurrences: a vendored checkout's .git is as unwanted as the
		// one at the root, and user patterns already match at any depth.
		{"sub/.git/HEAD", true},
		{"vendor/mod/.gitignore", true},
		{"sub/.terraform/providers", true},
		{"sub/terraform.tfstate.backup", true},
		{"envs/prod/secrets.tfvars", true},
		// The state directory is the one root-anchored entry, so CLI tool state
		// under .datarobot/cli still uploads and a subproject copy is untouched.
		{".datarobot/cli/state.json", false},
		{"sub/.datarobot/workload/manifest.json", false},
		{"agent.py", false},
		{".drignore", false},
		{".wapiignore", false},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			assert.Equal(t, tc.want, m.Match(tc.path, false))
		})
	}
}

func TestSystemExcludes_CaseInsensitive(t *testing.T) {
	t.Parallel()

	m := FromLines(nil)

	// macOS and Windows preserve case without distinguishing it, so a
	// differently-cased name is the same file to the filesystem.
	for _, path := range []string{
		".DataRobot.yaml",
		".Git/HEAD",
		".GitIgnore",
		".Terraform/providers",
		"Terraform.tfstate",
		"terraform.TFSTATE.backup",
		".DataRobot/Workload/manifest.json",
	} {
		t.Run(path, func(t *testing.T) {
			assert.True(t, m.Match(path, false))
		})
	}
}

func TestSystemExcludes_NotOverridable(t *testing.T) {
	t.Parallel()

	m := FromLines([]string{
		"!.wapi", "!.git", "!.datarobot.yaml", "!.terraform",
		"!.datarobot/workload", "!terraform.tfstate",
		"!*.tfvars", "!*.tfvars.json",
	})

	assert.True(t, m.Match(".wapi", true))
	assert.True(t, m.Match(".wapi/manifest.json", false))
	assert.True(t, m.Match(".git", true))
	assert.True(t, m.Match(".datarobot.yaml", false))
	assert.True(t, m.Match(".terraform", true))
	assert.True(t, m.Match(".datarobot/workload", true))
	assert.True(t, m.Match("terraform.tfstate", false))
	assert.True(t, m.Match("terraform.tfvars", false))
	assert.True(t, m.Match("terraform.tfvars.json", false))
}

// A project that already has an ignore file never receives the starter
// template, so anything the template alone covered would upload. Variable files
// carry the credentials the configuration was given, which is why they are
// system excludes rather than template lines.
func TestSystemExcludes_TfvarsWithLegacyIgnoreFileOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, LegacyFileName), []byte(".venv\n"), 0o644))

	m, err := New(dir)
	require.NoError(t, err)

	assert.True(t, m.Match("terraform.tfvars", false))
	assert.True(t, m.Match("prod.auto.tfvars", false))
	assert.True(t, m.Match("terraform.tfvars.json", false))
	assert.True(t, m.Match(".venv", true))
	assert.False(t, m.Match("main.tf", false))
}

func TestUserPatterns(t *testing.T) {
	t.Parallel()

	m := FromLines([]string{
		"__pycache__",
		"*.pyc",
		".env",
		"*.LOCAL.*",
		"build/",
		"!keep.me",
	})

	cases := []struct {
		path  string
		isDir bool
		want  bool
	}{
		{path: "agent.py", want: false},
		{path: "agent.pyc", want: true},
		{path: "src/utils.pyc", want: true},
		{path: "src/__pycache__", isDir: true, want: true},
		{path: ".env", want: true},
		{path: "agent.py.LOCAL.20260410T143052Z", want: true},
		{path: "build", isDir: true, want: true},
		{path: "build/dist.tar", want: true},
		{path: "keep.me", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			assert.Equal(t, tc.want, m.Match(tc.path, tc.isDir))
		})
	}
}

func TestUserPatterns_DirectoryNegation(t *testing.T) {
	t.Parallel()

	// A directory is asked about with a trailing slash and nothing else, so the
	// negation written to bring "build" back is reached. Probing the unslashed
	// form first would let the "*" rule answer and never consult "!build/".
	m := FromLines([]string{"*", "!build/"})

	assert.False(t, m.Match("build", true))
	// A regular file named "build" is not what "!build/" re-includes.
	assert.True(t, m.Match("build", false))
	assert.True(t, m.Match("agent.py", false))
}

func TestNew_NoIgnoreFile(t *testing.T) {
	t.Parallel()

	m, err := New(t.TempDir())
	require.NoError(t, err)

	assert.True(t, m.Match(".wapi/foo", false))
	assert.True(t, m.Match(".datarobot.yaml", false))
	assert.False(t, m.Match("agent.py", false))
}

func TestNew_LoadsDrignore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, FileName), []byte("*.tmp\n"), 0o644))

	m, err := New(dir)
	require.NoError(t, err)

	assert.True(t, m.Match("scratch.tmp", false))
	assert.False(t, m.Match("agent.py", false))
}

func TestNew_FallsBackToWapiignore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, LegacyFileName), []byte("*.log\n"), 0o644))

	m, err := New(dir)
	require.NoError(t, err)

	assert.True(t, m.Match("debug.log", false))
	assert.False(t, m.Match("agent.py", false))
}

func TestNew_DrignoreWinsOverWapiignore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, FileName), []byte("*.tmp\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, LegacyFileName), []byte("*.log\n"), 0o644))

	m, err := New(dir)
	require.NoError(t, err)

	assert.True(t, m.Match("scratch.tmp", false))
	assert.False(t, m.Match("debug.log", false))
}

func TestLocate(t *testing.T) {
	t.Parallel()

	t.Run("neither present", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, Locate(t.TempDir()))
	})

	t.Run("drignore wins", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, FileName), nil, 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, LegacyFileName), nil, 0o644))
		assert.Equal(t, filepath.Join(dir, FileName), Locate(dir))
	})

	t.Run("legacy only", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, LegacyFileName), nil, 0o644))
		assert.Equal(t, filepath.Join(dir, LegacyFileName), Locate(dir))
	})

	// A directory at the name is not an ignore file. Answering "present" would
	// have the template writer skip while New found nothing to read, leaving
	// the project with no patterns at all.
	t.Run("directory at the name is not a file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, FileName), 0o755))
		assert.Empty(t, Locate(dir))

		require.NoError(t, os.WriteFile(filepath.Join(dir, LegacyFileName), nil, 0o644))
		assert.Equal(t, filepath.Join(dir, LegacyFileName), Locate(dir))
	})
}

func TestSourceNoticeAndShadowWarning(t *testing.T) {
	t.Parallel()

	t.Run("no ignore file", func(t *testing.T) {
		t.Parallel()
		m, err := New(t.TempDir())
		require.NoError(t, err)
		assert.Empty(t, m.Source())
		assert.Empty(t, m.Notice())
		assert.Empty(t, m.ShadowWarning())
	})

	t.Run("drignore only", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, FileName), []byte("*.tmp\n"), 0o644))

		m, err := New(dir)
		require.NoError(t, err)
		assert.Equal(t, FileName, m.Source())
		assert.Empty(t, m.Notice())
		assert.Empty(t, m.ShadowWarning())
	})

	t.Run("legacy name is deprecated but works", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, LegacyFileName), []byte("*.log\n"), 0o644))

		m, err := New(dir)
		require.NoError(t, err)
		assert.Equal(t, LegacyFileName, m.Source())
		assert.Contains(t, m.Notice(), LegacyFileName)
		assert.Contains(t, m.Notice(), FileName)
		assert.Empty(t, m.ShadowWarning())
	})

	t.Run("both present names the inert file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, FileName), []byte("*.tmp\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, LegacyFileName), []byte("*.log\n"), 0o644))

		m, err := New(dir)
		require.NoError(t, err)
		assert.Equal(t, FileName, m.Source())
		// The two are never both set: the legacy name is only ever in effect
		// when it is the only one.
		assert.Empty(t, m.Notice())
		assert.Contains(t, m.ShadowWarning(), LegacyFileName)
	})
}

func TestNew_UnreadableFileFailsRatherThanFallingBack(t *testing.T) {
	t.Parallel()

	// Uploading under a different set of patterns than the one the user is
	// looking at is how a .env reaches the remote.
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, FileName), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, LegacyFileName), []byte("*.log\n"), 0o644))

	_, err := New(dir)
	require.Error(t, err)
}

func TestDefaultTemplate_ExcludesVenv(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, FileName), DefaultTemplate, 0o644))

	m, err := New(dir)
	require.NoError(t, err)

	assert.True(t, m.Match(".venv", true))
	assert.True(t, m.Match(".env", false))
	assert.True(t, m.Match("node_modules", true))
	assert.False(t, m.Match("agent.py", false))
	assert.False(t, m.Match(FileName, false))
}

func TestDefaultTemplate_ExcludesTerraformWorkingFiles(t *testing.T) {
	t.Parallel()

	// .terraform/, terraform.tfstate and *.tfvars are system excludes; these are
	// the rest of what a source.dir that doubles as the configuration directory
	// holds, and unlike the excludes a user may delete these lines.
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, FileName), DefaultTemplate, 0o644))

	m, err := New(dir)
	require.NoError(t, err)

	assert.True(t, m.Match(".terraform.lock.hcl", false))
	assert.True(t, m.Match(".terraform.tfstate.lock.info", false))
	assert.False(t, m.Match("main.tf", false))
}
