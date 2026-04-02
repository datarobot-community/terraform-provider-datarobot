package provider

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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

// TestGetFileInfoProductionScenarios tests scenarios that actually occurred in production.
func TestGetFileInfoProductionScenarios(t *testing.T) {
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

func TestIsNewRuntimeParametersAttrNotSupportedError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "runtimeParameters is not allowed key",
			err:      errors.New("runtimeParameters is not allowed key"),
			expected: true,
		},
		{
			name:     "feature flag message",
			err:      errors.New("field requires the RUNTIME_PARAMETERS_IMPROVEMENTS feature to be enabled"),
			expected: true,
		},
		{
			name:     "unrelated error",
			err:      errors.New("some other API error"),
			expected: false,
		},
		{
			name:     "partial match in longer message",
			err:      errors.New("400 Bad Request: runtimeParameters is not allowed key in this context"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNewRuntimeParametersAttrNotSupportedError(tt.err)
			if got != tt.expected {
				t.Errorf("isNewRuntimeParametersAttrNotSupportedError(%q) = %v, want %v", tt.err.Error(), got, tt.expected)
			}
		})
	}
}

func makeParamList(t *testing.T, params []RuntimeParameterValue) basetypes.ListValue {
	t.Helper()
	list, diags := listValueFromRuntimParameters(context.Background(), params)
	if diags.HasError() {
		t.Fatalf("makeParamList: %v", diags)
	}
	return list
}

func extractParams(t *testing.T, list basetypes.ListValue) []RuntimeParameterValue {
	t.Helper()
	var out []RuntimeParameterValue
	if diags := list.ElementsAs(context.Background(), &out, false); diags.HasError() {
		t.Fatalf("extractParams: %v", diags)
	}
	return out
}

func TestFormatRuntimeParameterValuesByManagedKeys(t *testing.T) {
	t.Parallel()

	t.Run("null parametersInPlan returns null", func(t *testing.T) {
		t.Parallel()
		nullList := basetypes.NewListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"key":   types.StringType,
				"type":  types.StringType,
				"value": types.StringType,
			},
		})
		got, diags := formatRuntimeParameterValuesByManagedKeys(context.Background(), nil, nullList)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if !got.IsNull() {
			t.Errorf("expected null list, got %v", got)
		}
	})

	t.Run("empty parametersInPlan returns empty list", func(t *testing.T) {
		t.Parallel()
		emptyList := makeParamList(t, []RuntimeParameterValue{})
		got, diags := formatRuntimeParameterValuesByManagedKeys(context.Background(), []client.RuntimeParameter{}, emptyList)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if got.IsNull() || len(got.Elements()) != 0 {
			t.Errorf("expected empty list, got %v", got)
		}
	})

	t.Run("unknown parametersInPlan returns empty list without error", func(t *testing.T) {
		t.Parallel()
		unknownList := basetypes.NewListUnknown(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"key":   types.StringType,
				"type":  types.StringType,
				"value": types.StringType,
			},
		})
		got, diags := formatRuntimeParameterValuesByManagedKeys(context.Background(), nil, unknownList)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if len(got.Elements()) != 0 {
			t.Errorf("expected empty result, got %v elements", len(got.Elements()))
		}
	})

	t.Run("declared key present in API refreshes value", func(t *testing.T) {
		t.Parallel()
		plan := makeParamList(t, []RuntimeParameterValue{
			{Key: types.StringValue("TIMEOUT"), Type: types.StringValue("numeric"), Value: types.StringValue("30")},
		})
		apiParams := []client.RuntimeParameter{
			{FieldName: "TIMEOUT", Type: "numeric", CurrentValue: "60"},
		}
		got, diags := formatRuntimeParameterValuesByManagedKeys(context.Background(), apiParams, plan)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		result := extractParams(t, got)
		if len(result) != 1 {
			t.Fatalf("expected 1 param, got %d", len(result))
		}
		if result[0].Value.ValueString() != "60" {
			t.Errorf("expected value '60', got %q", result[0].Value.ValueString())
		}
		if result[0].Key.ValueString() != "TIMEOUT" {
			t.Errorf("expected key 'TIMEOUT', got %q", result[0].Key.ValueString())
		}
		if result[0].Type.ValueString() != "numeric" {
			t.Errorf("expected type 'numeric', got %q", result[0].Type.ValueString())
		}
	})

	t.Run("declared key missing from API keeps plan value", func(t *testing.T) {
		t.Parallel()
		plan := makeParamList(t, []RuntimeParameterValue{
			{Key: types.StringValue("PENDING_KEY"), Type: types.StringValue("string"), Value: types.StringValue("original")},
		})
		got, diags := formatRuntimeParameterValuesByManagedKeys(context.Background(), []client.RuntimeParameter{}, plan)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		result := extractParams(t, got)
		if len(result) != 1 {
			t.Fatalf("expected 1 param, got %d", len(result))
		}
		if result[0].Value.ValueString() != "original" {
			t.Errorf("expected plan value 'original', got %q", result[0].Value.ValueString())
		}
	})

	t.Run("API keys not declared in plan are excluded", func(t *testing.T) {
		t.Parallel()
		plan := makeParamList(t, []RuntimeParameterValue{
			{Key: types.StringValue("MY_KEY"), Type: types.StringValue("string"), Value: types.StringValue("v")},
		})
		apiParams := []client.RuntimeParameter{
			{FieldName: "MY_KEY", Type: "string", CurrentValue: "v"},
			{FieldName: "INJECTED_BY_METADATA", Type: "string", CurrentValue: "secret"},
		}
		got, diags := formatRuntimeParameterValuesByManagedKeys(context.Background(), apiParams, plan)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		result := extractParams(t, got)
		if len(result) != 1 {
			t.Fatalf("expected 1 param, got %d", len(result))
		}
		if result[0].Key.ValueString() != "MY_KEY" {
			t.Errorf("expected only MY_KEY in result, got %q", result[0].Key.ValueString())
		}
	})

	t.Run("mixed: some refreshed, some kept from plan, extras excluded", func(t *testing.T) {
		t.Parallel()
		plan := makeParamList(t, []RuntimeParameterValue{
			{Key: types.StringValue("A"), Type: types.StringValue("string"), Value: types.StringValue("old-a")},
			{Key: types.StringValue("B"), Type: types.StringValue("string"), Value: types.StringValue("old-b")},
		})
		apiParams := []client.RuntimeParameter{
			{FieldName: "A", Type: "string", CurrentValue: "new-a"},
			{FieldName: "EXTRA", Type: "string", CurrentValue: "should-be-excluded"},
		}
		got, diags := formatRuntimeParameterValuesByManagedKeys(context.Background(), apiParams, plan)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		result := extractParams(t, got)
		if len(result) != 2 {
			t.Fatalf("expected 2 params, got %d", len(result))
		}
		if result[0].Key.ValueString() != "A" || result[0].Value.ValueString() != "new-a" {
			t.Errorf("param A: expected value 'new-a', got key=%q value=%q", result[0].Key.ValueString(), result[0].Value.ValueString())
		}
		if result[1].Key.ValueString() != "B" || result[1].Value.ValueString() != "old-b" {
			t.Errorf("param B: expected value 'old-b' (kept from plan), got key=%q value=%q", result[1].Key.ValueString(), result[1].Value.ValueString())
		}
	})
}
