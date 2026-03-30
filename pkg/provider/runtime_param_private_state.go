package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// privateStateReader / privateStateWriter mirror the methods on the framework's
// internal *privatestate.ProviderData, avoiding an import of that package.
type privateStateReader interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
}

type privateStateWriter interface {
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

const runtimeParamPrivateKey = "runtime_param_state"

type runtimeParamAPIVersion string

const (
	runtimeParamAPIV1 runtimeParamAPIVersion = "v1"
	runtimeParamAPIV2 runtimeParamAPIVersion = "v2"
)

// runtimeParamPrivateState tracks the API version used to last apply runtime
// parameters and the keys the user explicitly declared.
// api_version keeps Updates on the original path to avoid silent param-removal.
// managed_keys lets Read filter out model-metadata.yaml injected params.
type runtimeParamPrivateState struct {
	APIVersion  runtimeParamAPIVersion `json:"api_version"`
	ManagedKeys []string               `json:"managed_keys"`
}

func loadRuntimeParamPrivateState(
	ctx context.Context,
	private privateStateReader,
) (*runtimeParamPrivateState, error) {
	raw, diags := private.GetKey(ctx, runtimeParamPrivateKey)
	if diags.HasError() {
		return nil, fmt.Errorf("reading runtime param private state: %s", diags.Errors()[0].Detail())
	}
	if len(raw) == 0 {
		// No private state yet — resource was imported or pre-dates this change.
		return nil, nil
	}
	var ps runtimeParamPrivateState
	if err := json.Unmarshal(raw, &ps); err != nil {
		return nil, fmt.Errorf("parsing runtime param private state: %w", err)
	}
	return &ps, nil
}

func saveRuntimeParamPrivateState(
	ctx context.Context,
	private privateStateWriter,
	ps runtimeParamPrivateState,
) error {
	raw, err := json.Marshal(ps)
	if err != nil {
		return fmt.Errorf("serialising runtime param private state: %w", err)
	}
	if diags := private.SetKey(ctx, runtimeParamPrivateKey, raw); diags.HasError() {
		return fmt.Errorf("writing runtime param private state: %s", diags.Errors()[0].Detail())
	}
	return nil
}

// managedKeysFromPlan returns the parameter keys declared in the plan config,
// stored in private state so Read() can filter to user-owned keys only.
func managedKeysFromPlan(ctx context.Context, plan CustomModelResourceModel) []string {
	params := make([]RuntimeParameterValue, 0)
	if IsKnown(plan.RuntimeParameterValues) {
		_ = plan.RuntimeParameterValues.ElementsAs(ctx, &params, false)
	}
	keys := make([]string, len(params))
	for i, p := range params {
		keys[i] = p.Key.ValueString()
	}
	return keys
}
