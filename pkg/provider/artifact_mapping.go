package provider

import (
	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func loadArtifactIntoDataSourceModel(artifact *client.Artifact, data *ArtifactDataSourceModel) {
	data.ArtifactID = types.StringValue(artifact.ID)
	data.Name = types.StringValue(artifact.Name)
	if artifact.Description != "" {
		data.Description = types.StringValue(artifact.Description)
	} else {
		data.Description = types.StringNull()
	}
	data.Type = types.StringValue(string(artifact.Type))
	data.Status = types.StringValue(string(artifact.Status))
	if artifact.Version != nil {
		data.Version = types.Int64Value(int64(*artifact.Version))
	} else {
		data.Version = types.Int64Null()
	}
	if artifact.ArtifactRepositoryID != nil {
		data.ArtifactRepositoryID = types.StringValue(*artifact.ArtifactRepositoryID)
	} else {
		data.ArtifactRepositoryID = types.StringNull()
	}
	if artifact.CreatedAt != "" {
		data.CreatedAt = types.StringValue(artifact.CreatedAt)
	} else {
		data.CreatedAt = types.StringNull()
	}
	if artifact.UpdatedAt != "" {
		data.UpdatedAt = types.StringValue(artifact.UpdatedAt)
	} else {
		data.UpdatedAt = types.StringNull()
	}
	data.Creator = loadArtifactCreatorFromAPI(artifact.Creator)
	data.Tags = loadArtifactTagsFromAPI(artifact.Tags)
	if len(artifact.Permissions) > 0 {
		data.Permissions = make([]types.String, len(artifact.Permissions))
		for i, p := range artifact.Permissions {
			data.Permissions[i] = types.StringValue(p)
		}
	} else {
		data.Permissions = []types.String{}
	}
	spec := loadArtifactSpecIntoDataSourceModel(artifact.Spec)
	data.Spec = &spec
}

func loadArtifactCreatorFromAPI(creator *client.ArtifactUser) *ArtifactCreatorModel {
	if creator == nil {
		return nil
	}
	return &ArtifactCreatorModel{
		ID:       types.StringValue(creator.ID),
		FullName: optionalStringValue(creator.FullName),
		Email:    optionalStringValue(creator.Email),
		Username: optionalStringValue(creator.Username),
		Userhash: optionalStringValue(creator.Userhash),
	}
}

func loadArtifactTagsFromAPI(tags []client.ArtifactTag) []ArtifactTagModel {
	if len(tags) == 0 {
		return []ArtifactTagModel{}
	}
	result := make([]ArtifactTagModel, len(tags))
	for i, tag := range tags {
		result[i] = ArtifactTagModel{
			ID:    types.StringValue(tag.ID),
			Name:  types.StringValue(tag.Name),
			Value: types.StringValue(tag.Value),
		}
	}
	return result
}

func loadArtifactSpecIntoDataSourceModel(spec client.ArtifactSpec) ArtifactSpecDataSourceModel {
	groups := make([]ArtifactContainerGroupDSModel, len(spec.ContainerGroups))
	for i, g := range spec.ContainerGroups {
		containers := make([]ArtifactContainerDSModel, len(g.Containers))
		for j, c := range g.Containers {
			containers[j] = loadContainerIntoDataSourceModel(c)
		}
		groups[i] = ArtifactContainerGroupDSModel{
			Name:       optionalStringValueFromString(g.Name),
			Containers: containers,
		}
	}
	model := ArtifactSpecDataSourceModel{
		ContainerGroups: groups,
		TemplateID:      optionalStringValue(spec.TemplateID),
	}
	if spec.Storage != nil {
		model.Storage = &ArtifactNimStorageModel{
			Mode:    types.StringValue(spec.Storage.Mode),
			PvcSize: optionalStringValue(spec.Storage.PvcSize),
		}
	}
	return model
}

func loadContainerIntoDataSourceModel(c client.ArtifactContainer) ArtifactContainerDSModel {
	model := ArtifactContainerDSModel{
		Name:        optionalStringValue(c.Name),
		ImageURI:    types.StringValue(c.ImageURI),
		Description: types.StringValue(c.Description),
	}
	if c.Primary != nil {
		model.Primary = types.BoolValue(*c.Primary)
	} else {
		model.Primary = types.BoolNull()
	}
	if c.Port != nil {
		model.Port = types.Int64Value(*c.Port)
	} else {
		model.Port = types.Int64Null()
	}
	if len(c.Entrypoint) > 0 {
		model.Entrypoint = make([]types.String, len(c.Entrypoint))
		for i, e := range c.Entrypoint {
			model.Entrypoint[i] = types.StringValue(e)
		}
	}
	if len(c.EnvironmentVars) > 0 {
		model.EnvironmentVars = make([]ArtifactEnvironmentVariableModel, len(c.EnvironmentVars))
		for i, ev := range c.EnvironmentVars {
			m := ArtifactEnvironmentVariableModel{
				Source:         types.StringValue(ev.Source),
				Name:           types.StringValue(ev.Name),
				Value:          types.StringNull(),
				DrCredentialID: types.StringNull(),
				Key:            types.StringNull(),
			}
			if ev.Source == client.EnvironmentVariableSourceCredential {
				m.DrCredentialID = types.StringValue(ev.DrCredentialID)
				m.Key = types.StringValue(ev.Key)
			} else {
				m.Value = types.StringValue(ev.Value)
			}
			model.EnvironmentVars[i] = m
		}
	} else {
		model.EnvironmentVars = []ArtifactEnvironmentVariableModel{}
	}
	model.StartupProbe = loadProbeFromAPI(c.StartupProbe)
	model.ReadinessProbe = loadProbeFromAPI(c.ReadinessProbe)
	model.LivenessProbe = loadProbeFromAPI(c.LivenessProbe)
	model.ImageBuildConfig = loadImageBuildConfigFromAPI(c.ImageBuildConfig)
	model.Build = loadContainerBuildFromAPI(c.Build)
	model.SecurityContext = loadSecurityContextFromAPI(c.SecurityContext)
	return model
}

func loadImageBuildConfigFromAPI(cfg *client.ArtifactImageBuildConfig) *ArtifactImageBuildConfigModel {
	if cfg == nil {
		return nil
	}
	model := &ArtifactImageBuildConfigModel{}
	if cfg.CodeRef != nil {
		model.CodeRef = &ArtifactCodeRefModel{
			Provider: types.StringValue(cfg.CodeRef.Provider),
			Type:     types.StringValue(cfg.CodeRef.Type),
			DataRobot: ArtifactDataRobotCodeModel{
				CatalogID:        types.StringValue(cfg.CodeRef.DataRobot.CatalogID),
				CatalogVersionID: types.StringValue(cfg.CodeRef.DataRobot.CatalogVersionID),
			},
		}
	}
	if cfg.Dockerfile != nil {
		df := cfg.Dockerfile
		dockerfileModel := &ArtifactDockerfileModel{
			Source:                        types.StringValue(df.Source),
			Path:                          types.StringValue(df.Path),
			ExecutionEnvironmentID:        types.StringValue(df.ExecutionEnvironmentID),
			ExecutionEnvironmentVersionID: types.StringValue(df.ExecutionEnvironmentVersionID),
		}
		if len(df.Entrypoint) > 0 {
			dockerfileModel.Entrypoint = make([]types.String, len(df.Entrypoint))
			for i, e := range df.Entrypoint {
				dockerfileModel.Entrypoint[i] = types.StringValue(e)
			}
		}
		model.Dockerfile = dockerfileModel
	}
	return model
}

func loadContainerBuildFromAPI(build *client.ArtifactContainerBuildInfo) *ArtifactContainerBuildModel {
	if build == nil {
		return nil
	}
	return &ArtifactContainerBuildModel{
		ArtifactImageBuildID: types.StringValue(build.ArtifactImageBuildID),
		Status:               types.StringValue(build.Status),
		CreatedAt:            types.StringValue(build.CreatedAt),
	}
}

func loadSecurityContextFromAPI(sc *client.ArtifactSecurityContext) *ArtifactSecurityContextModel {
	if sc == nil {
		return nil
	}
	model := &ArtifactSecurityContextModel{
		AllowPrivilegeEscalation: optionalBoolValue(sc.AllowPrivilegeEscalation),
		ReadOnlyRootFilesystem:   optionalBoolValue(sc.ReadOnlyRootFilesystem),
	}
	if sc.Capabilities != nil {
		capModel := &ArtifactCapabilitiesModel{}
		if len(sc.Capabilities.Add) > 0 {
			capModel.Add = make([]types.String, len(sc.Capabilities.Add))
			for i, c := range sc.Capabilities.Add {
				capModel.Add[i] = types.StringValue(c)
			}
		}
		if len(sc.Capabilities.Drop) > 0 {
			capModel.Drop = make([]types.String, len(sc.Capabilities.Drop))
			for i, c := range sc.Capabilities.Drop {
				capModel.Drop[i] = types.StringValue(c)
			}
		}
		model.Capabilities = capModel
	}
	if sc.SeccompProfile != nil {
		model.SeccompProfile = &ArtifactSeccompProfileModel{
			Type:             types.StringValue(sc.SeccompProfile.Type),
			LocalhostProfile: optionalStringValue(sc.SeccompProfile.LocalhostProfile),
		}
	}
	return model
}

func optionalStringValue(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

func optionalStringValueFromString(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func optionalBoolValue(b *bool) types.Bool {
	if b == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*b)
}
