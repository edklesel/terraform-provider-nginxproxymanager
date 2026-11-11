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
	_ datasource.DataSource              = &proxyHostDataSource{}
	_ datasource.DataSourceWithConfigure = &proxyHostDataSource{}
)

type proxyHostDataSource struct {
	client *sdk.Client
}

func NewProxyHostDataSource() datasource.DataSource {
	return &proxyHostDataSource{}
}

func (d *proxyHostDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_proxy_host"
}

func (d *proxyHostDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the proxy host.",
			},
			"forward_host": schema.StringAttribute{
				Computed:    true,
				Description: "Host that the request is forwarded to.",
			},
			"forward_port": schema.Int64Attribute{
				Computed:    true,
				Description: "Port forwarded to.",
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
			"access_list_id": schema.Int64Attribute{
				Computed:    true,
				Description: "ID of the access list used.",
			},
			"certificate_id": schema.Int64Attribute{
				Computed:    true,
				Description: "ID of the certificate used.",
			},
			"ssl_forced": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether SSL is forced.",
			},
			"caching_enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether caching is enabled.",
			},
			"block_exploits": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether exploits are blocked.",
			},
			"allow_websocket_upgrade": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether websockets are allowed.",
			},
			"http2_support": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether HTTP2 support is enabled.",
			},
			"enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the proxy host is enabled or disabled.",
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
				Description: "ID of the user who owns the proxy host.",
			},
		},
		Blocks: map[string]schema.Block{
			"location": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"forward_host": schema.StringAttribute{
							Computed:    true,
							Description: "Host to forward the request on to.",
						},
						"forward_port": schema.Int64Attribute{
							Computed:    true,
							Description: "Port to forward to.",
						},
						"forward_scheme": schema.StringAttribute{
							Computed:    true,
							Description: "Scheme to use when forwarding.",
						},
						"path": schema.StringAttribute{
							Computed:    true,
							Description: "Path matched by the location.",
						},
					},
				},
			},
		},
	}
}

func (d *proxyHostDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	var state models.ProxyHostModel
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

	proxyHost, err := d.client.ProxyHosts.Get(id)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read proxy host with ID %s.", state.ID.ValueString()),
			err.Error(),
		)
		return
	}

	state.OwnerUserID = types.Int64Value(int64(proxyHost.OwnerUserID))
	state.ForwardScheme = types.StringValue(*proxyHost.ForwardScheme)
	state.DomainNames = []types.String{}
	state.ForwardHost = types.StringValue(*proxyHost.ForwardHost)
	state.ForwardPort = types.Int64Value(int64(*proxyHost.ForwardPort))
	state.AccessListID = types.Int64Value(int64(*proxyHost.AccessListID))
	state.CertificateID = types.Int64Value(int64(*proxyHost.CertificateID))
	state.SSLForced = types.BoolValue(*proxyHost.SSLForced)
	state.AdvancedConfig = types.StringValue(*proxyHost.AdvancedConfig)
	state.CachingEnabled = types.BoolValue(*proxyHost.CachingEnabled)
	state.BlockExploits = types.BoolValue(*proxyHost.BlockExploits)
	state.AllowWebsocketUpgrade = types.BoolValue(*proxyHost.AllowWebsocketUpgrade)
	state.HTTP2Support = types.BoolValue(*proxyHost.HTTP2Support)
	state.Enabled = types.BoolValue(*proxyHost.Enabled)
	state.Locations = []models.ProxyHostLocationModel{}
	state.HSTSEnabled = types.BoolValue(*proxyHost.HSTSEnabled)
	state.HSTSSubdomains = types.BoolValue(*proxyHost.HSTSSubdomains)

	for _, location := range *proxyHost.Locations {
		planLocation := models.ProxyHostLocationModel{
			ForwardScheme: types.StringValue(location.ForwardScheme),
			ForwardHost:   types.StringValue(location.ForwardHost),
			ForwardPort:   types.Int64Value(int64(location.ForwardPort)),
			Path:          types.StringValue(location.Path),
		}
		state.Locations = append(state.Locations, planLocation)
	}

	for _, domainName := range *proxyHost.DomainNames {
		state.DomainNames = append(state.DomainNames, types.StringValue(domainName))
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (d *proxyHostDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureProvider(req, resp)
}
