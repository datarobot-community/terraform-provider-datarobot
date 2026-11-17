package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"
)

// Exercise the file-sync/guard-update sequence with unchanged guard inputs. The
// guard endpoint must receive the inherited guards once, based on the new version.
func TestCustomModelGuardsRegeneratedAfterFileUpdate(t *testing.T) {
	for _, source := range []string{"folder_path", "files"} {
		t.Run(source, func(t *testing.T) {
			ctx := context.Background()
			service := mock_client.NewMockService(gomock.NewController(t))
			r := &CustomModelResource{provider: &Provider{service: service}}
			folder := t.TempDir()
			file := filepath.Join(folder, "custom.py")
			require.NoError(t, os.WriteFile(file, []byte("# updated model\n"), 0600))
			state := CustomModelResourceModel{
				ID:                  types.StringValue("model"),
				GuardConfigurations: []GuardConfiguration{{Name: types.StringValue("prompt tokens")}},
				OverallModerationConfiguration: &OverallModerationConfiguration{
					TimeoutSec: types.Int64Value(30), TimeoutAction: types.StringValue("block"),
				},
			}
			plan := state
			if source == "folder_path" {
				state.FolderPath = types.StringValue(folder)
				state.FolderPathHash = types.StringValue("old")
				plan.FolderPath = state.FolderPath
				plan.FolderPathHash = types.StringValue("new")
			} else {
				plan.Files = types.DynamicValue(types.TupleValueMust(
					[]attr.Type{types.StringType}, []attr.Value{types.StringValue(file)},
				))
				state.Files = plan.Files
				state.FilesHashes = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("old")})
				plan.FilesHashes = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("new")})
			}
			model := &client.CustomModel{ID: "model", LatestVersion: client.CustomModelVersion{
				ID: "old-version", BaseEnvironmentID: "env", BaseEnvironmentVersionID: "env-version",
				Items: []client.FileItem{
					{ID: "code", FileName: "custom.py", FileSource: "local"},
					{ID: "config", FileName: "moderation_config.yaml", FileSource: "local"},
				},
			}}
			inheritedGuard := client.GuardConfiguration{
				Name: "prompt tokens", Type: "ootb", OOTBType: "promptTokens", Stages: []string{"prompt"},
				Intervention: client.GuardIntervention{
					Action: "block", Message: "blocked", Conditions: []client.GuardCondition{{Comparator: "greaterThan", Comparand: 10}},
				},
			}
			gomock.InOrder(
				service.EXPECT().CreateCustomModelVersionCreateFromLatest(ctx, "model", &client.CreateCustomModelVersionFromLatestRequest{
					IsMajorUpdate: "false", BaseEnvironmentID: "env", BaseEnvironmentVersionID: "env-version", FilesToDelete: []string{"code", "config"},
				}).Return(&client.CustomModelVersion{}, nil),
				service.EXPECT().CreateCustomModelVersionFromFiles(ctx, "model", gomock.Any()).DoAndReturn(
					func(_ context.Context, _ string, req *client.CreateCustomModelVersionFromFilesRequest) (*client.CustomModelVersion, error) {
						require.Len(t, req.Files, 1)
						require.Equal(t, "# updated model\n", string(req.Files[0].Content))
						return &client.CustomModelVersion{ID: "files-version"}, nil
					}),
				service.EXPECT().GetCustomModel(ctx, "model").Return(&client.CustomModel{
					ID: "model", LatestVersion: client.CustomModelVersion{ID: "files-version"},
				}, nil),
				service.EXPECT().GetGuardConfigurationsForCustomModelVersion(ctx, "files-version").Return(
					&client.GuardConfigurationResponse{Data: []client.GuardConfiguration{inheritedGuard}}, nil),
				service.EXPECT().ListGuardTemplates(ctx).Return(nil, nil),
				service.EXPECT().CreateCustomModelVersionFromGuardConfigurations(ctx, "files-version", &client.CreateCustomModelVersionFromGuardsConfigurationRequest{
					CustomModelID: "model", Data: []client.GuardConfiguration{inheritedGuard},
					OverallConfig: client.OverallModerationConfiguration{TimeoutSec: 30, TimeoutAction: "block"},
				}).Return(&client.CreateCustomModelVersionFromGuardsConfigurationResponse{}, nil),
			)
			filesUpdated, err := r.updateLocalFiles(ctx, model, &state, plan)
			require.NoError(t, err)
			require.True(t, filesUpdated)
			guardsUpdated, err := r.updateGuardConfigurations(ctx, &state, plan, filesUpdated)
			require.NoError(t, err)
			require.True(t, guardsUpdated)
		})
	}
}

func TestCustomModelUnchangedGuardsSkipRegeneration(t *testing.T) {
	for _, tc := range []struct {
		name         string
		guards       []GuardConfiguration
		filesUpdated bool
	}{
		{"unchanged files", []GuardConfiguration{{Name: types.StringValue("guard")}}, false},
		{"no guards", nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &CustomModelResource{provider: &Provider{service: mock_client.NewMockService(gomock.NewController(t))}}
			state := CustomModelResourceModel{GuardConfigurations: tc.guards}
			created, err := r.updateGuardConfigurations(context.Background(), &state, state, tc.filesUpdated)
			require.NoError(t, err)
			require.False(t, created) // No service calls expected.
		})
	}
}

func TestCustomModelGuardRegenerationFailure(t *testing.T) {
	service := mock_client.NewMockService(gomock.NewController(t))
	r := &CustomModelResource{provider: &Provider{service: service}}
	state := CustomModelResourceModel{ID: types.StringValue("model"), GuardConfigurations: []GuardConfiguration{{Name: types.StringValue("guard")}}}
	service.EXPECT().GetCustomModel(gomock.Any(), "model").Return(&client.CustomModel{
		ID: "model", LatestVersion: client.CustomModelVersion{ID: "files-version"},
	}, nil)
	service.EXPECT().GetGuardConfigurationsForCustomModelVersion(gomock.Any(), "files-version").Return(&client.GuardConfigurationResponse{}, nil)
	service.EXPECT().ListGuardTemplates(gomock.Any()).Return(nil, nil)
	expected := errors.New("guard generation failed")
	service.EXPECT().CreateCustomModelVersionFromGuardConfigurations(gomock.Any(), "files-version", gomock.Any()).Return(nil, expected)
	created, err := r.updateGuardConfigurations(context.Background(), &state, state, true)
	require.ErrorIs(t, err, expected)
	require.False(t, created)
}

func TestAccCustomModelGuardsSurviveFileUpdate(t *testing.T) {
	testAccRequireTFAcc(t)
	for _, source := range []string{"folder_path", "files"} {
		t.Run(source, func(t *testing.T) {
			folder := t.TempDir()
			file := filepath.Join(folder, "custom.py")
			require.NoError(t, os.WriteFile(file, []byte("# initial model\n"), 0600))
			fileConfig := fmt.Sprintf("folder_path = %q", folder)
			if source == "files" {
				fileConfig = fmt.Sprintf("files = [[%q, \"custom.py\"]]", file)
			}
			config := fmt.Sprintf(`resource "datarobot_custom_model" "test" {
    name = "guards-survive-file-update-%s-%s"
    target_type = "TextGeneration"
    target_name = "response"
    language = "Python"
    base_environment_id = %q
    %s
    guard_configurations = [{
     template_name = "Prompt Tokens"
     name = "prompt tokens"
     stages = ["prompt"]
     intervention = {
      action = "block"
      message = "blocked"
      condition = jsonencode({comparator = "greaterThan", comparand = 10})
     }
    }]
   }`, nameSalt, source, testGenAIBaseEnvID, fileConfig)
			check := func(s *terraform.State) error {
				rs := s.RootModule().Resources["datarobot_custom_model.test"]
				if rs == nil {
					return fmt.Errorf("custom model missing from state")
				}
				service := client.NewService(cl)
				model, err := service.GetCustomModel(context.Background(), rs.Primary.ID)
				if err != nil {
					return err
				}
				for _, name := range []string{"custom.py", "moderation_config.yaml"} {
					found := false
					for _, item := range model.LatestVersion.Items {
						if item.FileName == name {
							found = true
						}
					}
					if !found {
						return fmt.Errorf("latest version %s is missing %s", model.LatestVersion.ID, name)
					}
				}
				guards, err := service.GetGuardConfigurationsForCustomModelVersion(context.Background(), model.LatestVersion.ID)
				if err != nil {
					return err
				}
				if len(guards.Data) != 1 || guards.Data[0].Name != "prompt tokens" {
					return fmt.Errorf("expected exactly one prompt tokens guard, got %+v", guards.Data)
				}
				return nil
			}
			resource.Test(t, resource.TestCase{
				PreCheck: func() { testAccPreCheck(t) }, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{Config: config, Check: check},
					{PreConfig: func() { require.NoError(t, os.WriteFile(file, []byte("# changed model\n"), 0600)) }, Config: config, Check: check},
					{Config: config, PlanOnly: true},
				},
			})
		})
	}
}
