// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	models "github.com/edklesel/terraform-provider-nginxproxymanager/internal/models"
	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &streamDataSource{}
	_ datasource.DataSourceWithConfigure = &streamDataSource{}
)

type streamDataSource struct {
	client *sdk.Client
}

func NewStreamDataSource() datasource.DataSource {
	return &streamDataSource{}
}

func (r *streamDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_stream"
}

func (r *streamDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the stream host.",
			},
			"forwarding_host": schema.StringAttribute{
				Computed:    true,
				Description: "Host that the stream is forwarded to.",
			},
			"forwarding_port": schema.Int64Attribute{
				Computed:    true,
				Description: "Port that the stream is forwarded on.",
			},
			"incoming_port": schema.Int64Attribute{
				Computed:    true,
				Description: "Port the request is coming in on.",
			},
			"tcp_forwarding": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether TCP Forwarding is enabled.",
			},
			"udp_forwarding": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether UDP Forwarding is enabled.",
			},
			"enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the stream is enabled.",
			},
			"owner_user_id": schema.Int64Attribute{
				Computed:    true,
				Description: "ID of the user who owns the stream.",
			},
		},
	}
}

func (r *streamDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	var state models.StreamModel
	diags := req.Config.Get(ctx, &state)
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
	state.Enabled = types.BoolValue(stream.Enabled != nil && *stream.Enabled)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *streamDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	r.client = configureProvider(req, resp)
}
