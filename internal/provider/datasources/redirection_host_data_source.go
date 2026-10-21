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
	_ datasource.DataSource              = &redirectionHostDataSource{}
	_ datasource.DataSourceWithConfigure = &redirectionHostDataSource{}
)

type redirectionHostDataSource struct {
	client *sdk.Client
}

func NewRedirectionHostDataSource() datasource.DataSource {
	return &redirectionHostDataSource{}
}

func (d *redirectionHostDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_redirection_host"
}

func (d *redirectionHostDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the redirection host.",
			},
			"forward_domain_name": schema.StringAttribute{
				Computed:    true,
				Description: "Domain name that the request is redirected to.",
			},
			"forward_http_code": schema.Int64Attribute{
				Computed:    true,
				Description: "HTTP code redirected with.",
			},
			"forward_scheme": schema.StringAttribute{
				Computed:    true,
				Description: "Scheme used when forwarding.",
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
			"block_exploits": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether exploits are blocked.",
			},
			"enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the redirection host is enabled.",
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
				Description: "ID of the user who owns the redirection host.",
			},
			"preserve_path": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the request path is preserved in the redirected request.",
			},
		},
	}
}

func (d *redirectionHostDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	var state models.RedirectionHostModel
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

	redirectionHost, err := d.client.RedirectionHosts.Get(id)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read redirection host with ID %s.", state.ID.ValueString()),
			err.Error(),
		)
		return
	}

	state.OwnerUserID = types.Int64Value(int64(redirectionHost.OwnerUserID))
	state.ForwardScheme = types.StringValue(*redirectionHost.ForwardScheme)
	state.DomainNames = []types.String{}
	state.ForwardDomainName = types.StringValue(*redirectionHost.ForwardDomainName)
	state.ForwardHTTPCode = types.Int64Value(int64(*redirectionHost.ForwardHTTPCode))
	state.CertificateID = types.Int64Value(int64(*redirectionHost.CertificateID))
	state.SSLForced = types.BoolValue(*redirectionHost.SSLForced)
	state.AdvancedConfig = types.StringValue(*redirectionHost.AdvancedConfig)
	state.BlockExploits = types.BoolValue(*redirectionHost.BlockExploits)
	state.HTTP2Support = types.BoolValue(*redirectionHost.HTTP2Support)
	state.Enabled = types.BoolValue(*redirectionHost.Enabled)
	state.HSTSEnabled = types.BoolValue(*redirectionHost.HSTSEnabled)
	state.HSTSSubdomains = types.BoolValue(*redirectionHost.HSTSSubdomains)
	state.PreservePath = types.BoolValue(*redirectionHost.PreservePath)

	for _, domainName := range *redirectionHost.DomainNames {
		state.DomainNames = append(state.DomainNames, types.StringValue(domainName))
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (d *redirectionHostDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureProvider(req, resp)
}
