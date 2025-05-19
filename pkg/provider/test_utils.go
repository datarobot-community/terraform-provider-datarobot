package provider

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// checkFileWithState is a test check function that verifies if a file exists in the state
// and prints all available files if the check fails.
func checkFileWithState(resourceName, filePath string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		// Get all files from the state
		var files []string
		for k, v := range rs.Primary.Attributes {
			if strings.HasPrefix(k, "files.") {
				files = append(files, fmt.Sprintf("%s: %s", k, v))
			}
		}

		// Check if the specific file exists
		expectedKey := fmt.Sprintf("files.%s", filePath)
		if v, ok := rs.Primary.Attributes[expectedKey]; !ok {
			return fmt.Errorf("file not found in state. Available files:\n%s", strings.Join(files, "\n"))
		} else if v != filePath {
			return fmt.Errorf("file path mismatch. Expected %s, got %s. Available files:\n%s", filePath, v, strings.Join(files, "\n"))
		}

		return nil
	}
}
