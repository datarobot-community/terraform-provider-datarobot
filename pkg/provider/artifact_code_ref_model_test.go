package provider

import (
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCodeRefObjectRoundtrip(t *testing.T) {
	t.Parallel()

	ref := &ArtifactCodeRefModel{
		CatalogID:        types.StringValue("aaaaaaaaaaaaaaaaaaaaaaaa"),
		CatalogVersionID: types.StringValue("bbbbbbbbbbbbbbbbbbbbbbbb"),
	}
	cfg := &ArtifactImageBuildConfigModel{CodeRef: artifactCodeRefNull()}
	if diags := setImageBuildConfigCodeRef(cfg, ref); diags.HasError() {
		t.Fatal(diags)
	}

	got := imageBuildConfigCodeRef(cfg)
	if got == nil {
		t.Fatal("expected code_ref after roundtrip")
	}
	if got.CatalogID.ValueString() != ref.CatalogID.ValueString() {
		t.Fatalf("catalog_id = %q", got.CatalogID.ValueString())
	}
}

func TestLoadImageBuildConfigFromAPISetsCodeRefObject(t *testing.T) {
	t.Parallel()

	model := loadImageBuildConfigFromAPI(&client.ArtifactImageBuildConfig{
		CodeRef: &client.ArtifactCodeRef{
			Type: "datarobot",
			DataRobot: client.ArtifactDataRobotCodeRef{
				CatalogID:        "aaaaaaaaaaaaaaaaaaaaaaaa",
				CatalogVersionID: "bbbbbbbbbbbbbbbbbbbbbbbb",
			},
		},
	}, nil)

	ref := imageBuildConfigCodeRef(model)
	if ref == nil {
		t.Fatal("expected code_ref from API")
	}
}
