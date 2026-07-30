package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func artifactCodeRefAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"catalog_id":         types.StringType,
		"catalog_version_id": types.StringType,
	}
}

func artifactCodeRefNull() types.Object {
	return types.ObjectNull(artifactCodeRefAttrTypes())
}

func codeRefObjectFromModel(ctx context.Context, ref *ArtifactCodeRefModel) (types.Object, diag.Diagnostics) {
	if ref == nil {
		return artifactCodeRefNull(), nil
	}
	if ref.CatalogID.IsUnknown() || ref.CatalogVersionID.IsUnknown() {
		return types.ObjectUnknown(artifactCodeRefAttrTypes()), nil
	}
	if ref.CatalogID.IsNull() && ref.CatalogVersionID.IsNull() {
		return artifactCodeRefNull(), nil
	}
	obj, diags := types.ObjectValue(artifactCodeRefAttrTypes(), map[string]attr.Value{
		"catalog_id":         ref.CatalogID,
		"catalog_version_id": ref.CatalogVersionID,
	})
	return obj, diags
}

func codeRefModelFromObject(ctx context.Context, obj types.Object) (*ArtifactCodeRefModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if obj.IsNull() || obj.IsUnknown() {
		return nil, diags
	}

	var ref ArtifactCodeRefModel
	diags.Append(obj.As(ctx, &ref, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}
	return &ref, diags
}

func imageBuildConfigCodeRef(cfg *ArtifactImageBuildConfigModel) *ArtifactCodeRefModel {
	if cfg == nil || cfg.CodeRef.IsNull() || cfg.CodeRef.IsUnknown() {
		return nil
	}
	ref, diags := codeRefModelFromObject(context.Background(), cfg.CodeRef)
	if diags.HasError() {
		return nil
	}
	return ref
}

func setImageBuildConfigCodeRef(cfg *ArtifactImageBuildConfigModel, ref *ArtifactCodeRefModel) diag.Diagnostics {
	if cfg == nil {
		return nil
	}
	obj, diags := codeRefObjectFromModel(context.Background(), ref)
	if diags.HasError() {
		return diags
	}
	cfg.CodeRef = obj
	return nil
}

func artifactCodeRefObject(ref *ArtifactCodeRefModel) types.Object {
	obj, diags := codeRefObjectFromModel(context.Background(), ref)
	if diags.HasError() {
		panic(diags.Errors()[0].Summary())
	}
	return obj
}

func imageBuildConfigWithCodeRef(ref *ArtifactCodeRefModel) *ArtifactImageBuildConfigModel {
	cfg := &ArtifactImageBuildConfigModel{}
	_ = setImageBuildConfigCodeRef(cfg, ref)
	return cfg
}
