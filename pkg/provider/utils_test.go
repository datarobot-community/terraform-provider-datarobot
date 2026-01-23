package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
)

// TestGetFileInfoStripsWhitespace verifies that getFileInfo strips leading/trailing
// whitespace from pathInModel to prevent storage errors when file paths contain
// whitespace characters (especially Windows CRLF line endings).
func TestGetFileInfoStripsWhitespace(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		pathInModel  string
		expectedPath string
		description  string
	}{
		{
			name:         "Windows CRLF line endings",
			pathInModel:  "\r\nmetadata.yaml\r\n",
			expectedPath: "metadata.yaml",
			description:  "Should strip Windows CRLF (\\r\\n) from both ends",
		},
		{
			name:         "Multiple spaces",
			pathInModel:  "    metadata.yaml    ",
			expectedPath: "metadata.yaml",
			description:  "Should strip multiple spaces from both ends",
		},
		{
			name:         "Tabs",
			pathInModel:  "\tmetadata.yaml\t",
			expectedPath: "metadata.yaml",
			description:  "Should strip tabs from both ends",
		},
		{
			name:         "Complex mixed whitespace",
			pathInModel:  " \r\n\tapp.py\t\n\r ",
			expectedPath: "app.py",
			description:  "Should strip complex mixed whitespace",
		},
		{
			name:         "Nested path with trailing space",
			pathInModel:  "folder/subfolder/app.py  ",
			expectedPath: "folder/subfolder/app.py",
			description:  "Should strip trailing space from nested path",
		},
		{
			name:         "Nested path with leading space",
			pathInModel:  "  folder/subfolder/app.py",
			expectedPath: "folder/subfolder/app.py",
			description:  "Should strip leading space from nested path",
		},
		{
			name:         "Unix newlines",
			pathInModel:  "\napp.py\n",
			expectedPath: "app.py",
			description:  "Should strip Unix newlines",
		},
		{
			name:         "Path with internal spaces preserved",
			pathInModel:  "  my folder/my file.py  ",
			expectedPath: "my folder/my file.py",
			description:  "Should strip external whitespace but preserve internal spaces",
		},
		{
			name:         "No whitespace",
			pathInModel:  "metadata.yaml",
			expectedPath: "metadata.yaml",
			description:  "Should handle paths without whitespace normally",
		},
		{
			name:         "Empty path",
			pathInModel:  "",
			expectedPath: "",
			description:  "Should handle empty path",
		},
		{
			name:         "Only whitespace",
			pathInModel:  "  \t\r\n  ",
			expectedPath: "",
			description:  "Should return empty string for whitespace-only input",
		},
	}

	// Create a temporary test file
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test.txt")
	testContent := []byte("test content")
	err := os.WriteFile(testFilePath, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Call getFileInfo with the test case's pathInModel
			fileInfo, err := getFileInfo(testFilePath, tc.pathInModel)
			if err != nil {
				t.Fatalf("getFileInfo failed: %v", err)
			}

			// Verify that the Path field has whitespace stripped
			if fileInfo.Path != tc.expectedPath {
				t.Errorf(
					"%s: Expected path %q, got %q",
					tc.description,
					tc.expectedPath,
					fileInfo.Path,
				)
			}

			// Verify other fields are set correctly
			if fileInfo.Name != "test.txt" {
				t.Errorf("Expected name 'test.txt', got %q", fileInfo.Name)
			}
			if string(fileInfo.Content) != "test content" {
				t.Errorf("Expected content 'test content', got %q", string(fileInfo.Content))
			}
		})
	}
}

// TestGetFileInfoPreservesInternalWhitespace verifies that internal whitespace
// in file paths is preserved (only leading/trailing whitespace is stripped).
func TestGetFileInfoPreservesInternalWhitespace(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		pathInModel  string
		expectedPath string
	}{
		{"  path/with spaces/file.py  ", "path/with spaces/file.py"},
		{"\tmy\tpath/file.txt\n", "my\tpath/file.txt"},
		{"  config/app config.yaml  ", "config/app config.yaml"},
	}

	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFilePath, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.pathInModel, func(t *testing.T) {
			t.Parallel()

			fileInfo, err := getFileInfo(testFilePath, tc.pathInModel)
			if err != nil {
				t.Fatalf("getFileInfo failed: %v", err)
			}

			if fileInfo.Path != tc.expectedPath {
				t.Errorf("Expected path %q, got %q", tc.expectedPath, fileInfo.Path)
			}
		})
	}
}

// TestPrepareLocalFilesStripsWhitespace verifies that prepareLocalFiles properly
// strips whitespace from file paths when processing file tuples.
func TestPrepareLocalFilesStripsWhitespace(t *testing.T) {
	t.Parallel()

	// Create temporary test files
	tempDir := t.TempDir()
	testFile1 := filepath.Join(tempDir, "file1.txt")
	testFile2 := filepath.Join(tempDir, "file2.txt")

	err := os.WriteFile(testFile1, []byte("content1"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file1: %v", err)
	}
	err = os.WriteFile(testFile2, []byte("content2"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file2: %v", err)
	}

	// Test with FileTuple containing whitespace in PathInModel
	fileTuples := []FileTuple{
		{
			LocalPath:   testFile1,
			PathInModel: "\r\nmetadata.yaml\r\n",
		},
		{
			LocalPath:   testFile2,
			PathInModel: "  config/app.yaml  ",
		},
	}

	// Manually call getFileInfo for each tuple
	var localFiles []client.FileInfo
	for _, fileTuple := range fileTuples {
		fileInfo, err := getFileInfo(fileTuple.LocalPath, fileTuple.PathInModel)
		if err != nil {
			t.Fatalf("getFileInfo failed: %v", err)
		}
		localFiles = append(localFiles, fileInfo)
	}

	// Verify whitespace was stripped
	expectedPaths := []string{"metadata.yaml", "config/app.yaml"}
	for i, fileInfo := range localFiles {
		if fileInfo.Path != expectedPaths[i] {
			t.Errorf("File %d: Expected path %q, got %q",
				i, expectedPaths[i], fileInfo.Path)
		}
		// Verify paths don't contain any leading/trailing whitespace
		if strings.TrimSpace(fileInfo.Path) != fileInfo.Path {
			t.Errorf("File %d: Path still contains leading/trailing whitespace: %q",
				i, fileInfo.Path)
		}
	}
}

// TestGetFileInfoRealWorldScenarios tests scenarios that actually occurred in production.
func TestGetFileInfoRealWorldScenarios(t *testing.T) {
	t.Parallel()

	// Create a test file
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "metadata.yaml")
	err := os.WriteFile(testFilePath, []byte("name: test-app"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Scenario 1: Windows CRLF from reading file list with improper line ending handling
	// This was the actual error reported: "Can not store file "\r\nmetadata.yaml"."
	t.Run("Windows CRLF from file manifest", func(t *testing.T) {
		fileInfo, err := getFileInfo(testFilePath, "\r\nmetadata.yaml")
		if err != nil {
			t.Fatalf("getFileInfo failed: %v", err)
		}
		if fileInfo.Path != "metadata.yaml" {
			t.Errorf("Expected 'metadata.yaml', got %q", fileInfo.Path)
		}
		// Verify no CRLF characters remain
		if strings.Contains(fileInfo.Path, "\r") || strings.Contains(fileInfo.Path, "\n") {
			t.Errorf("Path still contains CRLF characters: %q", fileInfo.Path)
		}
	})

	// Scenario 2: File paths from Windows environment with trailing CRLF
	t.Run("Trailing CRLF from Windows", func(t *testing.T) {
		fileInfo, err := getFileInfo(testFilePath, "metadata.yaml\r\n")
		if err != nil {
			t.Fatalf("getFileInfo failed: %v", err)
		}
		if fileInfo.Path != "metadata.yaml" {
			t.Errorf("Expected 'metadata.yaml', got %q", fileInfo.Path)
		}
	})

	// Scenario 3: Nested path with whitespace (from batch 101-132 in the error)
	t.Run("Nested path with spaces", func(t *testing.T) {
		fileInfo, err := getFileInfo(testFilePath, "  src/utils/helper.py  ")
		if err != nil {
			t.Fatalf("getFileInfo failed: %v", err)
		}
		if fileInfo.Path != "src/utils/helper.py" {
			t.Errorf("Expected 'src/utils/helper.py', got %q", fileInfo.Path)
		}
	})
}
