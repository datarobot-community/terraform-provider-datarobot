package provider

import (
	"archive/zip"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// TestZipDirectoryNoExtraRoot ensures that the zipDirectory helper does not add an
// extra top-level folder (e.g., docker_context/) and uses forward slashes for entries.
func TestZipDirectoryNoExtraRoot(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "zipdir-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	subDir := filepath.Join(tmpDir, "sub")
	subNested := filepath.Join(subDir, "nested")
	if err := os.MkdirAll(subNested, 0o755); err != nil {
		t.Fatalf("failed to create nested dirs: %v", err)
	}

	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(subDir, "file2.txt")
	file3 := filepath.Join(subNested, "file3.txt")
	if err := os.WriteFile(file1, []byte("one"), 0o644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("two"), 0o644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}
	if err := os.WriteFile(file3, []byte("three"), 0o644); err != nil {
		t.Fatalf("failed to write file3: %v", err)
	}

	zipPath := filepath.Join(tmpDir, "out.zip")
	content, err := zipDirectory(tmpDir, zipPath)
	if err != nil {
		t.Fatalf("zipDirectory failed: %v", err)
	}
	if len(content) == 0 {
		t.Fatalf("expected non-empty zip content")
	}

	// Write the returned bytes to a new file (zipDirectory already removed temp zip)
	zipCopy := filepath.Join(tmpDir, "copy.zip")
	if err := os.WriteFile(zipCopy, content, 0o644); err != nil {
		t.Fatalf("failed to write zip copy: %v", err)
	}

	zf, err := zip.OpenReader(zipCopy)
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	defer zf.Close()

	entries := make([]string, 0, len(zf.File))
	for _, f := range zf.File {
		entries = append(entries, f.Name)
		// Directory entries must end with a slash
		if f.FileInfo().IsDir() && !strings.HasSuffix(f.Name, "/") {
			t.Fatalf("directory entry %q missing trailing slash", f.Name)
		}
	}

	// We expect entries like: "file1.txt", "sub/", "sub/file2.txt"
	// No entry should start with the base directory name (which is random temp path)
	baseName := filepath.Base(tmpDir)
	for _, e := range entries {
		if strings.HasPrefix(e, baseName+"/") {
			// On Windows original bug would show directory; ensure it's not present.
			// Provide platform info for debugging.
			t.Fatalf("unexpected root folder in zip entry %q (base %q, GOOS %s)", e, baseName, runtime.GOOS)
		}
		if strings.Contains(e, "\\") { // should be forward slashes only
			t.Fatalf("zip entry %q contains backslash path separators", e)
		}
	}

	requiredFiles := []string{"file1.txt", "sub/file2.txt", "sub/nested/file3.txt"}
	for _, rf := range requiredFiles {
		found := false
		for _, e := range entries {
			if e == rf {
				found = true
				break
			}
		}
		if !found {
			sort.Strings(entries)
			t.Fatalf("expected file entry %q not found; entries=%v", rf, entries)
		}
	}
}

// TestZipDirectoryExplicitRootExclusion focuses narrowly on ensuring that the
// top-level directory name is never embedded as a prefix in any archive entry
// and that all path separators are forward slashes.
func TestZipDirectoryExplicitRootExclusion(t *testing.T) {
	root, err := os.MkdirTemp("", "zipdir-root-*")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(root)

	// Create nested structure
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	f1 := filepath.Join(root, "top.txt")
	f2 := filepath.Join(nested, "deep.txt")
	if err := os.WriteFile(f1, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(f2, []byte("y"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	zipPath := filepath.Join(root, "ctx.zip")
	data, err := zipDirectory(root, zipPath)
	if err != nil {
		t.Fatalf("zipDirectory: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("expected non-empty zip")
	}

	// Persist to open via zip reader
	copyPath := filepath.Join(root, "copy.zip")
	if err := os.WriteFile(copyPath, data, 0o644); err != nil {
		t.Fatalf("write copy: %v", err)
	}

	zr, err := zip.OpenReader(copyPath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer zr.Close()

	base := filepath.Base(root)
	sawTop := false
	sawDeep := false
	for _, f := range zr.File {
		name := f.Name
		if strings.HasPrefix(name, base+"/") {
			t.Fatalf("unexpected root directory prefix: %q (base=%q)", name, base)
		}
		if strings.Contains(name, "\\") {
			t.Fatalf("entry contains backslash: %q", name)
		}
		if name == "top.txt" {
			sawTop = true
		}
		if name == "a/b/deep.txt" {
			sawDeep = true
		}
	}
	if !sawTop || !sawDeep {
		t.Fatalf("missing expected entries; sawTop=%v sawDeep=%v", sawTop, sawDeep)
	}
}
