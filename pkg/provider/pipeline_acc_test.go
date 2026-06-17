package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// ─── pipeline ─────────────────────────────────────────────────────────────────

func TestAccPipelineResource(t *testing.T) {
	t.Parallel()
	rn := "datarobot_pipeline.test"
	srcFile := writeAccPipelineFile(t)
	var initialID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: accPipelineDraftConfig(srcFile),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(rn, "id"),
					resource.TestCheckResourceAttr(rn, "mode", "draft"),
					resource.TestCheckResourceAttrSet(rn, "source_file_hash"),
					captureAttr(rn, "id", &initialID),
					checkPipelineExistsInAPI(rn),
				),
			},
			{
				Config: accPipelineLockedConfig(srcFile),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "mode", "locked"),
					checkWorkloadIDPreserved(rn, &initialID),
					checkPipelineExistsInAPI(rn),
				),
			},
		},
	})
}

func TestAccPipelineReplaceOnDescriptionChange(t *testing.T) {
	t.Parallel()
	rn := "datarobot_pipeline.test"
	srcFile := writeAccPipelineFile(t)
	var initialID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: accPipelineLockedWithDescConfig(srcFile, "first description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "description", "first description"),
					captureAttr(rn, "id", &initialID),
				),
			},
			{
				Config: accPipelineLockedWithDescConfig(srcFile, "second description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "description", "second description"),
					checkIDChanged(rn, &initialID),
				),
			},
		},
	})
}

func checkPipelineExistsInAPI(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("pipeline ID is not set in state")
		}
		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("provider not found")
		}
		p.service = NewService(cl)
		_, err := p.service.GetPipeline(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("GetPipeline(%s): %w", rs.Primary.ID, err)
		}
		return nil
	}
}

// ─── pipeline environment ─────────────────────────────────────────────────────

func TestAccPipelineImageResource(t *testing.T) {
	t.Parallel()
	rn := "datarobot_pipeline_image.test"
	name := "penv-" + nameSalt
	var initialID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: accPipelineEnvConfig(name, []string{"numpy==1.26.4"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(rn, "id"),
					resource.TestCheckResourceAttr(rn, "name", name),
					resource.TestCheckResourceAttr(rn, "packages.#", "1"),
					resource.TestCheckResourceAttrSet(rn, "latest_version"),
					resource.TestCheckResourceAttrSet(rn, "latest_status"),
					captureAttr(rn, "id", &initialID),
					checkPipelineImageExistsInAPI(rn, name),
				),
			},
			{
				Config: accPipelineEnvConfig(name, []string{"numpy==1.26.4", "pandas>=2.0"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "packages.#", "2"),
					checkWorkloadIDPreserved(rn, &initialID),
					checkPipelineImageExistsInAPI(rn, name),
				),
			},
		},
	})
}

func TestAccPipelineImageReplaceOnNameChange(t *testing.T) {
	t.Parallel()
	rn := "datarobot_pipeline_image.test"
	name1 := "penv-a-" + nameSalt
	name2 := "penv-b-" + nameSalt
	var initialID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: accPipelineEnvConfig(name1, []string{"numpy==1.26.4"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "name", name1),
					captureAttr(rn, "id", &initialID),
				),
			},
			{
				Config: accPipelineEnvConfig(name2, []string{"numpy==1.26.4"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "name", name2),
					checkIDChanged(rn, &initialID),
				),
			},
		},
	})
}

func checkPipelineImageExistsInAPI(resourceName, expectedName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("pipeline environment ID is not set in state")
		}
		p, ok := testAccProvider.(*Provider)
		if !ok {
			return fmt.Errorf("provider not found")
		}
		p.service = NewService(cl)
		env, err := p.service.GetPipelineImage(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("GetPipelineImage(%s): %w", rs.Primary.ID, err)
		}
		if env.Name != expectedName {
			return fmt.Errorf("expected env name %q, got %q", expectedName, env.Name)
		}
		return nil
	}
}

// ─── pipeline input ───────────────────────────────────────────────────────────

func TestAccPipelineInputDraftResource(t *testing.T) {
	t.Parallel()
	rn := "datarobot_pipeline_input.test"
	srcFile := writeAccPipelineFile(t)
	var initialID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: accPipelineWithDraftInputConfig(srcFile, `{"param1":"value1"}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(rn, "id"),
					resource.TestCheckResourceAttr(rn, "state", string(client.PipelineInputStateValid)),
					captureAttr(rn, "id", &initialID),
				),
			},
			{
				Config: accPipelineWithDraftInputConfig(srcFile, `{"param1":"value2"}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkWorkloadIDPreserved(rn, &initialID),
				),
			},
		},
	})
}

// ─── pipeline schedule ────────────────────────────────────────────────────────

func TestAccPipelineScheduleResource(t *testing.T) {
	if os.Getenv("ACCEPTANCE_RUN_PIPELINES_SCHEDULING") == "" {
		t.Skip("schedule creation requires k8s CronJob wiring not yet deployed; set ACCEPTANCE_RUN_PIPELINES_SCHEDULING=1 to enable")
	}
	t.Parallel()
	rn := "datarobot_pipeline_schedule.test"
	srcFile := writeAccPipelineFile(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: accPipelineWithScheduleConfig(srcFile, "0 9 * * 1-5", "America/New_York"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(rn, "id"),
					resource.TestCheckResourceAttr(rn, "cron_expression", "0 9 * * 1-5"),
					resource.TestCheckResourceAttr(rn, "timezone", "America/New_York"),
					resource.TestCheckResourceAttrSet(rn, "pipeline_id"),
					resource.TestCheckResourceAttrSet(rn, "pipeline_input_id"),
				),
			},
			{
				Config: accPipelineWithScheduleConfig(srcFile, "0 10 * * 1-5", "America/Chicago"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "cron_expression", "0 10 * * 1-5"),
					resource.TestCheckResourceAttr(rn, "timezone", "America/Chicago"),
				),
			},
		},
	})
}

// ─── config helpers ───────────────────────────────────────────────────────────

func accPipelineDraftConfig(srcFile string) string {
	return fmt.Sprintf(`
resource "datarobot_pipeline" "test" {
  source_file = %q
  mode        = "draft"
}
`, srcFile)
}

func accPipelineLockedConfig(srcFile string) string {
	return fmt.Sprintf(`
resource "datarobot_pipeline" "test" {
  source_file = %q
  mode        = "locked"
}
`, srcFile)
}

func accPipelineLockedWithDescConfig(srcFile, desc string) string {
	return fmt.Sprintf(`
resource "datarobot_pipeline" "test" {
  source_file = %q
  mode        = "locked"
  description = %q
}
`, srcFile, desc)
}

func accPipelineEnvConfig(name string, pkgs []string) string {
	pkgList := ""
	for _, p := range pkgs {
		pkgList += fmt.Sprintf("    %q,\n", p)
	}
	return fmt.Sprintf(`
resource "datarobot_pipeline_image" "test" {
  name     = %q
  packages = [
%s  ]
}
`, name, pkgList)
}

func accPipelineWithDraftInputConfig(srcFile, payloadJSON string) string {
	return fmt.Sprintf(`
resource "datarobot_pipeline" "test" {
  source_file = %q
  mode        = "draft"
}

resource "datarobot_pipeline_input" "test" {
  pipeline_id = datarobot_pipeline.test.id
  payload     = %q
}
`, srcFile, payloadJSON)
}

func accPipelineWithScheduleConfig(srcFile, cron, timezone string) string {
	return fmt.Sprintf(`
resource "datarobot_pipeline" "test" {
  source_file = %q
  mode        = "locked"
}

resource "datarobot_pipeline_input" "test" {
  pipeline_id = datarobot_pipeline.test.id
  version     = datarobot_pipeline.test.current_version
  payload     = "{}"
}

resource "datarobot_pipeline_schedule" "test" {
  pipeline_id       = datarobot_pipeline.test.id
  version           = datarobot_pipeline.test.current_version
  pipeline_input_id = datarobot_pipeline_input.test.id
  cron_expression   = %q
  timezone          = %q
}
`, srcFile, cron, timezone)
}

// ─── utility ─────────────────────────────────────────────────────────────────

func writeAccPipelineFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "pipeline.py")
	// Combine nameSalt with a hash of the test name so parallel tests never share
	// a pipeline function name (the API rejects duplicate draft names with 409).
	salt := nameSalt[:8]
	testSalt := fmt.Sprintf("%s_%s", salt, sanitizePyIdent(t.Name()))
	content := fmt.Sprintf(`import datarobot as dr


@dr.task
def task_a_%s(x):
    return x


@dr.task
def task_b_%s(x):
    return x


@dr.pipeline
def pipeline_%s(x):
    return task_b_%s(task_a_%s(x))
`, testSalt, testSalt, testSalt, testSalt, testSalt)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("writeAccPipelineFile: %v", err)
	}
	return path
}

// sanitizePyIdent truncates s to 20 chars and replaces any non-alphanumeric rune with _.
func sanitizePyIdent(s string) string {
	if len(s) > 20 {
		s = s[len(s)-20:]
	}
	out := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			out[i] = c
		} else {
			out[i] = '_'
		}
	}
	return string(out)
}
