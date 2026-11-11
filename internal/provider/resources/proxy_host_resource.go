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

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &proxyHostResource{}
	_ resource.ResourceWithConfigure      = &proxyHostResource{}
	_ resource.ResourceWithImportState    = &proxyHostResource{}
	_ resource.ResourceWithValidateConfig = &proxyHostResource{}
)

type proxyHostResource struct {
	client *sdk.Client
}

func NewProxyHostResource() resource.Resource {
	return &proxyHostResource{}
}

func (r *proxyHostResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_proxy_host"
}

func (r *proxyHostResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the proxy host.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"forward_host": schema.StringAttribute{
				Required:    true,
				Description: "Host to forward the request on to.",
			},
			"forward_port": schema.Int64Attribute{
				Required:    true,
				Description: "Port to forward to.",
			},
			"forward_scheme": schema.StringAttribute{
				Required:    true,
				Description: "Scheme to use when forwarding.",
			},
			"domain_names": schema.ListAttribute{
				Required:    true,
				Description: "Domain names to accept.",
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"access_list_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "ID of the access list to use.",
				Default:     int64default.StaticInt64(0),
			},
			"certificate_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "ID of the certificate to use.",
				Default:     int64default.StaticInt64(0),
			},
			"ssl_forced": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to force SSL.",
				Default:     booldefault.StaticBool(false),
			},
			"caching_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to enable caching,",
				Default:     booldefault.StaticBool(false),
			},
			"block_exploits": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to block exploits",
				Default:     booldefault.StaticBool(false),
			},
			"allow_websocket_upgrade": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to allow websockets.",
				Default:     booldefault.StaticBool(false),
			},
			"http2_support": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to enable HTTP2 support.",
				Default:     booldefault.StaticBool(false),
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable or disable the proxy host.",
				Default:     booldefault.StaticBool(true),
			},
			"hsts_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable HSTS.",
				Validators:  []validator.Bool{},
				Default:     booldefault.StaticBool(false),
			},
			"hsts_subdomains": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable HSTS for subdomains.",
				Default:     booldefault.StaticBool(false),
			},
			"advanced_config": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Advanced NGINX config.",
				Default:     stringdefault.StaticString(""),
			},
			"owner_user_id": schema.Int64Attribute{
				Computed:    true,
				Description: "ID of the user which owns the proxy host.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"location": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"forward_host": schema.StringAttribute{
							Required:    true,
							Description: "Host to forward the request on to.",
						},
						"forward_port": schema.Int64Attribute{
							Required:    true,
							Description: "Port to forward to.",
						},
						"forward_scheme": schema.StringAttribute{
							Required:    true,
							Description: "Scheme to use when forwarding.",
						},
						"path": schema.StringAttribute{
							Required:    true,
							Description: "Path to match, e.g `/api`.",
						},
					},
				},
			},
		},
	}
}

func (r *proxyHostResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data models.ProxyHostModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// ssl_forced and http2_upgrade require a certificate ID to be set
	if data.CertificateID.IsNull() {
		if data.SSLForced.ValueBool() {
			resp.Diagnostics.AddAttributeError(
				path.Root("ssl_forced"),
				"ssl_forced cannot be true when no certificate_id is given",
				"Forcing SSL requires a SSL Certificate to be set on the Proxy Host",
			)
		}
		if data.HTTP2Support.ValueBool() {
			resp.Diagnostics.AddAttributeError(
				path.Root("http2_support"),
				"http2_support cannot be true when no certificate_id is given",
				"Enabling HTTP2 support requires a Certificate to be set",
			)
		}
	}

	// hsts_enabled depends on ssl_forced being true
	if !data.SSLForced.ValueBool() && data.HSTSEnabled.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			path.Root("hsts_enabled"),
			"hsts_enabled cannot be true when ssl_forced is false",
			"Enabling HSTS requires SSL to be forced",
		)
	}
	// hsts_dubdomains depends on hsts_enabled being true
	if !data.HSTSEnabled.ValueBool() && data.HSTSSubdomains.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			path.Root("hsts_subdomains"),
			"hsts_subdomains cannot be true when hsts_enabled is false",
			"Enabling HSTS for Subdomains requires HSTS to be enabled",
		)
	}
}

func (r *proxyHostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.ProxyHostModel
	diags := req.Plan.Get(ctx, &plan)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var _proxyHost = sdk.ProxyHostBase{
		ForwardScheme:         plan.ForwardScheme.ValueStringPointer(),
		ForwardHost:           plan.ForwardHost.ValueStringPointer(),
		ForwardPort:           utils.ReturnPointer(int(plan.ForwardPort.ValueInt64())),
		DomainNames:           utils.ReturnPointer([]string{}),
		AccessListID:          utils.ReturnPointer(int(plan.AccessListID.ValueInt64())),
		CertificateID:         utils.ReturnPointer(int(plan.CertificateID.ValueInt64())),
		CachingEnabled:        plan.CachingEnabled.ValueBoolPointer(),
		BlockExploits:         plan.BlockExploits.ValueBoolPointer(),
		AdvancedConfig:        plan.AdvancedConfig.ValueStringPointer(),
		AllowWebsocketUpgrade: plan.AllowWebsocketUpgrade.ValueBoolPointer(),
		HTTP2Support:          plan.HTTP2Support.ValueBoolPointer(),
		HSTSEnabled:           plan.HSTSEnabled.ValueBoolPointer(),
		HSTSSubdomains:        plan.HSTSSubdomains.ValueBoolPointer(),
		Locations:             utils.ReturnPointer([]sdk.Location{}),
	}

	for _, domainName := range plan.DomainNames {
		*_proxyHost.DomainNames = append(*_proxyHost.DomainNames, domainName.ValueString())
	}

	for _, location := range plan.Locations {
		_location := sdk.Location{
			ForwardScheme: location.ForwardScheme.ValueString(),
			ForwardPort:   int(location.ForwardPort.ValueInt64()),
			ForwardHost:   location.ForwardHost.ValueString(),
			Path:          location.Path.ValueString(),
		}
		*_proxyHost.Locations = append(*_proxyHost.Locations, _location)
	}

	proxyHost, err := r.client.ProxyHosts.Create(_proxyHost)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating proxy host.",
			fmt.Sprintf("Unable to create proxy host, unexpected error: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(proxyHost.ID))
	plan.OwnerUserID = types.Int64Value(int64(proxyHost.OwnerUserID))
	plan.ForwardScheme = types.StringValue(*proxyHost.ForwardScheme)
	plan.DomainNames = []types.String{}
	plan.ForwardHost = types.StringValue(*proxyHost.ForwardHost)
	plan.ForwardPort = types.Int64Value(int64(*proxyHost.ForwardPort))
	plan.AccessListID = types.Int64Value(int64(*proxyHost.AccessListID))
	plan.CertificateID = types.Int64Value(int64(*proxyHost.CertificateID))
	plan.SSLForced = types.BoolValue(*proxyHost.SSLForced)
	plan.AdvancedConfig = types.StringValue(*proxyHost.AdvancedConfig)
	plan.CachingEnabled = types.BoolValue(*proxyHost.CachingEnabled)
	plan.BlockExploits = types.BoolValue(*proxyHost.BlockExploits)
	plan.AllowWebsocketUpgrade = types.BoolValue(*proxyHost.AllowWebsocketUpgrade)
	plan.HTTP2Support = types.BoolValue(*proxyHost.HTTP2Support)
	plan.Enabled = types.BoolValue(*proxyHost.Enabled)
	plan.Locations = []models.ProxyHostLocationModel{}
	plan.HSTSEnabled = types.BoolValue(*proxyHost.HSTSEnabled)
	plan.HSTSSubdomains = types.BoolValue(*proxyHost.HSTSSubdomains)

	for _, location := range *proxyHost.Locations {
		planLocation := models.ProxyHostLocationModel{
			ForwardScheme: types.StringValue(location.ForwardScheme),
			ForwardHost:   types.StringValue(location.ForwardHost),
			ForwardPort:   types.Int64Value(int64(location.ForwardPort)),
			Path:          types.StringValue(location.Path),
		}
		plan.Locations = append(plan.Locations, planLocation)
	}

	for _, domainName := range *proxyHost.DomainNames {
		plan.DomainNames = append(plan.DomainNames, types.StringValue(domainName))
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *proxyHostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	var state models.ProxyHostModel
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

	proxyHost, err := r.client.ProxyHosts.Get(id)
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

func (r *proxyHostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.ProxyHostModel
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

	var _proxyHost = sdk.ProxyHostBase{
		ForwardScheme:         plan.ForwardScheme.ValueStringPointer(),
		ForwardHost:           plan.ForwardHost.ValueStringPointer(),
		ForwardPort:           utils.ReturnPointer(int(plan.ForwardPort.ValueInt64())),
		DomainNames:           utils.ReturnPointer([]string{}),
		AccessListID:          utils.ReturnPointer(int(plan.AccessListID.ValueInt64())),
		CertificateID:         utils.ReturnPointer(int(plan.CertificateID.ValueInt64())),
		CachingEnabled:        plan.CachingEnabled.ValueBoolPointer(),
		BlockExploits:         plan.BlockExploits.ValueBoolPointer(),
		AdvancedConfig:        plan.AdvancedConfig.ValueStringPointer(),
		AllowWebsocketUpgrade: plan.AllowWebsocketUpgrade.ValueBoolPointer(),
		HTTP2Support:          plan.HTTP2Support.ValueBoolPointer(),
		HSTSEnabled:           plan.HSTSEnabled.ValueBoolPointer(),
		HSTSSubdomains:        plan.HSTSSubdomains.ValueBoolPointer(),
		Locations:             utils.ReturnPointer([]sdk.Location{}),
	}

	for _, domainName := range plan.DomainNames {
		*_proxyHost.DomainNames = append(*_proxyHost.DomainNames, domainName.ValueString())
	}

	for _, location := range plan.Locations {
		_location := sdk.Location{
			ForwardScheme: location.ForwardScheme.ValueString(),
			ForwardPort:   int(location.ForwardPort.ValueInt64()),
			ForwardHost:   location.ForwardHost.ValueString(),
			Path:          location.Path.ValueString(),
		}
		*_proxyHost.Locations = append(*_proxyHost.Locations, _location)
	}

	proxyHost, err := r.client.ProxyHosts.Update(id, _proxyHost)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating proxy host.",
			fmt.Sprintf("Unable to update proxy host %d, unexpected error: %s", id, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(proxyHost.ID))
	plan.OwnerUserID = types.Int64Value(int64(proxyHost.OwnerUserID))
	plan.ForwardScheme = types.StringValue(*proxyHost.ForwardScheme)
	plan.DomainNames = []types.String{}
	plan.ForwardHost = types.StringValue(*proxyHost.ForwardHost)
	plan.ForwardPort = types.Int64Value(int64(*proxyHost.ForwardPort))
	plan.AccessListID = types.Int64Value(int64(*proxyHost.AccessListID))
	plan.CertificateID = types.Int64Value(int64(*proxyHost.CertificateID))
	plan.SSLForced = types.BoolValue(*proxyHost.SSLForced)
	plan.AdvancedConfig = types.StringValue(*proxyHost.AdvancedConfig)
	plan.CachingEnabled = types.BoolValue(*proxyHost.CachingEnabled)
	plan.BlockExploits = types.BoolValue(*proxyHost.BlockExploits)
	plan.AllowWebsocketUpgrade = types.BoolValue(*proxyHost.AllowWebsocketUpgrade)
	plan.HTTP2Support = types.BoolValue(*proxyHost.HTTP2Support)
	plan.Enabled = types.BoolValue(*proxyHost.Enabled)
	plan.Locations = []models.ProxyHostLocationModel{}
	plan.HSTSEnabled = types.BoolValue(*proxyHost.HSTSEnabled)
	plan.HSTSSubdomains = types.BoolValue(*proxyHost.HSTSSubdomains)

	for _, location := range *proxyHost.Locations {
		planLocation := models.ProxyHostLocationModel{
			ForwardScheme: types.StringValue(location.ForwardScheme),
			ForwardHost:   types.StringValue(location.ForwardHost),
			ForwardPort:   types.Int64Value(int64(location.ForwardPort)),
			Path:          types.StringValue(location.Path),
		}
		plan.Locations = append(plan.Locations, planLocation)
	}

	for _, domainName := range *proxyHost.DomainNames {
		plan.DomainNames = append(plan.DomainNames, types.StringValue(domainName))
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *proxyHostResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	var state models.ProxyHostModel
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

	err = r.client.ProxyHosts.Delete(id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting proxy host",
			fmt.Sprintf("Unable to delete proxy host, unexpected error: %s", err.Error()),
		)
	}
}

func (r *proxyHostResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureProvider(req, resp)
}

func (r *proxyHostResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
