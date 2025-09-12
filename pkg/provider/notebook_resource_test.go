package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccNotebookResource(t *testing.T) {
	t.Parallel()
	resourceName := "datarobot_notebook.test"

	// Create a temporary notebook file for testing
	tempDir := t.TempDir()
	notebookPath := filepath.Join(tempDir, "test_notebook.ipynb")

	notebookContent := []byte(`{
  "cells": [
    {
      "cell_type": "markdown",
      "metadata": {
        "id": "f8939937",
        "language": "markdown"
      },
      "source": [
        "# Test Notebook",
        "This is a test notebook for terraform provider testing."
      ]
    },
    {
      "cell_type": "code",
      "metadata": {
        "id": "0b4e03d1",
        "language": "python"
      },
      "source": [
        "# Import libraries",
        "import pandas as pd",
        "import numpy as np",
        "",
        "# Example code",
        "df = pd.DataFrame({'a': [1, 2, 3], 'b': [4, 5, 6]})",
        "print(df)"
      ]
    }
  ],
	"metadata": {
		"kernelspec": {
			"name": "python",
			"language": "python",
			"display_name": "Python 3.11"
		},
		"language_info": {
			"name": "python",
			"version": "3.11"
		}
	},
	"nbformat": 4,
	"nbformat_minor": 5
}`)

	err := os.WriteFile(notebookPath, notebookContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test notebook file: %v", err)
	}

	// Calculate file hash for verification
	hash := sha256.Sum256(notebookContent)
	hashStr := hex.EncodeToString(hash[:])

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccNotebookResourceConfig(notebookPath, "test-use-case"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNotebookExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "file_path", notebookPath),
					resource.TestCheckResourceAttr(resourceName, "file_hash", hashStr),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// These fields can't be read from the API during import
				ImportStateVerifyIgnore: []string{
					"file_path",
					"file_hash",
					"use_case_id",
				},
			},
			// Update testing - change use_case_id
			{
				Config: testAccNotebookResourceConfigUpdated(notebookPath, "updated-use-case"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNotebookExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "file_path", notebookPath),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestNotebookResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	NewNotebookResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}

func TestNotebookResourceMethods(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_client.NewMockService(ctrl)

	r := &models.NotebookResource{
		provider: &Provider{
			service: mockService,
		},
	}

	// Test Metadata
	t.Run("Metadata", func(t *testing.T) {
		ctx := context.Background()
		req := fwresource.MetadataRequest{
			ProviderTypeName: "datarobot",
		}
		resp := &fwresource.MetadataResponse{}

		r.Metadata(ctx, req, resp)

		if resp.TypeName != "datarobot_notebook" {
			t.Errorf("Expected type name to be 'datarobot_notebook', got: %s", resp.TypeName)
		}
	})
}

func testAccNotebookResourceConfig(filePath, useCaseName string) string {
	return fmt.Sprintf(`
resource "datarobot_use_case" "test" {
  name = "%s"
  description = "Test use case for notebook testing"
}

resource "datarobot_notebook" "test" {
  file_path = "%s"
  use_case_id = datarobot_use_case.test.id
}
`, useCaseName, filePath)
}

func testAccNotebookResourceConfigUpdated(filePath, useCaseName string) string {
	return fmt.Sprintf(`
resource "datarobot_use_case" "updated" {
  name = "%s"
  description = "Updated test use case for notebook testing"
}

resource "datarobot_notebook" "test" {
  file_path = "%s"
  use_case_id = datarobot_use_case.updated.id
}
`, useCaseName, filePath)
}

func testAccCheckNotebookExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Notebook ID is set")
		}

		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("Provider not configured")
		}

		p.service = client.NewService(cl)

		traceAPICall("GetNotebook")
		_, err := p.service.GetNotebook(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error fetching notebook with ID %s: %s", rs.Primary.ID, err)
		}

		return nil
	}
}
