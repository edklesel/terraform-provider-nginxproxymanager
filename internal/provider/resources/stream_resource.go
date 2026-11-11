// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	models "github.com/edklesel/terraform-provider-nginxproxymanager/internal/models"
	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
	utils "github.com/edklesel/terraform-provider-nginxproxymanager/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &streamResource{}
	_ resource.ResourceWithConfigure   = &streamResource{}
	_ resource.ResourceWithImportState = &streamResource{}
)

type streamResource struct {
	client *sdk.Client
}

func NewStreamResource() resource.Resource {
	return &streamResource{}
}

func (r *streamResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_stream"
}

func (r *streamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the proxy host.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"forwarding_host": schema.StringAttribute{
				Required:    true,
				Description: "Host to forward the stream to.",
			},
			"forwarding_port": schema.Int64Attribute{
				Required:    true,
				Description: "Port to forward the stream to.",
			},
			"incoming_port": schema.Int64Attribute{
				Required:    true,
				Description: "Port the request is coming in on.",
			},
			"tcp_forwarding": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to enable TCP Forwarding",
				Default:     booldefault.StaticBool(true),
			},
			"udp_forwarding": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to enable UDP Forwarding",
				Default:     booldefault.StaticBool(false),
			},
			"owner_user_id": schema.Int64Attribute{
				Computed:    true,
				Description: "ID of the user which owns the stream.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *streamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.StreamModel
	diags := req.Plan.Get(ctx, &plan)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var _stream = sdk.StreamBase{
		ForwardingHost: plan.ForwardingHost.ValueStringPointer(),
		ForwardingPort: utils.ReturnPointer(int(plan.ForwardingPort.ValueInt64())),
		IncomingPort:   utils.ReturnPointer(int(plan.IncomingPort.ValueInt64())),
		TCPForwarding:  plan.TCPForwarding.ValueBoolPointer(),
		UDPForwarding:  plan.UDPForwarding.ValueBoolPointer(),
	}

	stream, err := r.client.Streams.Create(_stream)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating stream.",
			fmt.Sprintf("Unable to create stream, unexpected error: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(stream.ID))
	plan.OwnerUserID = types.Int64Value(int64(stream.OwnerUserID))
	plan.ForwardingHost = types.StringValue(*stream.ForwardingHost)
	plan.ForwardingPort = types.Int64Value(int64(*stream.ForwardingPort))
	plan.IncomingPort = types.Int64Value(int64(*stream.IncomingPort))
	plan.TCPForwarding = types.BoolValue(*stream.TCPForwarding)
	plan.UDPForwarding = types.BoolValue(*stream.UDPForwarding)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *streamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	var state models.StreamModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to convert ID value %s to int type", state.ID),
			err.Error(),
		)
		return
	}

	stream, err := r.client.Streams.Get(id)
	if sdk.IsNotFound(err) {
		// Deleted outside Terraform, so plan to create it again
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read stream with ID %s.", state.ID.ValueString()),
			err.Error(),
		)
		return
	}

	state.ID = types.StringValue(strconv.Itoa(stream.ID))
	state.OwnerUserID = types.Int64Value(int64(stream.OwnerUserID))
	state.ForwardingHost = types.StringValue(*stream.ForwardingHost)
	state.ForwardingPort = types.Int64Value(int64(*stream.ForwardingPort))
	state.IncomingPort = types.Int64Value(int64(*stream.IncomingPort))
	state.TCPForwarding = types.BoolValue(*stream.TCPForwarding)
	state.UDPForwarding = types.BoolValue(*stream.UDPForwarding)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *streamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.StreamModel
	diags := req.Plan.Get(ctx, &plan)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(plan.ID.ValueString())
	if err != nil {
		diags.AddError(
			"Unable to parse ID from plan",
			fmt.Sprintf("Error received when parsing ID value from plan: %s", err),
		)
		return
	}

	var _stream = sdk.StreamBase{
		ForwardingHost: plan.ForwardingHost.ValueStringPointer(),
		ForwardingPort: utils.ReturnPointer(int(plan.ForwardingPort.ValueInt64())),
		IncomingPort:   utils.ReturnPointer(int(plan.IncomingPort.ValueInt64())),
		TCPForwarding:  plan.TCPForwarding.ValueBoolPointer(),
		UDPForwarding:  plan.UDPForwarding.ValueBoolPointer(),
	}

	stream, err := r.client.Streams.Update(id, _stream)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating stream.",
			fmt.Sprintf("Unable to update stream %d, unexpected error: %s", id, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(stream.ID))
	plan.OwnerUserID = types.Int64Value(int64(stream.OwnerUserID))
	plan.ForwardingHost = types.StringValue(*stream.ForwardingHost)
	plan.ForwardingPort = types.Int64Value(int64(*stream.ForwardingPort))
	plan.IncomingPort = types.Int64Value(int64(*stream.IncomingPort))
	plan.TCPForwarding = types.BoolValue(*stream.TCPForwarding)
	plan.UDPForwarding = types.BoolValue(*stream.UDPForwarding)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *streamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	var state models.StreamModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to convert ID value %s to int type", state.ID),
			err.Error(),
		)
		return
	}

	err = r.client.Streams.Delete(id)
	if err != nil && !sdk.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"Error deleting stream",
			fmt.Sprintf("Unable to delete stream, unexpected error: %s", err.Error()),
		)
	}
}

func (r *streamResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureProvider(req, resp)
}

func (r *streamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
