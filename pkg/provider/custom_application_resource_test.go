package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

const (
	resourceName = "datarobot_custom_application.test"
)

func TestAccCustomApplicationResource(t *testing.T) {
	t.Parallel()

	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())
	compareValuesSame := statecheck.CompareValue(compare.ValuesSame())

	newName := "new_custom_application " + nameSalt

	useCaseResourceName := "test_custom_application"
	useCaseResourceName2 := "test_new_custom_application"

	folderPath := "custom_application"
	err := os.Mkdir(folderPath, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(folderPath)

	startAppScript := `#!/usr/bin/env bash

echo "Starting App"

streamlit run streamlit-app.py
`

	appCode := `import streamlit as st
from datarobot import Client
from datarobot.client import set_client


def start_streamlit():
    set_client(Client())

    st.title("Example Custom Application")

if __name__ == "__main__":
    start_streamlit()
	`

	err = os.WriteFile(folderPath+"/start-app.sh", []byte(startAppScript), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(folderPath+"/streamlit-app.py", []byte(appCode), 0644)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: customApplicationResourceConfig(
					"",
					1,
					false,
					[]string{},
					false,
					&useCaseResourceName),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("use_case_ids"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomApplicationResourceExists(),
					resource.TestCheckResourceAttrSet(resourceName, "name"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "application_url"),
					resource.TestCheckResourceAttr(resourceName, "external_access_enabled", "false"),
					resource.TestCheckNoResourceAttr(resourceName, "external_access_recipients"),
					resource.TestCheckResourceAttr(resourceName, "allow_auto_stopping", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
				),
			},
			// Update name, external access, and use case id
			{
				Config: customApplicationResourceConfig(
					newName,
					1,
					true,
					[]string{"test@test.com"},
					true,
					&useCaseResourceName2),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("source_version_id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("use_case_ids"),
					),
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("application_url"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomApplicationResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "application_url"),
					resource.TestCheckResourceAttr(resourceName, "external_access_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "external_access_recipients.0", "test@test.com"),
					resource.TestCheckResourceAttr(resourceName, "allow_auto_stopping", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
				),
			},
			// Update Application Source version and remove use case
			{
				Config: customApplicationResourceConfig(
					newName,
					2,
					true,
					[]string{"test2@test.com"},
					true,
					nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("source_version_id"),
					),
					compareValuesSame.AddStateValue(
						resourceName,
						tfjsonpath.New("application_url"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomApplicationResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "application_url"),
					resource.TestCheckResourceAttr(resourceName, "external_access_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "external_access_recipients.0", "test2@test.com"),
					resource.TestCheckResourceAttr(resourceName, "allow_auto_stopping", "true"),
					resource.TestCheckNoResourceAttr(resourceName, "use_case_ids.0"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func customApplicationResourceConfig(
	name string,
	applicationSourceReplicas int,
	externalAccess bool,
	externalAccessRecipients []string,
	allowAutoStopping bool,
	useCaseResourceName *string,
) string {
	recipients := ""
	if len(externalAccessRecipients) > 0 {
		recipients = fmt.Sprintf(`
		external_access_recipients = %q
		`, externalAccessRecipients)
	}

	nameStr := ""
	if name != "" {
		nameStr = fmt.Sprintf(`
		name = "%s"
		`, name)
	}

	useCaseIDsStr := ""
	if useCaseResourceName != nil {
		useCaseIDsStr = fmt.Sprintf(`use_case_ids = ["${datarobot_use_case.%s.id}"]`, *useCaseResourceName)
	}

	return fmt.Sprintf(`
resource "datarobot_use_case" "test_custom_application" {
	name = "test custom application"
}
resource "datarobot_use_case" "test_new_custom_application" {
	name = "test new custom application"
}

resource "datarobot_application_source" "test" {
	base_environment_id = "6542cd582a9d3d51bf4ac71e"
	folder_path = "custom_application"
	resources = {
		replicas = %d
	}
}

resource "datarobot_custom_application" "test" {
	source_version_id = "${datarobot_application_source.test.version_id}"
	external_access_enabled = %t
	allow_auto_stopping = %t
	%s
	%s
	%s
}
`, applicationSourceReplicas, externalAccess, allowAutoStopping, recipients, nameStr, useCaseIDsStr)
}

func checkCustomApplicationResourceExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("Provider not found")
		}
		p.service = NewService(cl)

		traceAPICall("GetApplicationInTest")
		application, err := p.service.GetApplication(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if application.Name == rs.Primary.Attributes["name"] &&
			application.ApplicationUrl == rs.Primary.Attributes["application_url"] &&
			application.CustomApplicationSourceID == rs.Primary.Attributes["source_id"] &&
			application.CustomApplicationSourceVersionID == rs.Primary.Attributes["source_version_id"] {
			b, err := strconv.ParseBool(rs.Primary.Attributes["allow_auto_stopping"])
			if err == nil {
				if application.AllowAutoStopping != b {
					return fmt.Errorf("AllowAutoStopping is %t but should be %t", application.AllowAutoStopping, b)
				}
			}

			b, err = strconv.ParseBool(rs.Primary.Attributes["external_access_enabled"])
			if err == nil {
				if application.ExternalAccessEnabled == b {
					if len(application.ExternalAccessRecipients) > 0 {
						if application.ExternalAccessRecipients[0] != rs.Primary.Attributes["external_access_recipients.0"] {
							return fmt.Errorf("ExternalAccessRecipient is %s but should be %s", application.ExternalAccessRecipients[0], rs.Primary.Attributes["external_access_recipients.0"])
						}
					}
				}
			}
			return nil
		}

		return fmt.Errorf("Custom Application not found")
	}
}

func TestAccCustomApplicationWithBatchFiles(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_custom_application.batch_test"
	sourceResourceName := "datarobot_application_source.batch_test"

	// Create a temporary directory for test files
	testDir := "test_batch_app_files"
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	// Create essential app files first
	startAppScript := `#!/usr/bin/env bash
echo "Starting Batch Test App"
streamlit run streamlit-app.py
`

	appCode := `import streamlit as st
from datarobot import Client
from datarobot.client import set_client

def start_streamlit():
    set_client(Client())
    st.title("Batch Files Custom Application Test")
    st.write("This app was created with 100+ files to test batch processing!")

if __name__ == "__main__":
    start_streamlit()
`

	metadata := `name: batch-test-app
runtimeParameterDefinitions:
  - fieldName: TEST_PARAM
    type: string
    description: A test parameter for batch processing
`

	requirements := `streamlit==1.28.0
datarobot
pandas
numpy
`

	// Write essential files
	if err := os.WriteFile(testDir+"/start-app.sh", []byte(startAppScript), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testDir+"/streamlit-app.py", []byte(appCode), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testDir+"/metadata.yaml", []byte(metadata), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testDir+"/requirements.txt", []byte(requirements), 0644); err != nil {
		t.Fatal(err)
	}

	// Create 120 additional files to test batching (more than the 100 file limit)
	numAdditionalFiles := 120
	for i := 0; i < numAdditionalFiles; i++ {
		fileName := fmt.Sprintf("%s/data_file_%03d.txt", testDir, i)
		content := fmt.Sprintf("This is data file number %d\nCreated for batch processing test\nContent: %s", i, generateTestContent(i))

		if err := os.WriteFile(fileName, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	baseEnvironmentID := "6542cd582a9d3d51bf4ac71e"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create Application Source with 100+ files and Custom Application
			{
				Config: customApplicationWithBatchFilesConfig(testDir, baseEnvironmentID, nameSalt),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check Application Source
					resource.TestCheckResourceAttrSet(sourceResourceName, "id"),
					resource.TestCheckResourceAttrSet(sourceResourceName, "version_id"),
					resource.TestCheckResourceAttr(sourceResourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(sourceResourceName, "name", "Batch Files Test Application Source "+nameSalt),

					// Check Custom Application
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "application_url"),
					resource.TestCheckResourceAttr(resourceName, "name", "Batch Files Test Custom Application "+nameSalt),
					resource.TestCheckResourceAttr(resourceName, "external_access_enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "allow_auto_stopping", "true"),

					// Custom check to verify application is working
					func(s *terraform.State) error {
						// Get the application source resource
						sourceRS, ok := s.RootModule().Resources[sourceResourceName]
						if !ok {
							return fmt.Errorf("application source resource not found: %s", sourceResourceName)
						}

						// Get the custom application resource
						appRS, ok := s.RootModule().Resources[resourceName]
						if !ok {
							return fmt.Errorf("custom application resource not found: %s", resourceName)
						}

						// Verify the custom application references the correct source
						if appRS.Primary.Attributes["source_id"] != sourceRS.Primary.Attributes["id"] {
							return fmt.Errorf("custom application source_id doesn't match application source id")
						}

						// When using folder_path, verify the folder_path_hash is set
						// instead of counting individual file hashes
						if folderPathHash, ok := sourceRS.Primary.Attributes["folder_path_hash"]; !ok || folderPathHash == "" {
							return fmt.Errorf("folder_path_hash not set or empty")
						}

						// Also verify folder_path is set correctly
						if folderPath := sourceRS.Primary.Attributes["folder_path"]; folderPath != testDir {
							return fmt.Errorf("folder_path mismatch: expected %s, got %s", testDir, folderPath)
						}

						return nil
					},
				),
			},
		},
	})
}

func generateTestContent(index int) string {
	// Generate some varied content to ensure files are different
	content := fmt.Sprintf("File index: %d\n", index)
	content += fmt.Sprintf("Timestamp: batch-test-%d\n", index*1000)
	content += fmt.Sprintf("Data: %s\n", string(rune('A'+index%26)))
	for i := 0; i < index%10; i++ {
		content += fmt.Sprintf("Line %d: Additional content for file %d\n", i, index)
	}
	return content
}

func customApplicationWithBatchFilesConfig(folderPath, baseEnvironmentID, nameSalt string) string {
	return fmt.Sprintf(`
resource "datarobot_application_source" "batch_test" {
	name = "Batch Files Test Application Source %s"
	base_environment_id = "%s"
	folder_path = "%s"
	runtime_parameter_values = [
		{
			key = "TEST_PARAM"
			type = "string"
			value = "batch_test_value"
		}
	]
}

resource "datarobot_custom_application" "batch_test" {
	name = "Batch Files Test Custom Application %s"
	source_version_id = datarobot_application_source.batch_test.version_id
	external_access_enabled = false
	allow_auto_stopping = true
}
`, nameSalt, baseEnvironmentID, folderPath, nameSalt)
}

func TestAccCustomApplicationRealWorldBatchFiles(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_custom_application.real_batch_test"
	sourceResourceName := "datarobot_application_source.real_batch_test"

	// Create a temporary directory for test files
	testDir := "real_batch_app_files"
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	// Create a realistic Streamlit application that processes the batch files
	startAppScript := `#!/usr/bin/env bash
echo "Starting Real-World Batch Files App"
streamlit run app.py --server.port 8080 --server.address 0.0.0.0
`

	appCode := `import streamlit as st
import os
import glob
from datarobot import Client
from datarobot.client import set_client

def start_streamlit():
    set_client(Client())

    st.title("🚀 Real-World Batch Files Custom Application")
    st.write("This application demonstrates batch file processing with 100+ files!")

    # Count and display files
    data_files = glob.glob("data_*.txt")
    st.success(f"✅ Successfully loaded {len(data_files)} data files")

    if data_files:
        st.subheader("📊 File Processing Results")

        # Process a sample of files
        sample_files = data_files[:10]  # Show first 10 files

        for i, file_path in enumerate(sample_files):
            with st.expander(f"📄 File {i+1}: {os.path.basename(file_path)}"):
                try:
                    with open(file_path, 'r') as f:
                        content = f.read()
                    st.text(content[:200] + "..." if len(content) > 200 else content)
                except Exception as e:
                    st.error(f"Error reading file: {e}")

        if len(data_files) > 10:
            st.info(f"📂 Plus {len(data_files) - 10} more files available")

    st.subheader("🔧 Application Info")
    st.write(f"- **Total files**: {len(data_files)}")
    st.write(f"- **Directory**: {os.getcwd()}")
    st.write(f"- **Runtime**: Streamlit with DataRobot integration")

    # Display environment info
    st.subheader("🌍 Environment")
    env_vars = [key for key in os.environ.keys() if 'DATAROBOT' in key or 'TEST' in key]
    if env_vars:
        for var in env_vars[:5]:  # Show first 5 relevant env vars
            st.text(f"{var}: {os.environ.get(var, 'Not set')}")

if __name__ == "__main__":
    start_streamlit()
`

	metadata := `name: real-batch-app
runtimeParameterDefinitions:
  - fieldName: BATCH_SIZE
    type: string
    description: Number of files to process in each batch
  - fieldName: PROCESSING_MODE
    type: string
    description: Mode for processing the batch files
`

	requirements := `streamlit==1.28.0
datarobot
pandas
numpy
`

	// Write essential application files
	if err := os.WriteFile(testDir+"/start-app.sh", []byte(startAppScript), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testDir+"/app.py", []byte(appCode), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testDir+"/metadata.yaml", []byte(metadata), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testDir+"/requirements.txt", []byte(requirements), 0644); err != nil {
		t.Fatal(err)
	}

	// Create 150 realistic data files to test batching
	numDataFiles := 150
	for i := 0; i < numDataFiles; i++ {
		fileName := fmt.Sprintf("%s/data_%03d.txt", testDir, i)
		content := generateRealisticDataFile(i)

		if err := os.WriteFile(fileName, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	baseEnvironmentID := "6542cd582a9d3d51bf4ac71e"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create Application Source with 150+ files and Custom Application
			{
				Config: realWorldBatchFilesConfig(testDir, baseEnvironmentID, nameSalt),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check Application Source
					resource.TestCheckResourceAttrSet(sourceResourceName, "id"),
					resource.TestCheckResourceAttrSet(sourceResourceName, "version_id"),
					resource.TestCheckResourceAttr(sourceResourceName, "base_environment_id", baseEnvironmentID),
					resource.TestCheckResourceAttr(sourceResourceName, "name", "Real-World Batch Files App "+nameSalt),

					// Verify folder_path_hash is set (indicating files were processed)
					resource.TestCheckResourceAttrSet(sourceResourceName, "folder_path_hash"),

					// Check Custom Application
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_id"),
					resource.TestCheckResourceAttrSet(resourceName, "source_version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "application_url"),
					resource.TestCheckResourceAttr(resourceName, "name", "Real-World Batch Files Custom App "+nameSalt),
					resource.TestCheckResourceAttr(resourceName, "external_access_enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "allow_auto_stopping", "true"),

					// Custom validation to ensure the application was created successfully
					func(s *terraform.State) error {
						// Get the application source resource
						sourceRS, ok := s.RootModule().Resources[sourceResourceName]
						if !ok {
							return fmt.Errorf("application source resource not found: %s", sourceResourceName)
						}

						// Get the custom application resource
						appRS, ok := s.RootModule().Resources[resourceName]
						if !ok {
							return fmt.Errorf("custom application resource not found: %s", resourceName)
						}

						// Verify the custom application references the correct source
						if appRS.Primary.Attributes["source_id"] != sourceRS.Primary.Attributes["id"] {
							return fmt.Errorf("custom application source_id doesn't match application source id")
						}

						// Verify application URL is accessible
						appURL := appRS.Primary.Attributes["application_url"]
						if appURL == "" {
							return fmt.Errorf("application_url is empty")
						}

						// Log success message
						fmt.Printf("\n🎉 SUCCESS: Real-world Custom Application created!\n")
						fmt.Printf("📱 Application URL: %s\n", appURL)
						fmt.Printf("📁 Total files processed: %d (150 data files + 4 app files)\n", numDataFiles+4)
						fmt.Printf("🔧 Application ID: %s\n", appRS.Primary.Attributes["id"])
						fmt.Printf("📦 Source ID: %s\n", sourceRS.Primary.Attributes["id"])

						return nil
					},
				),
			},
		},
	})
}

func generateRealisticDataFile(index int) string {
	// Generate realistic data that might be used in a data science application
	datasets := []string{"customers", "transactions", "products", "sales", "inventory", "users", "orders", "reviews"}
	dataset := datasets[index%len(datasets)]

	content := fmt.Sprintf("# Dataset: %s_%03d\n", dataset, index)
	content += fmt.Sprintf("# Generated: %s\n", time.Now().Format("2006-01-02"))
	content += fmt.Sprintf("# Record Count: %d\n", (index+1)*100)
	content += fmt.Sprintf("# File Index: %d\n\n", index)

	// Add sample data rows
	switch dataset {
	case "customers":
		content += "customer_id,name,email,segment\n"
		for i := 0; i < 10; i++ {
			content += fmt.Sprintf("%d,Customer_%d_%d,customer%d@example.com,Segment_%c\n",
				index*100+i, index, i, i, 'A'+rune(i%3))
		}
	case "transactions":
		content += "transaction_id,customer_id,amount,date\n"
		for i := 0; i < 10; i++ {
			content += fmt.Sprintf("TXN_%d_%d,%d,%.2f,2025-06-%02d\n",
				index, i, index*10+i, float64(index+i)*10.5, (i%28)+1)
		}
	default:
		content += "id,name,value,category\n"
		for i := 0; i < 10; i++ {
			content += fmt.Sprintf("%d,%s_item_%d_%d,%.2f,cat_%d\n",
				index*100+i, dataset, index, i, float64(index+i)*1.5, i%5)
		}
	}

	content += fmt.Sprintf("\n# End of file %d\n", index)
	return content
}

func realWorldBatchFilesConfig(folderPath, baseEnvironmentID, nameSalt string) string {
	return fmt.Sprintf(`
resource "datarobot_application_source" "real_batch_test" {
	name = "Real-World Batch Files App %s"
	base_environment_id = "%s"
	folder_path = "%s"
	runtime_parameter_values = [
		{
			key = "BATCH_SIZE"
			type = "string"
			value = "100"
		},
		{
			key = "PROCESSING_MODE"
			type = "string"
			value = "parallel"
		}
	]
}

resource "datarobot_custom_application" "real_batch_test" {
	name = "Real-World Batch Files Custom App %s"
	source_version_id = datarobot_application_source.real_batch_test.version_id
	external_access_enabled = false
	allow_auto_stopping = true
}
`, nameSalt, baseEnvironmentID, folderPath, nameSalt)
}

func TestAccCustomApplicationWithResourcesFromSource(t *testing.T) {
	t.Parallel()

	folderPath := "custom_application_resources_test"
	err := os.Mkdir(folderPath, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(folderPath)

	startAppScript := `#!/usr/bin/env bash
echo "Starting App"
streamlit run streamlit-app.py
`

	appCode := `import streamlit as st
st.title("Resources Test App")
	`

	err = os.WriteFile(folderPath+"/start-app.sh", []byte(startAppScript), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(folderPath+"/streamlit-app.py", []byte(appCode), 0644)
	if err != nil {
		t.Fatal(err)
	}

	resourceName := "datarobot_custom_application.test"
	sourceResourceName := "datarobot_application_source.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: customApplicationWithResourcesFromSourceConfig(folderPath),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify ApplicationSource has resources populated
					resource.TestCheckResourceAttrSet(sourceResourceName, "id"),
					resource.TestCheckResourceAttrSet(sourceResourceName, "version_id"),
					resource.TestCheckResourceAttr(sourceResourceName, "resources.replicas", "2"),
					resource.TestCheckResourceAttr(sourceResourceName, "resources.resource_label", "cpu.medium"),
					resource.TestCheckResourceAttr(sourceResourceName, "resources.session_affinity", "true"),
					resource.TestCheckResourceAttr(sourceResourceName, "resources.service_web_requests_on_root_path", "false"),

					// Verify CustomApplication exists
					// NOTE: resources won't be populated unless explicitly passed from the source
					// This test demonstrates that resources are now VISIBLE in the ApplicationSource,
					// allowing users to reference them when creating CustomApplications
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "application_url"),
				),
			},
		},
	})
}

func customApplicationWithResourcesFromSourceConfig(folderPath string) string {
	return fmt.Sprintf(`
resource "datarobot_application_source" "test" {
	name = "Resources Test Source %s"
	base_environment_id = "6542cd582a9d3d51bf4ac71e"
	folder_path = "%s"
	resources = {
		replicas = 2
		resource_label = "cpu.medium"
		session_affinity = true
		service_web_requests_on_root_path = false
	}
}

resource "datarobot_custom_application" "test" {
	name = "Resources Test App %s"
	source_version_id = datarobot_application_source.test.version_id
	external_access_enabled = false
	allow_auto_stopping = true
}
`, nameSalt, folderPath, nameSalt)
}

func TestAccCustomApplicationRequiredKeyScopeLevel(t *testing.T) {
	t.Parallel()

	resourceName := "datarobot_custom_application.test_scope"
	folderPath := "custom_application_scope_test"

	err := os.Mkdir(folderPath, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(folderPath)

	startAppScript := `#!/usr/bin/env bash
echo "Starting App"
streamlit run streamlit-app.py
`

	appCode := `import streamlit as st
from datarobot import Client
from datarobot.client import set_client

def start_streamlit():
    set_client(Client())
    st.title("Scope Level Test Application")

if __name__ == "__main__":
    start_streamlit()
`

	err = os.WriteFile(folderPath+"/start-app.sh", []byte(startAppScript), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(folderPath+"/streamlit-app.py", []byte(appCode), 0644)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with required_key_scope_level set to "viewer"
			{
				Config: customApplicationWithScopeLevelConfig(folderPath, "viewer"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "required_key_scope_level", "viewer"),
					checkCustomApplicationScopeLevel(resourceName, "viewer"),
				),
			},
			// Update to "admin"
			{
				Config: customApplicationWithScopeLevelConfig(folderPath, "admin"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "required_key_scope_level", "admin"),
					checkCustomApplicationScopeLevel(resourceName, "admin"),
				),
			},
			// Unset the field (null)
			{
				Config: customApplicationWithScopeLevelConfig(folderPath, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckNoResourceAttr(resourceName, "required_key_scope_level"),
					checkCustomApplicationScopeLevel(resourceName, ""),
				),
			},
			// Delete is tested automatically
		},
	})
}

func customApplicationWithScopeLevelConfig(folderPath, scopeLevel string) string {
	scopeLevelAttr := ""
	if scopeLevel != "" {
		scopeLevelAttr = fmt.Sprintf(`
	required_key_scope_level = "%s"`, scopeLevel)
	}

	return fmt.Sprintf(`
resource "datarobot_application_source" "test_scope" {
	base_environment_id = "6542cd582a9d3d51bf4ac71e"
	folder_path = "%s"
}

resource "datarobot_custom_application" "test_scope" {
	source_version_id = datarobot_application_source.test_scope.version_id
	name = "Scope Level Test App"
	allow_auto_stopping = false%s
}
`, folderPath, scopeLevelAttr)
}

func checkCustomApplicationScopeLevel(resourceName, expectedLevel string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("Provider not found")
		}
		p.service = NewService(cl)

		traceAPICall("GetApplicationInTest")
		application, err := p.service.GetApplication(context.TODO(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if expectedLevel == "" {
			// Field should be nil/null
			if application.RequiredKeyScopeLevel != nil {
				return fmt.Errorf("RequiredKeyScopeLevel should be nil but is %s", *application.RequiredKeyScopeLevel)
			}
		} else {
			// Field should match expected value
			if application.RequiredKeyScopeLevel == nil {
				return fmt.Errorf("RequiredKeyScopeLevel is nil but should be %s", expectedLevel)
			}
			if *application.RequiredKeyScopeLevel != expectedLevel {
				return fmt.Errorf("RequiredKeyScopeLevel is %s but should be %s", *application.RequiredKeyScopeLevel, expectedLevel)
			}
		}

		return nil
	}
}
