package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
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

func TestPrepareLocalFilesBatching(t *testing.T) {
	// Create a temporary directory with >100 files to test batching logic
	testDir := t.TempDir()

	// Create 105 files to exceed the batch limit of 100
	totalFiles := 105
	for i := 0; i < totalFiles; i++ {
		fileName := fmt.Sprintf("test_file_%03d.txt", i)
		filePath := testDir + "/" + fileName
		content := fmt.Sprintf("Test file content %d", i)

		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	folderPath := types.StringValue(testDir)
	var files []FileTuple

	localFiles, err := prepareLocalFiles(folderPath, files)
	if err != nil {
		t.Fatalf("prepareLocalFiles failed: %v", err)
	}

	// Should return all 105 files
	if len(localFiles) != totalFiles {
		t.Errorf("Expected %d files, got %d", totalFiles, len(localFiles))
	}

	// Simulate the batching logic that would be used in createCustomModelVersionFromFiles
	const batchSize = 100
	batches := [][]client.FileInfo{}

	for i := 0; i < len(localFiles); i += batchSize {
		end := i + batchSize
		if end > len(localFiles) {
			end = len(localFiles)
		}

		batchToUpload := localFiles[i:end]
		if len(batchToUpload) > 0 {
			batches = append(batches, batchToUpload)
		}
	}

	// Should have exactly 2 batches: 100 files + 5 files
	if len(batches) != 2 {
		t.Errorf("Expected 2 batches, got %d", len(batches))
	}

	// First batch should have 100 files
	if len(batches[0]) != 100 {
		t.Errorf("Expected first batch to have 100 files, got %d", len(batches[0]))
	}

	// Second batch should have 5 files
	if len(batches[1]) != 5 {
		t.Errorf("Expected second batch to have 5 files, got %d", len(batches[1]))
	}

	t.Logf("Successfully created %d batches: [%d files, %d files]", len(batches), len(batches[0]), len(batches[1]))
}
