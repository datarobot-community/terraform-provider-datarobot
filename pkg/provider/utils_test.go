package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPrepareLocalFilesDeduplication(t *testing.T) {
	// Create a temporary directory and files for testing
	tmpDir, err := os.MkdirTemp("", "test_dedup")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	testFile1 := filepath.Join(tmpDir, "test1.txt")
	testFile2 := filepath.Join(tmpDir, "test2.txt")
	
	if err := os.WriteFile(testFile1, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create test file 1: %v", err)
	}
	if err := os.WriteFile(testFile2, []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
	}

	// Test case: folder_path includes test1.txt, files explicitly includes test1.txt
	// Should result in only one instance of test1.txt
	folderPath := types.StringValue(tmpDir)
	files := []FileTuple{
		{
			Source:      types.StringValue(testFile1),
			Destination: types.StringValue("test1.txt"), // Same destination as folder scan
		},
		{
			Source:      types.StringValue(testFile2),
			Destination: types.StringValue("custom_test2.txt"), // Different destination
		},
	}

	localFiles, err := prepareLocalFiles(folderPath, files)
	if err != nil {
		t.Fatalf("prepareLocalFiles failed: %v", err)
	}

	// Should have 3 files total: test1.txt (from folder), test2.txt (from folder), custom_test2.txt (from files)
	// test1.txt should not be duplicated even though it's in both folder and files
	if len(localFiles) != 3 {
		t.Errorf("Expected 3 files, got %d", len(localFiles))
	}

	// Verify no duplicate paths
	pathsSeen := make(map[string]bool)
	for _, file := range localFiles {
		if pathsSeen[file.Path] {
			t.Errorf("Duplicate path found: %s", file.Path)
		}
		pathsSeen[file.Path] = true
	}

	// Verify expected paths exist
	expectedPaths := []string{"test1.txt", "test2.txt", "custom_test2.txt"}
	for _, expectedPath := range expectedPaths {
		if !pathsSeen[expectedPath] {
			t.Errorf("Expected path not found: %s", expectedPath)
		}
	}
}
