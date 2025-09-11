package data

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/datarobot-community/terraform-provider-datarobot/internal/client"
	"github.com/datarobot-community/terraform-provider-datarobot/internal/common"
	"github.com/datarobot-community/terraform-provider-datarobot/pkg/models"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure types fully satisfy framework interfaces.
var _ resource.Resource = &DatasetFromFileResource{}
var _ resource.ResourceWithImportState = &DatasetFromFileResource{}
var _ resource.ResourceWithModifyPlan = &DatasetFromFileResource{}

func NewDatasetFromFileResource() resource.Resource { return &DatasetFromFileResource{} }

type DatasetFromFileResource struct {
    service client.Service
}

func (r *DatasetFromFileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_dataset_from_file"
}

func (r *DatasetFromFileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        MarkdownDescription: "Dataset uploaded from a local file",
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{Computed: true, MarkdownDescription: "Dataset ID", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
            "file_path": schema.StringAttribute{Required: true, MarkdownDescription: "Path to the file to upload", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
            "file_hash": schema.StringAttribute{Computed: true, MarkdownDescription: "Hash of file contents"},
            "name": schema.StringAttribute{Optional: true, MarkdownDescription: "Dataset name; defaults to filename"},
            "use_case_ids": schema.ListAttribute{Optional: true, MarkdownDescription: "Use Case IDs to link", ElementType: types.StringType},
        },
    }
}

func (r *DatasetFromFileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil { return }
    accessor, ok := req.ProviderData.(common.ServiceAccessor)
    if !ok {
        resp.Diagnostics.AddError("Unexpected Provider Data Type", fmt.Sprintf("Expected ServiceAccessor, got %T", req.ProviderData))
        return
    }
    r.service = accessor.GetService()
}

func (r *DatasetFromFileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var data models.DatasetFromFileResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() { return }

    filePath := data.FilePath.ValueString()
    info, err := os.Stat(filePath)
    if err != nil { resp.Diagnostics.AddError("File stat error", err.Error()); return }
    fileName := info.Name()
    f, err := os.Open(filePath)
    if err != nil { resp.Diagnostics.AddError("File open error", err.Error()); return }
    defer f.Close()
    content, err := io.ReadAll(f)
    if err != nil { resp.Diagnostics.AddError("File read error", err.Error()); return }

    common.TraceAPICall("CreateDatasetFromFile")
    createResp, err := r.service.CreateDatasetFromFile(ctx, fileName, content)
    if err != nil { resp.Diagnostics.AddError("Create dataset error", err.Error()); return }

    if common.IsKnown(data.Name) {
        common.TraceAPICall("UpdateDataset")
        if _, err = r.service.UpdateDataset(ctx, createResp.ID, &client.UpdateDatasetRequest{Name: data.Name.ValueString()}); err != nil {
            resp.Diagnostics.AddError("Update dataset error", err.Error()); return }
    }

    dataset, err := common.WaitForDatasetToBeReady(ctx, r.service, createResp.ID)
    if err != nil { resp.Diagnostics.AddError("Dataset not ready", err.Error()); return }
    data.ID = types.StringValue(dataset.ID)

    for _, uc := range data.UseCaseIDs {
        common.TraceAPICall("AddDatasetToUseCase")
        if err = common.AddEntityToUseCase(ctx, r.service, uc.ValueString(), "dataset", dataset.ID); err != nil {
            resp.Diagnostics.AddError(fmt.Sprintf("Add dataset to use case %s", uc.ValueString()), err.Error()); return }
    }

    resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DatasetFromFileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var data models.DatasetFromFileResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() { return }
    if data.ID.IsNull() { return }
    common.TraceAPICall("GetDataset")
    if _, err := r.service.GetDataset(ctx, data.ID.ValueString()); err != nil {
        if _, ok := err.(*client.NotFoundError); ok {
            resp.Diagnostics.AddWarning("Dataset not found", fmt.Sprintf("Dataset %s not found; removing from state", data.ID.ValueString()))
            resp.State.RemoveResource(ctx); return
        }
        resp.Diagnostics.AddError("Get dataset error", err.Error())
        return
    }
    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatasetFromFileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var state models.DatasetFromFileResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() { return }
    var plan models.DatasetFromFileResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() { return }

    if common.IsKnown(plan.Name) {
        common.TraceAPICall("UpdateDataset")
        if _, err := r.service.UpdateDataset(ctx, plan.ID.ValueString(), &client.UpdateDatasetRequest{Name: plan.Name.ValueString()}); err != nil {
            resp.Diagnostics.AddError("Update dataset error", err.Error()); return }
    }

    // update use cases links
    stateUCs := make([]types.String, len(state.UseCaseIDs))
    copy(stateUCs, state.UseCaseIDs)
    planUCs := make([]types.String, len(plan.UseCaseIDs))
    copy(planUCs, plan.UseCaseIDs)
    if err := common.UpdateUseCasesForEntity(ctx, r.service, "dataset", plan.ID.ValueString(), stateUCs, planUCs); err != nil {
        resp.Diagnostics.AddError("Update use cases error", err.Error()); return }

    resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *DatasetFromFileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state models.DatasetFromFileResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() { return }
    common.TraceAPICall("DeleteDataset")
    if err := r.service.DeleteDataset(ctx, state.ID.ValueString()); err != nil {
        if !errors.Is(err, &client.NotFoundError{}) {
            resp.Diagnostics.AddError("Delete dataset error", err.Error())
        }
    }
}

func (r *DatasetFromFileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *DatasetFromFileResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
    if req.Plan.Raw.IsNull() { return } // destroy
    var plan models.DatasetFromFileResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...) ; if resp.Diagnostics.HasError() { return }
    // compute file hash
    hash, err := common.ComputeFileHash(plan.FilePath.ValueString())
    if err != nil { resp.Diagnostics.AddError("Compute file hash error", err.Error()); return }
    plan.FileHash = types.StringValue(hash)
    resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
    if req.State.Raw.IsNull() { return } // create
    var state models.DatasetFromFileResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...) ; if resp.Diagnostics.HasError() { return }
    if plan.FileHash != state.FileHash { resp.RequiresReplace.Append(path.Root("file_hash")) }
}
