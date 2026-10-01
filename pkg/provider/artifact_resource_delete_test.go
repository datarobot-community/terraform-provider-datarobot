package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/artifactsource/wapi"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	mock_client "github.com/datarobot-community/terraform-provider-datarobot/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	tfresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func artifactResourceConfigInRepository(name, status, repositoryID string) string {
	return fmt.Sprintf(`
resource "datarobot_artifact" "test" {
  name                   = %q
  description            = "test artifact description"
  type                   = "service"
  status                 = %q
  artifact_repository_id = %q
%s
}
`, name, status, repositoryID, artifactTestContainerSpecBlock(artifactTestImageURI))
}

// newArtifactDeleteMockService hooks a mock service into the provider for a resource.Test run.
func newArtifactDeleteMockService(t *testing.T) *mock_client.MockService {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	mockService := mock_client.NewMockService(ctrl)
	t.Cleanup(HookGlobal(&NewService, func(c *client.Client) client.Service {
		return mockService
	}))
	mockAPIKey(t)
	t.Setenv(DataRobotApiKeyEnvVar, "fake")
	return mockService
}

// expectNoArtifactRepositoryDelete fails the test if destroy deletes a repository. A
// missing expectation would fail it too, but gomock calls t.Fatalf from the provider's
// goroutine, which stalls the in-process provider server instead of failing the test.
func expectNoArtifactRepositoryDelete(t *testing.T, mockService *mock_client.MockService) {
	mockService.EXPECT().
		DeleteArtifactRepository(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string) error {
			t.Errorf("destroy deleted artifact repository %s, which this resource did not create", id)
			return nil
		}).AnyTimes()
}

// countArtifactVersionDeletes records DeleteArtifact calls instead of failing on them, for
// the same reason as expectNoArtifactRepositoryDelete.
func countArtifactVersionDeletes(mockService *mock_client.MockService) *[]string {
	var deleted []string
	mockService.EXPECT().
		DeleteArtifact(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string) error {
			deleted = append(deleted, id)
			return nil
		}).AnyTimes()
	return &deleted
}

// A repository named in the configuration was not created by the resource and can hold
// versions other clients put there, so destroy must never delete it. Only a draft version
// is deleted; the Workload API refuses to delete a locked one.
func TestIntegrationArtifactDestroyKeepsConfiguredRepository(t *testing.T) {
	for _, status := range []client.ArtifactStatus{client.ArtifactStatusLocked, client.ArtifactStatusDraft} {
		t.Run(string(status), func(t *testing.T) {
			mockService := newArtifactDeleteMockService(t)

			artifactID := uuid.NewString()
			repoID := uuid.NewString()
			name := "configured-repo-" + uuid.NewString()[:8]
			artifact := artifactFixtureWithStatus(artifactID, &repoID, name, status)
			cliVersion := client.Artifact{ID: uuid.NewString(), Status: client.ArtifactStatusLocked}

			mockService.EXPECT().
				CreateArtifact(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, req *client.CreateArtifactRequest) (*client.Artifact, error) {
					if req.ArtifactRepositoryID == nil || *req.ArtifactRepositoryID != repoID {
						t.Errorf("expected create in repository %s, got %v", repoID, req.ArtifactRepositoryID)
					}
					return artifact, nil
				})
			mockService.EXPECT().GetArtifact(gomock.Any(), artifactID).Return(artifact, nil).AnyTimes()
			mockService.EXPECT().
				ListArtifacts(gomock.Any(), &client.ListArtifactsRequest{RepositoryID: repoID}).
				Return([]client.Artifact{cliVersion}, nil).AnyTimes()
			expectNoArtifactRepositoryDelete(t, mockService)
			deleted := countArtifactVersionDeletes(mockService)

			resource.Test(t, resource.TestCase{
				IsUnitTest:               true,
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: artifactResourceConfigInRepository(name, string(status), repoID),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("datarobot_artifact.test", "artifact_repository_id", repoID),
							resource.TestCheckResourceAttr("datarobot_artifact.test", "created_artifact_repository_id", ""),
						),
					},
				},
			})

			want := []string{}
			if status == client.ArtifactStatusDraft {
				want = []string{artifactID}
			}
			if fmt.Sprint(*deleted) != fmt.Sprint(want) {
				t.Fatalf("destroy deleted artifact versions %v, want %v", *deleted, want)
			}
		})
	}
}

// A tainted object reaches Delete without the provider's private state, which is why
// ownership is a state attribute: replacing a tainted artifact in a configured
// repository must still leave the repository alone and create the new version in it.
func TestIntegrationArtifactTaintedReplaceKeepsConfiguredRepository(t *testing.T) {
	mockService := newArtifactDeleteMockService(t)

	repoID := uuid.NewString()
	name := "tainted-" + uuid.NewString()[:8]
	first := artifactFixtureWithStatus(uuid.NewString(), &repoID, name, client.ArtifactStatusLocked)
	second := artifactFixtureWithStatus(uuid.NewString(), &repoID, name, client.ArtifactStatusLocked)
	byID := map[string]*client.Artifact{first.ID: first, second.ID: second}

	created := []*client.Artifact{first, second}
	mockService.EXPECT().
		CreateArtifact(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *client.CreateArtifactRequest) (*client.Artifact, error) {
			if req.ArtifactRepositoryID == nil || *req.ArtifactRepositoryID != repoID {
				t.Errorf("expected create in repository %s, got %v", repoID, req.ArtifactRepositoryID)
			}
			next := created[0]
			created = created[1:]
			return next, nil
		}).Times(2)
	mockService.EXPECT().
		GetArtifact(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string) (*client.Artifact, error) {
			return byID[id], nil
		}).AnyTimes()
	mockService.EXPECT().ListArtifacts(gomock.Any(), gomock.Any()).Return([]client.Artifact{*first}, nil).AnyTimes()
	expectNoArtifactRepositoryDelete(t, mockService)
	deleted := countArtifactVersionDeletes(mockService)

	config := artifactResourceConfigInRepository(name, "locked", repoID)
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{
				Config: config,
				Taint:  []string{"datarobot_artifact.test"},
				Check:  resource.TestCheckResourceAttr("datarobot_artifact.test", "artifact_id", second.ID),
			},
		},
	})

	if len(*deleted) != 0 {
		t.Fatalf("destroy deleted locked artifact versions %v", *deleted)
	}
}

// An imported artifact may sit in a repository Terraform never created, so its destroy
// keeps the repository like a configured one does.
func TestIntegrationArtifactDestroyAfterImportKeepsRepository(t *testing.T) {
	mockService := newArtifactDeleteMockService(t)

	artifactID := uuid.NewString()
	repoID := uuid.NewString()
	name := "imported-" + uuid.NewString()[:8]
	artifact := artifactFixtureWithStatus(artifactID, &repoID, name, client.ArtifactStatusLocked)

	mockService.EXPECT().GetArtifact(gomock.Any(), artifactID).Return(artifact, nil).AnyTimes()
	mockService.EXPECT().ListArtifacts(gomock.Any(), gomock.Any()).Return([]client.Artifact{*artifact}, nil).AnyTimes()
	expectNoArtifactRepositoryDelete(t, mockService)
	deleted := countArtifactVersionDeletes(mockService)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             artifactResourceConfigWithStatus(name, "locked"),
				ResourceName:       "datarobot_artifact.test",
				ImportState:        true,
				ImportStateId:      artifactID,
				ImportStatePersist: true,
			},
		},
	})

	if len(*deleted) != 0 {
		t.Fatalf("destroy deleted locked artifact versions %v", *deleted)
	}
}

// The repository a resource created stays its to delete after the configuration moves
// the resource to another repository; only the version there is the resource's.
func TestIntegrationArtifactDestroyAfterMoveDeletesCreatedRepository(t *testing.T) {
	mockService := newArtifactDeleteMockService(t)

	createdRepo := uuid.NewString()
	sharedRepo := uuid.NewString()
	name := "moved-" + uuid.NewString()[:8]
	first := artifactFixture(uuid.NewString(), &createdRepo, name)
	second := artifactFixture(uuid.NewString(), &sharedRepo, name)
	byID := map[string]*client.Artifact{first.ID: first, second.ID: second}

	gomock.InOrder(
		mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).Return(first, nil),
		mockService.EXPECT().
			CreateArtifact(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, req *client.CreateArtifactRequest) (*client.Artifact, error) {
				if req.ArtifactRepositoryID == nil || *req.ArtifactRepositoryID != sharedRepo {
					t.Errorf("expected the new version in repository %s, got %v", sharedRepo, req.ArtifactRepositoryID)
				}
				return second, nil
			}),
	)
	mockService.EXPECT().
		GetArtifact(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string) (*client.Artifact, error) {
			return byID[id], nil
		}).AnyTimes()
	mockService.EXPECT().
		ListArtifacts(gomock.Any(), &client.ListArtifactsRequest{RepositoryID: sharedRepo}).
		Return([]client.Artifact{*second}, nil).AnyTimes()
	deletedRepos := []string{}
	mockService.EXPECT().
		DeleteArtifactRepository(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string) error {
			deletedRepos = append(deletedRepos, id)
			return nil
		}).AnyTimes()
	deleted := countArtifactVersionDeletes(mockService)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: artifactResourceConfigWithStatus(name, "locked"),
				Check:  resource.TestCheckResourceAttr("datarobot_artifact.test", "created_artifact_repository_id", createdRepo),
			},
			{
				Config: artifactResourceConfigInRepository(name, "locked", sharedRepo),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("datarobot_artifact.test", "artifact_repository_id", sharedRepo),
					resource.TestCheckResourceAttr("datarobot_artifact.test", "created_artifact_repository_id", createdRepo),
				),
			},
		},
	})

	if fmt.Sprint(deletedRepos) != fmt.Sprint([]string{createdRepo}) {
		t.Fatalf("destroy deleted repositories %v, want only %s", deletedRepos, createdRepo)
	}
	if len(*deleted) != 0 {
		t.Fatalf("destroy deleted locked artifact versions %v", *deleted)
	}
}

// testArtifactApplyDelete exercises the real Delete() implementation.
func testArtifactApplyDelete(t *testing.T, r *ArtifactResource, model ArtifactResourceModel) diag.Diagnostics {
	t.Helper()
	schema, diags := testArtifactResourceSchemaFor(r)
	if diags.HasError() {
		t.Fatalf("schema: %v", diags)
	}
	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("state: %v", diags)
	}
	resp := &tfresource.DeleteResponse{State: state}
	r.Delete(context.Background(), tfresource.DeleteRequest{State: state}, resp)
	return resp.Diagnostics
}

func artifactDeleteTestModel(artifact *client.Artifact, createdRepository types.String) ArtifactResourceModel {
	var model ArtifactResourceModel
	loadArtifactIntoModel(artifact, &model)
	model.ID = types.StringValue(uuid.NewString())
	model.CreatedArtifactRepositoryID = createdRepository
	return model
}

func diagWarningSummaries(diags diag.Diagnostics) []string {
	var out []string
	for _, d := range diags.Warnings() {
		out = append(out, d.Summary())
	}
	return out
}

// State written before created_artifact_repository_id existed has it null, and destroy
// keeps deleting the repository as earlier provider versions did.
func TestArtifactRepositoryOwnershipLegacyStateDeletesRepository(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)
	repoID := uuid.NewString()
	mockService.EXPECT().DeleteArtifactRepository(gomock.Any(), repoID).Return(nil)

	r := &ArtifactResource{provider: &Provider{service: mockService}}
	diags := testArtifactApplyDelete(t, r, artifactDeleteTestModel(artifactFixture(uuid.NewString(), &repoID, "legacy"), types.StringNull()))
	if diags.HasError() {
		t.Fatalf("delete: %v", diags)
	}
}

// The first plan after the upgrade records ownership, inferred from the configuration.
func TestArtifactRepositoryOwnershipInferredForLegacyState(t *testing.T) {
	repoID := uuid.NewString()
	for _, tc := range []struct {
		name       string
		configRepo types.String
		want       string
	}{
		{"configuration names the repository", types.StringValue(repoID), ""},
		{"configuration names none", types.StringNull(), repoID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := artifactDeleteTestModel(artifactFixture(uuid.NewString(), &repoID, "legacy"), types.StringNull())
			config := state
			config.ArtifactRepositoryID = tc.configRepo
			config.CreatedArtifactRepositoryID = types.StringNull()

			r := &ArtifactResource{provider: &Provider{}}
			plan, diags := testArtifactApplyModifyPlan(context.Background(), r, config, state, state)
			if diags.HasError() {
				t.Fatalf("modify plan: %v", diags)
			}
			if got := plan.CreatedArtifactRepositoryID; !got.Equal(types.StringValue(tc.want)) {
				t.Fatalf("created_artifact_repository_id = %s, want %q", got, tc.want)
			}
		})
	}
}

// A draft locked since the last refresh answers 409 to its delete. That is the locked
// refusal, so it takes the locked warning path rather than failing the destroy.
func TestArtifactRepositoryOwnershipStaleDraftLockedSinceRefresh(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)
	repoID := uuid.NewString()
	draft := artifactFixtureWithStatus(uuid.NewString(), &repoID, "stale", client.ArtifactStatusDraft)
	locked := artifactFixtureWithStatus(draft.ID, &repoID, "stale", client.ArtifactStatusLocked)

	mockService.EXPECT().DeleteArtifact(gomock.Any(), draft.ID).Return(client.NewGenericError("409 Conflict: Cannot delete locked artifact"))
	mockService.EXPECT().GetArtifact(gomock.Any(), draft.ID).Return(locked, nil)
	mockService.EXPECT().ListArtifacts(gomock.Any(), gomock.Any()).Return([]client.Artifact{*locked}, nil)

	r := &ArtifactResource{provider: &Provider{service: mockService}}
	diags := testArtifactApplyDelete(t, r, artifactDeleteTestModel(draft, types.StringValue("")))
	if diags.HasError() {
		t.Fatalf("delete failed: %v", diags)
	}
	if got := diagWarningSummaries(diags); fmt.Sprint(got) != "[Locked Artifact left in its repository]" {
		t.Fatalf("warnings = %v", got)
	}
}

// The Workload API answers 404 to a delete the principal may not make (it lacks the
// owner role on the repository). The artifact is still there, so that is an error.
func TestArtifactRepositoryOwnershipDeleteRefusedAsNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)
	repoID := uuid.NewString()
	draft := artifactFixtureWithStatus(uuid.NewString(), &repoID, "shared", client.ArtifactStatusDraft)

	mockService.EXPECT().DeleteArtifact(gomock.Any(), draft.ID).Return(&client.NotFoundError{Resource: draft.ID})
	mockService.EXPECT().GetArtifact(gomock.Any(), draft.ID).Return(draft, nil)

	r := &ArtifactResource{provider: &Provider{service: mockService}}
	diags := testArtifactApplyDelete(t, r, artifactDeleteTestModel(draft, types.StringValue("")))
	if !diags.HasError() || !strings.Contains(diags.Errors()[0].Detail(), "owner role") {
		t.Fatalf("expected an owner role error, got %v", diags)
	}
}

// Deleting the last live version also removes the repository, which the destroy says.
func TestArtifactRepositoryOwnershipLastDraftRemovesRepository(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)
	repoID := uuid.NewString()
	draft := artifactFixtureWithStatus(uuid.NewString(), &repoID, "last", client.ArtifactStatusDraft)

	mockService.EXPECT().DeleteArtifact(gomock.Any(), draft.ID).Return(nil)
	mockService.EXPECT().ListArtifacts(gomock.Any(), &client.ListArtifactsRequest{RepositoryID: repoID}).Return(nil, nil)

	r := &ArtifactResource{provider: &Provider{service: mockService}}
	diags := testArtifactApplyDelete(t, r, artifactDeleteTestModel(draft, types.StringValue("")))
	if diags.HasError() {
		t.Fatalf("delete: %v", diags)
	}
	if got := diagWarningSummaries(diags); fmt.Sprint(got) != "[Artifact repository removed with its last version]" {
		t.Fatalf("warnings = %v", got)
	}
}

// A locked version left in place must not keep its source directory bound to it, or
// re-creating the resource over that directory in another repository is refused.
func TestArtifactRepositoryOwnershipLockedVersionReleasesSourceDirectory(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)
	repoID := uuid.NewString()
	locked := artifactFixtureWithStatus(uuid.NewString(), &repoID, "bound", client.ArtifactStatusLocked)
	mockService.EXPECT().ListArtifacts(gomock.Any(), gomock.Any()).Return([]client.Artifact{*locked}, nil)

	dir := t.TempDir()
	if err := wapi.Initialize(dir, wapi.InitOptions{ArtifactID: locked.ID, CatalogID: artifactSourceTestCatalogID}); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	model := artifactDeleteTestModel(locked, types.StringValue(""))
	model.Source = &ArtifactSourceModel{
		Dir:            types.StringValue(dir),
		DirHash:        types.StringValue("hash"),
		GenerateIgnore: types.BoolValue(true),
		WaitForBuild:   types.BoolValue(true),
	}

	r := &ArtifactResource{provider: &Provider{service: mockService}}
	if diags := testArtifactApplyDelete(t, r, model); diags.HasError() {
		t.Fatalf("delete: %v", diags)
	}
	cfg, err := wapi.LoadConfig(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.ArtifactID != "" {
		t.Fatalf("directory still bound to %s", cfg.ArtifactID)
	}
}

// A create records the repository as its own only when the API made a new one, also when
// the configuration named a repository and the API put the artifact elsewhere.
func TestArtifactRepositoryOwnershipRecordedFromCreateResponse(t *testing.T) {
	named := "named-repo"
	other := "other-repo"
	for _, tc := range []struct {
		name      string
		requested *string
		got       *string
		want      string
	}{
		{"new repository", nil, &other, other},
		{"named repository used", &named, &named, ""},
		{"named repository ignored", &named, &other, other},
		{"no repository in response", nil, nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := createdArtifactRepository(&client.CreateArtifactRequest{ArtifactRepositoryID: tc.requested}, &client.Artifact{ArtifactRepositoryID: tc.got})
			if got != tc.want {
				t.Fatalf("createdArtifactRepository = %q, want %q", got, tc.want)
			}
		})
	}
}

// A build that outlives the wait leaves an artifact behind. It is saved to state
// (Terraform marks it tainted) instead of being lost together with its repository.
func TestArtifactRepositoryOwnershipKeptOnBuildTimeout(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockService := mock_client.NewMockService(ctrl)

	sourceDir := writeArtifactSourceTree(t, map[string]string{"main.py": "app"})
	artifactID := uuid.NewString()
	repoID := uuid.NewString()
	name := "source-build-timeout-" + uuid.NewString()[:8]
	draftArtifact := artifactFixtureDraftWithBuildConfig(artifactID, &repoID, name)
	patchedArtifact := artifactSourcePatchedArtifact(draftArtifact, artifactSourceTestCatalogID, artifactSourceTestVersionID)
	timeoutErr := &client.ArtifactBuildTimeoutError{ArtifactID: artifactID, BuildID: artifactSourceTestBuildID}

	mockService.EXPECT().CreateArtifact(gomock.Any(), gomock.Any()).Return(draftArtifact, nil)
	mockService.EXPECT().FilesAPI().Return(newSyncTestFilesAPI())
	mockService.EXPECT().PatchArtifactCodeRef(gomock.Any(), artifactID, gomock.Any(), gomock.Any()).Return(patchedArtifact, nil)
	mockService.EXPECT().
		TriggerArtifactBuild(gomock.Any(), artifactID).
		Return(&client.ArtifactBuildTriggerResponse{BuildIDs: []string{artifactSourceTestBuildID}}, nil)
	mockService.EXPECT().
		WaitForArtifactBuild(gomock.Any(), artifactID, artifactSourceTestBuildID, gomock.Any()).
		Return(nil, timeoutErr)
	mockService.EXPECT().BaseURL().Return("https://app.datarobot.com").AnyTimes()
	mockService.EXPECT().GetArtifactBuildLogs(gomock.Any(), artifactID, artifactSourceTestBuildID).Return("", nil).AnyTimes()

	r := &ArtifactResource{provider: &Provider{service: mockService}}
	result, diags := testArtifactApplyCreate(context.Background(), r, artifactResourceModelWithSource(name, sourceDir))
	if got := diagErrorSummary(diags); got != "Timeout waiting for artifact image build" {
		t.Fatalf("error summary = %q (%v)", got, diags)
	}
	if result.ArtifactID.ValueString() != artifactID {
		t.Fatalf("artifact_id in state = %s, want %s", result.ArtifactID, artifactID)
	}
	if result.CreatedArtifactRepositoryID.ValueString() != repoID {
		t.Fatalf("created_artifact_repository_id = %s, want %s", result.CreatedArtifactRepositoryID, repoID)
	}
}
