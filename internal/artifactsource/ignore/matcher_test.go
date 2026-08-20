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
		{".terraform", true},
		{".terraform/providers/lock.json", true},
		{"terraform.tfstate", true},
		{"terraform.tfstate.backup", true},
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

func TestSystemExcludes_NotOverridable(t *testing.T) {
	t.Parallel()

	m := FromLines([]string{"!.wapi", "!.git", "!.datarobot.yaml", "!.terraform"})

	assert.True(t, m.Match(".wapi", true))
	assert.True(t, m.Match(".wapi/manifest.json", false))
	assert.True(t, m.Match(".git", true))
	assert.True(t, m.Match(".datarobot.yaml", false))
	assert.True(t, m.Match(".terraform", true))
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
	assert.False(t, m.Match(".drignore", false))
}
