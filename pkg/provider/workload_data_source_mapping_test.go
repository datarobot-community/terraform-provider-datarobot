package provider

import (
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestLoadWorkloadIntoDataSourceModel(t *testing.T) {
	t.Parallel()

	fullName := "Jane Doe"
	email := "jane@example.com"
	username := "jane"
	userhash := "abc123"
	artifactName := "my-artifact"
	artifactType := client.ArtifactTypeService
	artifactStatus := client.ArtifactStatusLocked
	version := 2
	artifactRepoID := "674a1b2c3d4e5f6789015000"
	templateID := "674a1b2c3d4e5f6789016000"
	protonID := "674a1b2c3d4e5f6789013000"
	artifactID := "674a1b2c3d4e5f6789014000"
	endpoint := "https://workloads.example.com/674a1b2c3d4e5f6789012345"
	lastResponse := "2026-01-16T11:59:30Z"
	lastRequestAt := "2026-01-16T11:59:00Z"
	replicaCount := int64(2)
	gpuCount := 0
	cpu := 1.0
	memory := int64(8589934592)

	workload := &client.Workload{
		ID:          "674a1b2c3d4e5f6789012345",
		Name:        "my-workload",
		Description: strPtr("test workload"),
		CreatedAt:   "2026-01-15T10:30:00Z",
		UpdatedAt:   "2026-01-16T12:00:00Z",
		Creator: &client.UserData{
			ID:       "674a1b2c3d4e5f6789012000",
			FullName: &fullName,
			Email:    &email,
			Username: &username,
			Userhash: &userhash,
		},
		ProtonID:   &protonID,
		ArtifactID: &artifactID,
		Artifact: &client.WorkloadArtifactInfo{
			ID:                   artifactID,
			Name:                 &artifactName,
			Type:                 &artifactType,
			Status:               &artifactStatus,
			Version:              &version,
			ArtifactRepositoryID: &artifactRepoID,
			TemplateID:           &templateID,
		},
		Type:   client.ArtifactTypeService,
		Status: client.ProtonStatusRunning,
		Replacement: &client.WorkloadReplacement{
			Status:             "in_progress",
			CandidateProtonIDs: []string{"674a1b2c3d4e5f6789017000"},
			Strategy:           "rolling",
		},
		Runtime: client.WorkloadRuntime{
			ContainerGroups: []client.GroupRuntime{
				{
					Name:                  "default",
					ReplicaCount:          &replicaCount,
					BundleSelectionPolicy: strPtr("availability"),
					ResourceBundles:       []string{"cpu.small"},
					Containers: []client.ContainerOverride{
						{
							Name: "main",
							ResourceAllocation: &client.ResourceAllocation{
								CPU:    &cpu,
								Memory: &memory,
							},
						},
					},
					ResolvedBundle: &client.ResolvedBundle{
						ID:          "cpu.small",
						CPUCount:    1.0,
						MemoryBytes: 8589934592,
						GPUCount:    &gpuCount,
					},
				},
			},
		},
		Permissions: []string{"CAN_VIEW", "CAN_UPDATE"},
		Importance:  client.WorkloadImportanceHigh,
		RequestStats: &client.RequestStats{
			TotalRequests:      100,
			ConcurrentRequests: 3,
			LastRequestAt:      &lastRequestAt,
			ResponseTime:       42,
			ErrorRate:          1.5,
			RequestRates:       []int{10, 12, 8, 9, 11, 7, 13},
			ErrorRates:         []int{0, 1, 0, 2, 1, 0, 1},
		},
		Tags: []client.TagInfo{
			{ID: "674a1b2c3d4e5f6789018000", Name: "env", Value: "staging"},
		},
		Endpoint:     &endpoint,
		LastResponse: &lastResponse,
		Owners: []client.UserData{
			{ID: "674a1b2c3d4e5f6789012000", Username: &username},
		},
	}

	var got WorkloadDataSourceModel
	loadWorkloadIntoDataSourceModel(workload, &got)

	want := WorkloadDataSourceModel{
		ID:          types.StringValue("674a1b2c3d4e5f6789012345"),
		Name:        types.StringValue("my-workload"),
		Description: types.StringValue("test workload"),
		CreatedAt:   types.StringValue("2026-01-15T10:30:00Z"),
		UpdatedAt:   types.StringValue("2026-01-16T12:00:00Z"),
		Creator: &WorkloadUserDataModel{
			ID:       types.StringValue("674a1b2c3d4e5f6789012000"),
			FullName: types.StringValue("Jane Doe"),
			Email:    types.StringValue("jane@example.com"),
			Username: types.StringValue("jane"),
			Userhash: types.StringValue("abc123"),
		},
		ProtonID:   types.StringValue(protonID),
		ArtifactID: types.StringValue(artifactID),
		Artifact: &WorkloadArtifactInfoModel{
			ID:                   types.StringValue(artifactID),
			Name:                 types.StringValue(artifactName),
			Type:                 types.StringValue("service"),
			Status:               types.StringValue("locked"),
			Version:              types.Int64Value(2),
			ArtifactRepositoryID: types.StringValue(artifactRepoID),
			TemplateID:           types.StringValue(templateID),
		},
		Type:   types.StringValue("service"),
		Status: types.StringValue("running"),
		Replacement: &WorkloadReplacementModel{
			Status:             types.StringValue("in_progress"),
			CandidateProtonIDs: []types.String{types.StringValue("674a1b2c3d4e5f6789017000")},
			Strategy:           types.StringValue("rolling"),
		},
		Runtime: &WorkloadDataSourceRuntimeModel{
			ContainerGroups: []WorkloadDataSourceGroupRuntimeModel{
				{
					Name:                  types.StringValue("default"),
					ReplicaCount:          types.Int64Value(2),
					BundleSelectionPolicy: types.StringValue("availability"),
					ResourceBundles:       []types.String{types.StringValue("cpu.small")},
					Autoscaling:           nil,
					Containers: []WorkloadContainerOverrideModel{
						{
							Name: types.StringValue("main"),
							ResourceAllocation: &WorkloadResourceAllocationModel{
								CPU:       types.Float64Value(1.0),
								GPU:       types.Float64Null(),
								GPUMemory: types.StringNull(),
								Memory:    types.StringValue("8589934592"),
							},
						},
					},
					ResolvedBundle: &WorkloadResolvedBundleModel{
						ID:           types.StringValue("cpu.small"),
						CPUCount:     types.Float64Value(1.0),
						MemoryBytes:  types.Int64Value(8589934592),
						GPUCount:     types.Int64Value(0),
						GPUMaker:     types.StringNull(),
						GPUTypeLabel: types.StringNull(),
					},
				},
			},
		},
		Permissions: []types.String{types.StringValue("CAN_VIEW"), types.StringValue("CAN_UPDATE")},
		Importance:  types.StringValue("high"),
		RequestStats: &WorkloadRequestStatsModel{
			TotalRequests:      types.Int64Value(100),
			ConcurrentRequests: types.Int64Value(3),
			LastRequestAt:      types.StringValue(lastRequestAt),
			ResponseTime:       types.Int64Value(42),
			ErrorRate:          types.Float64Value(1.5),
			RequestRates: []types.Int64{
				types.Int64Value(10), types.Int64Value(12), types.Int64Value(8),
				types.Int64Value(9), types.Int64Value(11), types.Int64Value(7),
				types.Int64Value(13),
			},
			ErrorRates: []types.Int64{
				types.Int64Value(0), types.Int64Value(1), types.Int64Value(0),
				types.Int64Value(2), types.Int64Value(1), types.Int64Value(0),
				types.Int64Value(1),
			},
		},
		Tags: []WorkloadTagModel{
			{
				ID:    types.StringValue("674a1b2c3d4e5f6789018000"),
				Name:  types.StringValue("env"),
				Value: types.StringValue("staging"),
			},
		},
		Endpoint:     types.StringValue(endpoint),
		LastResponse: types.StringValue(lastResponse),
		Owners: []WorkloadUserDataModel{
			{
				ID:       types.StringValue("674a1b2c3d4e5f6789012000"),
				FullName: types.StringNull(),
				Email:    types.StringNull(),
				Username: types.StringValue("jane"),
				Userhash: types.StringNull(),
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("loadWorkloadIntoDataSourceModel() mismatch (-want +got):\n%s", diff)
	}
}

func strPtr(s string) *string {
	return &s
}
