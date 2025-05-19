package provider

import (
	"os"
	"testing"
)

func TestCreateOrCleanDirectory(t *testing.T) {
	dirPath := "test_directory"

	// Ensure directory is removed at the end of the test
	defer os.RemoveAll(dirPath)

	// Test case 1: Directory does not exist initially
	// First, make sure it doesn't exist
	os.RemoveAll(dirPath)

	err := createOrCleanDirectory(dirPath)
	if err != nil {
		t.Fatalf("Failed to create directory when it didn't exist: %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		t.Fatal("Directory should exist but doesn't")
	}

	// Create a file in the directory
	testFilePath := dirPath + "/test.txt"
	if err := os.WriteFile(testFilePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test case 2: Directory exists with content
	err = createOrCleanDirectory(dirPath)
	if err != nil {
		t.Fatalf("Failed to clean and recreate directory: %v", err)
	}

	// Verify directory exists but file is gone
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		t.Fatal("Directory should exist but doesn't")
	}

	if _, err := os.Stat(testFilePath); !os.IsNotExist(err) {
		t.Fatal("File should have been removed but still exists")
	}

	// Test case 3: Call function twice in a row
	err = createOrCleanDirectory(dirPath)
	if err != nil {
		t.Fatalf("Failed on second call to createOrCleanDirectory: %v", err)
	}

	// Verify directory still exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		t.Fatal("Directory should still exist but doesn't")
	}
}
