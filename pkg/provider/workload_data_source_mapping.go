package provider

import (
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func loadWorkloadIntoDataSourceModel(workload *client.Workload, data *WorkloadDataSourceModel) {
	data.ID = types.StringValue(workload.ID)
	data.Name = types.StringValue(workload.Name)
	data.Description = stringPtrToTF(workload.Description)
	data.CreatedAt = types.StringValue(workload.CreatedAt)
	data.UpdatedAt = types.StringValue(workload.UpdatedAt)
	data.Creator = loadUserDataModel(workload.Creator)
	data.ProtonID = stringPtrToTF(workload.ProtonID)
	data.ArtifactID = stringPtrToTF(workload.ArtifactID)
	data.Artifact = loadWorkloadArtifactInfoModel(workload.Artifact)
	data.Type = types.StringValue(string(workload.Type))
	data.Status = types.StringValue(string(workload.Status))
	data.Replacement = loadWorkloadReplacementModel(workload.Replacement)
	data.Runtime = loadWorkloadDataSourceRuntimeFromAPI(workload.Runtime)
	data.Permissions = loadPermissionsFromAPI(workload.Permissions)
	data.Importance = types.StringValue(string(workload.Importance))
	data.RequestStats = loadWorkloadRequestStatsModel(workload.RequestStats)
	data.Tags = loadWorkloadTagsFromAPI(workload.Tags)
	data.Endpoint = stringPtrToTF(workload.Endpoint)
	data.LastResponse = stringPtrToTF(workload.LastResponse)
	data.Owners = loadWorkloadOwnersFromAPI(workload.Owners)
}

func stringPtrToTF(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

func loadUserDataModel(user *client.UserData) *WorkloadUserDataModel {
	if user == nil {
		return nil
	}
	return &WorkloadUserDataModel{
		ID:       types.StringValue(user.ID),
		FullName: stringPtrToTF(user.FullName),
		Email:    stringPtrToTF(user.Email),
		Username: stringPtrToTF(user.Username),
		Userhash: stringPtrToTF(user.Userhash),
	}
}

func loadWorkloadArtifactInfoModel(artifact *client.WorkloadArtifactInfo) *WorkloadArtifactInfoModel {
	if artifact == nil {
		return nil
	}
	model := &WorkloadArtifactInfoModel{
		ID:                   types.StringValue(artifact.ID),
		Name:                 stringPtrToTF(artifact.Name),
		ArtifactRepositoryID: stringPtrToTF(artifact.ArtifactRepositoryID),
		TemplateID:           stringPtrToTF(artifact.TemplateID),
	}
	if artifact.Type != nil {
		model.Type = types.StringValue(string(*artifact.Type))
	} else {
		model.Type = types.StringNull()
	}
	if artifact.Status != nil {
		model.Status = types.StringValue(string(*artifact.Status))
	} else {
		model.Status = types.StringNull()
	}
	if artifact.Version != nil {
		model.Version = types.Int64Value(int64(*artifact.Version))
	} else {
		model.Version = types.Int64Null()
	}
	return model
}

func loadWorkloadReplacementModel(replacement *client.WorkloadReplacement) *WorkloadReplacementModel {
	if replacement == nil {
		return nil
	}
	candidateIDs := make([]types.String, len(replacement.CandidateProtonIDs))
	for i, id := range replacement.CandidateProtonIDs {
		candidateIDs[i] = types.StringValue(id)
	}
	return &WorkloadReplacementModel{
		Status:             types.StringValue(replacement.Status),
		CandidateProtonIDs: candidateIDs,
		Strategy:           types.StringValue(replacement.Strategy),
	}
}

func loadWorkloadRequestStatsModel(stats *client.RequestStats) *WorkloadRequestStatsModel {
	if stats == nil {
		return nil
	}
	requestRates := make([]types.Int64, len(stats.RequestRates))
	for i, rate := range stats.RequestRates {
		requestRates[i] = types.Int64Value(int64(rate))
	}
	errorRates := make([]types.Int64, len(stats.ErrorRates))
	for i, rate := range stats.ErrorRates {
		errorRates[i] = types.Int64Value(int64(rate))
	}
	return &WorkloadRequestStatsModel{
		TotalRequests:      types.Int64Value(int64(stats.TotalRequests)),
		ConcurrentRequests: types.Int64Value(int64(stats.ConcurrentRequests)),
		LastRequestAt:      stringPtrToTF(stats.LastRequestAt),
		ResponseTime:       types.Int64Value(int64(stats.ResponseTime)),
		ErrorRate:          types.Float64Value(stats.ErrorRate),
		RequestRates:       requestRates,
		ErrorRates:         errorRates,
	}
}

func loadPermissionsFromAPI(permissions []string) []types.String {
	if len(permissions) == 0 {
		return nil
	}
	result := make([]types.String, len(permissions))
	for i, p := range permissions {
		result[i] = types.StringValue(p)
	}
	return result
}

func loadWorkloadTagsFromAPI(tags []client.TagInfo) []WorkloadTagModel {
	if len(tags) == 0 {
		return nil
	}
	result := make([]WorkloadTagModel, len(tags))
	for i, tag := range tags {
		result[i] = WorkloadTagModel{
			ID:    types.StringValue(tag.ID),
			Name:  types.StringValue(tag.Name),
			Value: types.StringValue(tag.Value),
		}
	}
	return result
}

func loadWorkloadOwnersFromAPI(owners []client.UserData) []WorkloadUserDataModel {
	if len(owners) == 0 {
		return nil
	}
	result := make([]WorkloadUserDataModel, len(owners))
	for i, owner := range owners {
		if model := loadUserDataModel(&owner); model != nil {
			result[i] = *model
		}
	}
	return result
}

func loadWorkloadDataSourceRuntimeFromAPI(runtime client.WorkloadRuntime) *WorkloadDataSourceRuntimeModel {
	if len(runtime.ContainerGroups) == 0 {
		return nil
	}
	model := &WorkloadDataSourceRuntimeModel{
		ContainerGroups: make([]WorkloadDataSourceGroupRuntimeModel, len(runtime.ContainerGroups)),
	}
	for i, g := range runtime.ContainerGroups {
		model.ContainerGroups[i] = loadWorkloadDataSourceGroupRuntimeFromAPI(g)
	}
	return model
}

func loadWorkloadDataSourceGroupRuntimeFromAPI(g client.GroupRuntime) WorkloadDataSourceGroupRuntimeModel {
	name := g.Name
	if name == "" {
		name = "default"
	}
	m := WorkloadDataSourceGroupRuntimeModel{
		Name: types.StringValue(name),
	}
	if g.ReplicaCount != nil {
		m.ReplicaCount = types.Int64Value(*g.ReplicaCount)
	} else {
		m.ReplicaCount = types.Int64Null()
	}
	bundleSelectionPolicy := "availability"
	if g.BundleSelectionPolicy != nil {
		bundleSelectionPolicy = *g.BundleSelectionPolicy
	}
	m.BundleSelectionPolicy = types.StringValue(bundleSelectionPolicy)
	if g.Autoscaling != nil {
		autoscaling := &WorkloadAutoscalingModel{}
		if g.Autoscaling.Enabled != nil {
			autoscaling.Enabled = types.BoolValue(*g.Autoscaling.Enabled)
		} else {
			autoscaling.Enabled = types.BoolNull()
		}
		autoscaling.Policies = make([]WorkloadAutoscalingPolicyModel, len(g.Autoscaling.Policies))
		for i, p := range g.Autoscaling.Policies {
			policy := WorkloadAutoscalingPolicyModel{
				ScalingMetric: types.StringValue(p.ScalingMetric),
				Target:        types.Float64Value(p.Target),
				MinCount:      types.Int64Value(p.MinCount),
				MaxCount:      types.Int64Value(p.MaxCount),
			}
			if p.Priority != nil {
				policy.Priority = types.Int64Value(*p.Priority)
			} else {
				policy.Priority = types.Int64Null()
			}
			autoscaling.Policies[i] = policy
		}
		m.Autoscaling = autoscaling
	}
	if len(g.ResourceBundles) > 0 {
		m.ResourceBundles = make([]types.String, len(g.ResourceBundles))
		for i, rb := range g.ResourceBundles {
			m.ResourceBundles[i] = types.StringValue(rb)
		}
	}
	if len(g.Containers) > 0 {
		m.Containers = make([]WorkloadContainerOverrideModel, len(g.Containers))
		for i, c := range g.Containers {
			m.Containers[i] = loadContainerOverrideFromAPI(c)
		}
	}
	if g.ResolvedBundle != nil {
		rb := g.ResolvedBundle
		resolved := &WorkloadResolvedBundleModel{
			ID:          types.StringValue(rb.ID),
			CPUCount:    types.Float64Value(rb.CPUCount),
			MemoryBytes: types.Int64Value(rb.MemoryBytes),
		}
		if rb.GPUCount != nil {
			resolved.GPUCount = types.Int64Value(int64(*rb.GPUCount))
		} else {
			resolved.GPUCount = types.Int64Null()
		}
		resolved.GPUMaker = stringPtrToTF(rb.GPUMaker)
		resolved.GPUTypeLabel = stringPtrToTF(rb.GPUTypeLabel)
		m.ResolvedBundle = resolved
	}
	return m
}
