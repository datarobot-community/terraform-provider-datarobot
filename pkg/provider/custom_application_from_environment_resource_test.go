package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccCustomApplicationFromEnvironmentResource(t *testing.T) {
	t.Parallel()

	// TODO: enable this test once Execution Environments don't take forever to create
	t.Skip("Skipping registered model from leaderboard test for environment")

	if !strings.Contains(globalTestCfg.Endpoint, "staging") {
		t.Skip("Skipping custom application from environment test")
	}

	resourceName := "datarobot_custom_application_from_environment.test"

	compareValuesDiffer := statecheck.CompareValue(compare.ValuesDiffer())

	name := "custom_app_from_env" + nameSalt
	newName := "new_custom_app_from_env " + nameSalt

	environmentID := testCustomAppEnvID
	environmentID2 := testCustomAppEnvID2

	useCaseResourceName := "test_custom_application"
	useCaseResourceName2 := "test_new_custom_application"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: customApplicationFromEnvironmentResourceConfig(
					name,
					environmentID,
					false,
					[]string{},
					&useCaseResourceName),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("use_case_ids"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomApplicationFromEnvironmentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "environment_id", environmentID),
					resource.TestCheckResourceAttrSet(resourceName, "environment_version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "environment_id"),
					resource.TestCheckResourceAttrSet(resourceName, "application_url"),
					resource.TestCheckResourceAttr(resourceName, "external_access_enabled", "false"),
					resource.TestCheckNoResourceAttr(resourceName, "external_access_recipients"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
				),
			},
			// Update name, external access, and use case id
			{
				Config: customApplicationFromEnvironmentResourceConfig(
					newName,
					environmentID,
					true,
					[]string{"test@test.com"},
					&useCaseResourceName2),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("use_case_ids"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomApplicationFromEnvironmentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "environment_id", environmentID),
					resource.TestCheckResourceAttrSet(resourceName, "environment_version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "environment_id"),
					resource.TestCheckResourceAttrSet(resourceName, "application_url"),
					resource.TestCheckResourceAttr(resourceName, "external_access_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "external_access_recipients.0", "test@test.com"),
					resource.TestCheckResourceAttrSet(resourceName, "use_case_ids.0"),
				),
			},
			// Update environment id triggers replace
			{
				Config: customApplicationFromEnvironmentResourceConfig(
					newName,
					environmentID2,
					true,
					[]string{"test2@test.com"},
					nil),
				ConfigStateChecks: []statecheck.StateCheck{
					compareValuesDiffer.AddStateValue(
						resourceName,
						tfjsonpath.New("id"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkCustomApplicationFromEnvironmentResourceExists(),
					resource.TestCheckResourceAttr(resourceName, "name", newName),
					resource.TestCheckResourceAttr(resourceName, "environment_id", environmentID2),
					resource.TestCheckResourceAttrSet(resourceName, "environment_version_id"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "application_url"),
					resource.TestCheckResourceAttr(resourceName, "external_access_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "external_access_recipients.0", "test2@test.com"),
					resource.TestCheckNoResourceAttr(resourceName, "use_case_ids.0"),
				),
			},
			// Delete is tested automatically
		},
	})
}

func customApplicationFromEnvironmentResourceConfig(
	name string,
	environmentID string,
	externalAccess bool,
	externalAccessRecipients []string,
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
resource "datarobot_use_case" "test_custom_application_from_environment" {
	name = "test custom application from env"
}
resource "datarobot_use_case" "test_new_custom_application_from_environment" {
	name = "test new custom application from env"
}

resource "datarobot_custom_application_from_environment" "test" {
	environment_id = "%s"
	external_access_enabled = %t
	%s
	%s
	%s
}
`, environmentID, externalAccess, recipients, nameStr, useCaseIDsStr)
}

func checkCustomApplicationFromEnvironmentResourceExists() resource.TestCheckFunc {
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
			application.EnvVersionID == rs.Primary.Attributes["environment_version_id"] {
			b, err := strconv.ParseBool(rs.Primary.Attributes["external_access_enabled"])
			if err == nil {
				if application.ExternalAccessEnabled == b {
					if len(application.ExternalAccessRecipients) > 0 {
						if application.ExternalAccessRecipients[0] == rs.Primary.Attributes["external_access_recipients.0"] {
							return nil
						}
					} else if rs.Primary.Attributes["external_access_recipients.0"] == "" {
						return nil
					}
				}
			}
			return nil
		}

		return fmt.Errorf("Custom Application not found")
	}
}

func customApplicationFromEnvironmentSchema(t *testing.T) fwresource.SchemaResponse {
	t.Helper()

	resp := fwresource.SchemaResponse{}
	NewCustomApplicationFromEnvironmentResource().Schema(context.Background(), fwresource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", resp.Diagnostics)
	}
	return resp
}

func TestCustomApplicationFromEnvironmentResourceModelMatchesSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	resp := customApplicationFromEnvironmentSchema(t)

	// Every field of the model needs an attribute in the schema, or the framework rejects the plan.
	plan := tfsdk.Plan{Schema: resp.Schema, Raw: tftypes.NewValue(resp.Schema.Type().TerraformType(ctx), nil)}
	model := CustomApplicationFromEnvironmentResourceModel{
		ExternalAccessRecipients: types.ListNull(types.StringType),
		Resources:                types.ObjectNull(applicationResourcesAttrTypes()),
	}
	if diags := plan.Set(ctx, &model); diags.HasError() {
		t.Fatalf("the model has fields that the schema lacks: %+v", diags)
	}
}

func TestCustomApplicationFromEnvironmentCreateSendsRequiredKeyScopeLevel(t *testing.T) {
	cases := []struct {
		name  string
		level types.String
		want  client.ScopeLevel
	}{
		{"set", types.StringValue("admin"), client.ScopeLevel("admin")},
		{"unset", types.StringNull(), client.ScopeLevel("")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			applicationID := "app-id"
			application := &client.Application{ID: applicationID, Status: "running"}
			mockService := mock_client.NewMockService(ctrl)
			var sent *client.CreateCustomApplicationRequest
			mockService.EXPECT().
				CreateCustomApplication(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, req *client.CreateCustomApplicationRequest) (*client.Application, error) {
					sent = req
					return application, nil
				})
			mockService.EXPECT().UpdateApplication(gomock.Any(), applicationID, gomock.Any()).Return(application, nil).AnyTimes()
			mockService.EXPECT().GetApplication(gomock.Any(), applicationID).Return(application, nil).AnyTimes()

			r := &CustomApplicationFromEnvironmentResource{provider: &Provider{service: mockService}}
			resp := customApplicationFromEnvironmentSchema(t)
			plan := tfsdk.Plan{Schema: resp.Schema, Raw: tftypes.NewValue(resp.Schema.Type().TerraformType(ctx), nil)}
			model := CustomApplicationFromEnvironmentResourceModel{
				Name:                     types.StringValue("app"),
				EnvironmentID:            types.StringValue("env-id"),
				ExternalAccessRecipients: types.ListNull(types.StringType),
				Resources:                types.ObjectNull(applicationResourcesAttrTypes()),
				RequiredKeyScopeLevel:    tc.level,
			}
			if diags := plan.Set(ctx, &model); diags.HasError() {
				t.Fatalf("set plan: %+v", diags)
			}

			createResp := &fwresource.CreateResponse{State: tfsdk.State{
				Schema: resp.Schema,
				Raw:    tftypes.NewValue(resp.Schema.Type().TerraformType(ctx), nil),
			}}
			r.Create(ctx, fwresource.CreateRequest{Plan: plan}, createResp)

			if createResp.Diagnostics.HasError() {
				t.Fatalf("create diagnostics: %+v", createResp.Diagnostics)
			}
			if sent == nil || sent.RequiredKeyScopeLevel != tc.want {
				t.Fatalf("request scope level = %+v, want %q", sent, tc.want)
			}
		})
	}
}
