package provider

import (
	"testing"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Optional+Computed compute settings must resolve to null (not stay unknown) when the API omits them.
func TestLoadCustomModelFromVectorDatabaseToState(t *testing.T) {
	t.Run("api omits optional computed fields and they were unknown -> null", func(t *testing.T) {
		state := CustomModelFromVectorDatabaseResourceModel{
			Replicas:            types.Int64Unknown(),
			NetworkEgressPolicy: types.StringUnknown(),
			ResourceBundleID:    types.StringUnknown(),
			MemoryMB:            types.Int64Unknown(),
		}
		cm := client.CustomModel{
			ID:          "cm-1",
			Name:        "model",
			Description: "",
			LatestVersion: client.CustomModelVersion{
				ID: "ver-1",
			},
		}

		loadCustomModelFromVectorDatabaseToState(cm, &state)

		if state.ID.ValueString() != "cm-1" {
			t.Errorf("ID = %q, want cm-1", state.ID.ValueString())
		}
		if state.VersionID.ValueString() != "ver-1" {
			t.Errorf("VersionID = %q, want ver-1", state.VersionID.ValueString())
		}
		if !state.Description.IsNull() {
			t.Errorf("Description = %v, want null", state.Description)
		}
		for name, v := range map[string]interface{ IsNull() bool }{
			"replicas":              state.Replicas,
			"network_egress_policy": state.NetworkEgressPolicy,
			"resource_bundle_id":    state.ResourceBundleID,
			"memory_mb":             state.MemoryMB,
		} {
			if !v.IsNull() {
				t.Errorf("%s = %v, want null (must not stay unknown after apply)", name, v)
			}
		}
		if state.Replicas.IsUnknown() || state.MemoryMB.IsUnknown() {
			t.Errorf("compute settings must not remain unknown after apply")
		}
	})

	t.Run("api returns values -> populated", func(t *testing.T) {
		state := CustomModelFromVectorDatabaseResourceModel{
			Replicas:            types.Int64Unknown(),
			NetworkEgressPolicy: types.StringUnknown(),
			ResourceBundleID:    types.StringUnknown(),
			MemoryMB:            types.Int64Unknown(),
		}
		replicas := int64(3)
		egress := "PUBLIC"
		bundle := "cpu.micro"
		mem := float64(2048 * 1024 * 1024) // 2048 MB in bytes
		cm := client.CustomModel{
			ID:          "cm-2",
			Description: "a model",
			LatestVersion: client.CustomModelVersion{
				ID:                  "ver-2",
				Replicas:            &replicas,
				NetworkEgressPolicy: &egress,
				ResourceBundleID:    &bundle,
				MaximumMemory:       &mem,
			},
		}

		loadCustomModelFromVectorDatabaseToState(cm, &state)

		if state.Description.ValueString() != "a model" {
			t.Errorf("Description = %q, want 'a model'", state.Description.ValueString())
		}
		if state.Replicas.ValueInt64() != 3 {
			t.Errorf("Replicas = %d, want 3", state.Replicas.ValueInt64())
		}
		if state.NetworkEgressPolicy.ValueString() != "PUBLIC" {
			t.Errorf("NetworkEgressPolicy = %q, want PUBLIC", state.NetworkEgressPolicy.ValueString())
		}
		if state.ResourceBundleID.ValueString() != "cpu.micro" {
			t.Errorf("ResourceBundleID = %q, want cpu.micro", state.ResourceBundleID.ValueString())
		}
		if state.MemoryMB.ValueInt64() != 2048 {
			t.Errorf("MemoryMB = %d, want 2048", state.MemoryMB.ValueInt64())
		}
	})

	t.Run("user-set value preserved when api omits it", func(t *testing.T) {
		// User-set (known) value is kept even when the API omits it.
		state := CustomModelFromVectorDatabaseResourceModel{
			Replicas:            types.Int64Value(5),
			NetworkEgressPolicy: types.StringUnknown(),
			ResourceBundleID:    types.StringUnknown(),
			MemoryMB:            types.Int64Unknown(),
		}
		cm := client.CustomModel{ID: "cm-3", LatestVersion: client.CustomModelVersion{ID: "ver-3"}}

		loadCustomModelFromVectorDatabaseToState(cm, &state)

		if state.Replicas.ValueInt64() != 5 {
			t.Errorf("Replicas = %v, want preserved value 5", state.Replicas)
		}
		if !state.NetworkEgressPolicy.IsNull() {
			t.Errorf("NetworkEgressPolicy = %v, want null", state.NetworkEgressPolicy)
		}
	})
}

// Custom-chunking VDB -> built-in chunker fields null; built-in chunking -> concrete values.
func TestLoadVectorDatabaseToTerraformState_Chunking(t *testing.T) {
	t.Run("custom chunking -> built-in fields null", func(t *testing.T) {
		vdb := &client.VectorDatabase{
			ID:             "vdb-1",
			Name:           "vdb",
			DatasetID:      "ds-1",
			UseCaseID:      "uc-1",
			EmbeddingModel: "intfloat/e5-base-v2",
			CustomChunking: true,
			// API reports these empty for a custom-chunking VDB:
			ChunkSize:      0,
			ChunkingMethod: "",
			Separators:     nil,
		}
		var data VectorDatabaseResourceModel

		loadVectorDatabaseToTerraformState(vdb, &data)

		cp := data.ChunkingParameters
		if cp == nil {
			t.Fatal("ChunkingParameters is nil")
		}
		if !cp.CustomChunking.ValueBool() {
			t.Errorf("CustomChunking = false, want true")
		}
		if !cp.ChunkSize.IsNull() {
			t.Errorf("ChunkSize = %v, want null", cp.ChunkSize)
		}
		if !cp.ChunkingMethod.IsNull() {
			t.Errorf("ChunkingMethod = %v, want null", cp.ChunkingMethod)
		}
		if !cp.Separators.IsNull() {
			t.Errorf("Separators = %v, want null", cp.Separators)
		}
	})

	t.Run("built-in chunking -> fields populated", func(t *testing.T) {
		vdb := &client.VectorDatabase{
			ID:             "vdb-2",
			Name:           "vdb",
			DatasetID:      "ds-2",
			UseCaseID:      "uc-2",
			EmbeddingModel: "intfloat/e5-base-v2",
			CustomChunking: false,
			ChunkSize:      500,
			ChunkingMethod: "recursive",
			Separators:     []string{"\n", " "},
		}
		var data VectorDatabaseResourceModel

		loadVectorDatabaseToTerraformState(vdb, &data)

		cp := data.ChunkingParameters
		if cp == nil {
			t.Fatal("ChunkingParameters is nil")
		}
		if cp.ChunkSize.ValueInt64() != 500 {
			t.Errorf("ChunkSize = %d, want 500", cp.ChunkSize.ValueInt64())
		}
		if cp.ChunkingMethod.ValueString() != "recursive" {
			t.Errorf("ChunkingMethod = %q, want recursive", cp.ChunkingMethod.ValueString())
		}
		if cp.Separators.IsNull() || len(cp.Separators.Elements()) != 2 {
			t.Errorf("Separators = %v, want 2 elements", cp.Separators)
		}
	})
}
