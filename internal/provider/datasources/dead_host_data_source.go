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
	_ datasource.DataSource              = &deadHostDataSource{}
	_ datasource.DataSourceWithConfigure = &deadHostDataSource{}
)

type deadHostDataSource struct {
	client *sdk.Client
}

func NewDeadHostDataSource() datasource.DataSource {
	return &deadHostDataSource{}
}

func (r *deadHostDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dead_host"
}

func (r *deadHostDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the dead host.",
			},
			"domain_names": schema.ListAttribute{
				Computed:    true,
				Description: "Domain names accepted.",
				ElementType: types.StringType,
			},
			"certificate_id": schema.Int64Attribute{
				Computed:    true,
				Description: "ID of the certificate used.",
			},
			"ssl_forced": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether SSL is forced.",
			},
			"http2_support": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether HTTP2 support is enabled.",
			},
			"enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the dead host is enabled.",
			},
			"hsts_enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether HSTS is enabled.",
			},
			"hsts_subdomains": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether HSTS is enabled for subdomains.",
			},
			"advanced_config": schema.StringAttribute{
				Computed:    true,
				Description: "Advanced NGINX config.",
			},
			"owner_user_id": schema.Int64Attribute{
				Computed:    true,
				Description: "ID of the user who owns the dead host.",
			},
		},
	}
}

func (r *deadHostDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	var state models.DeadHostModel
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

	deadHost, err := r.client.DeadHosts.Get(id)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read dead host with ID %s.", state.ID.ValueString()),
			err.Error(),
		)
		return
	}

	state.OwnerUserID = types.Int64Value(int64(deadHost.OwnerUserID))
	state.DomainNames = []types.String{}
	state.CertificateID = types.Int64Value(int64(*deadHost.CertificateID))
	state.SSLForced = types.BoolValue(*deadHost.SSLForced)
	state.AdvancedConfig = types.StringValue(*deadHost.AdvancedConfig)
	state.HTTP2Support = types.BoolValue(*deadHost.HTTP2Support)
	state.HSTSEnabled = types.BoolValue(*deadHost.HSTSEnabled)
	state.HSTSSubdomains = types.BoolValue(*deadHost.HSTSSubdomains)
	state.Enabled = types.BoolValue(*deadHost.Enabled)

	for _, domainName := range *deadHost.DomainNames {
		state.DomainNames = append(state.DomainNames, types.StringValue(domainName))
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *deadHostDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	r.client = configureProvider(req, resp)
}
