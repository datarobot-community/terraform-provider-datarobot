package provider

import (
	"os"
	"testing"
)

func TestCustomModelLLMValidationDirectoryCreation(t *testing.T) {
	// This is a unit test that just tests the directory creation portion
	// of the TestAccCustomModelLLMValidationResource test

	folderPath := "custom_model_llm_validation"

	// Make sure the directory doesn't exist at the start
	os.RemoveAll(folderPath)

	// Make sure we clean up after the test
	defer os.RemoveAll(folderPath)

	// First, test the case where the directory already exists
	if err := os.Mkdir(folderPath, 0755); err != nil {
		t.Fatalf("Failed to create test directory for first case: %v", err)
	}

	// Now use our helper function, which should handle the "directory exists" case
	if err := createOrCleanDirectory(folderPath); err != nil {
		t.Fatalf("createOrCleanDirectory failed: %v", err)
	}

	// Test we can write a file to the clean directory
	modelContents := `import pandas as pd

def load_model(code_dir):
	return True

def score(data, model, **kwargs):
	return pd.DataFrame({"answer": ["answer"]})`

	if err := os.WriteFile(folderPath+"/custom.py", []byte(modelContents), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Verify the file was written correctly
	contents, err := os.ReadFile(folderPath + "/custom.py")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	if string(contents) != modelContents {
		t.Fatalf("File contents don't match what was written")
	}
}
