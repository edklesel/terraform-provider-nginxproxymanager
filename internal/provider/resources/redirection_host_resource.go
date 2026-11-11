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

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
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
	_ resource.Resource                   = &redirectionHostResource{}
	_ resource.ResourceWithConfigure      = &redirectionHostResource{}
	_ resource.ResourceWithImportState    = &redirectionHostResource{}
	_ resource.ResourceWithValidateConfig = &redirectionHostResource{}
)

type redirectionHostResource struct {
	client *sdk.Client
}

func NewRedirectionHostResource() resource.Resource {
	return &redirectionHostResource{}
}

func (r *redirectionHostResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_redirection_host"
}

func (r *redirectionHostResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the redirection host.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"forward_domain_name": schema.StringAttribute{
				Required:    true,
				Description: "Domain name to redirect the request to.",
			},
			"forward_scheme": schema.StringAttribute{
				Required:    true,
				Description: "Scheme to use when forwarding.",
			},
			"forward_http_code": schema.Int64Attribute{
				Required:    true,
				Description: "HTTP code to redirect with.",
			},
			"domain_names": schema.SetAttribute{
				Required:    true,
				Description: "Domain names to accept. NPM stores these sorted, so the order doesn't matter.",
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
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
			"block_exploits": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to block exploits",
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
				Description: "Enable or disable the redirection host.",
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
			"preserve_path": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to preserve the request path in the redirected request.",
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
				Description: "ID of the user which owns the redirection host.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *redirectionHostResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data models.RedirectionHostModel
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
				"Forcing SSL requires a SSL Certificate to be set on the Redirection Host",
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

func (r *redirectionHostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.RedirectionHostModel
	diags := req.Plan.Get(ctx, &plan)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var _redirectionHost = sdk.RedirectionHostBase{
		ForwardDomainName: plan.ForwardDomainName.ValueStringPointer(),
		ForwardHTTPCode:   utils.ReturnPointer(int(plan.ForwardHTTPCode.ValueInt64())),
		ForwardScheme:     plan.ForwardScheme.ValueStringPointer(),
		DomainNames:       utils.ReturnPointer([]string{}),
		CertificateID:     utils.ReturnPointer(int(plan.CertificateID.ValueInt64())),
		SSLForced:         plan.SSLForced.ValueBoolPointer(),
		BlockExploits:     plan.BlockExploits.ValueBoolPointer(),
		AdvancedConfig:    plan.AdvancedConfig.ValueStringPointer(),
		HTTP2Support:      plan.HTTP2Support.ValueBoolPointer(),
		HSTSEnabled:       plan.HSTSEnabled.ValueBoolPointer(),
		HSTSSubdomains:    plan.HSTSSubdomains.ValueBoolPointer(),
		PreservePath:      plan.PreservePath.ValueBoolPointer(),
	}

	for _, domainName := range plan.DomainNames {
		*_redirectionHost.DomainNames = append(*_redirectionHost.DomainNames, domainName.ValueString())
	}

	redirectionHost, err := r.client.RedirectionHosts.Create(_redirectionHost)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating redirection host.",
			fmt.Sprintf("Unable to create redirection host, unexpected error: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(redirectionHost.ID))
	plan.OwnerUserID = types.Int64Value(int64(redirectionHost.OwnerUserID))
	plan.DomainNames = []types.String{}
	plan.ForwardDomainName = types.StringValue(*redirectionHost.ForwardDomainName)
	plan.ForwardHTTPCode = types.Int64Value(int64(*redirectionHost.ForwardHTTPCode))
	plan.ForwardScheme = types.StringValue(*redirectionHost.ForwardScheme)
	plan.CertificateID = types.Int64Value(int64(*redirectionHost.CertificateID))
	plan.SSLForced = types.BoolValue(*redirectionHost.SSLForced)
	plan.AdvancedConfig = types.StringValue(*redirectionHost.AdvancedConfig)
	plan.BlockExploits = types.BoolValue(*redirectionHost.BlockExploits)
	plan.HTTP2Support = types.BoolValue(*redirectionHost.HTTP2Support)
	plan.Enabled = types.BoolValue(*redirectionHost.Enabled)
	plan.HSTSEnabled = types.BoolValue(*redirectionHost.HSTSEnabled)
	plan.HSTSSubdomains = types.BoolValue(*redirectionHost.HSTSSubdomains)
	plan.PreservePath = types.BoolValue(*redirectionHost.PreservePath)

	for _, domainName := range *redirectionHost.DomainNames {
		plan.DomainNames = append(plan.DomainNames, types.StringValue(domainName))
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *redirectionHostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	var state models.RedirectionHostModel
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

	redirectionHost, err := r.client.RedirectionHosts.Get(id)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read redirection host with ID %s.", state.ID.ValueString()),
			err.Error(),
		)
		return
	}

	state.OwnerUserID = types.Int64Value(int64(redirectionHost.OwnerUserID))
	state.DomainNames = []types.String{}
	state.ForwardDomainName = types.StringValue(*redirectionHost.ForwardDomainName)
	state.ForwardScheme = types.StringValue(*redirectionHost.ForwardScheme)
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

func (r *redirectionHostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.RedirectionHostModel
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

	var _redirectionHost = sdk.RedirectionHostBase{
		ForwardDomainName: plan.ForwardDomainName.ValueStringPointer(),
		ForwardHTTPCode:   utils.ReturnPointer(int(plan.ForwardHTTPCode.ValueInt64())),
		ForwardScheme:     plan.ForwardScheme.ValueStringPointer(),
		DomainNames:       utils.ReturnPointer([]string{}),
		CertificateID:     utils.ReturnPointer(int(plan.CertificateID.ValueInt64())),
		BlockExploits:     plan.BlockExploits.ValueBoolPointer(),
		AdvancedConfig:    plan.AdvancedConfig.ValueStringPointer(),
		HTTP2Support:      plan.HTTP2Support.ValueBoolPointer(),
		HSTSEnabled:       plan.HSTSEnabled.ValueBoolPointer(),
		HSTSSubdomains:    plan.HSTSSubdomains.ValueBoolPointer(),
		PreservePath:      plan.PreservePath.ValueBoolPointer(),
	}

	for _, domainName := range plan.DomainNames {
		*_redirectionHost.DomainNames = append(*_redirectionHost.DomainNames, domainName.ValueString())
	}

	redirectionHost, err := r.client.RedirectionHosts.Update(id, _redirectionHost)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating redirection host.",
			fmt.Sprintf("Unable to update redirection host %d, unexpected error: %s", id, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(redirectionHost.ID))
	plan.OwnerUserID = types.Int64Value(int64(redirectionHost.OwnerUserID))
	plan.DomainNames = []types.String{}
	plan.ForwardDomainName = types.StringValue(*redirectionHost.ForwardDomainName)
	plan.ForwardHTTPCode = types.Int64Value(int64(*redirectionHost.ForwardHTTPCode))
	plan.ForwardScheme = types.StringValue(*redirectionHost.ForwardScheme)
	plan.CertificateID = types.Int64Value(int64(*redirectionHost.CertificateID))
	plan.SSLForced = types.BoolValue(*redirectionHost.SSLForced)
	plan.AdvancedConfig = types.StringValue(*redirectionHost.AdvancedConfig)
	plan.BlockExploits = types.BoolValue(*redirectionHost.BlockExploits)
	plan.HTTP2Support = types.BoolValue(*redirectionHost.HTTP2Support)
	plan.Enabled = types.BoolValue(*redirectionHost.Enabled)
	plan.HSTSEnabled = types.BoolValue(*redirectionHost.HSTSEnabled)
	plan.HSTSSubdomains = types.BoolValue(*redirectionHost.HSTSSubdomains)
	plan.PreservePath = types.BoolValue(*redirectionHost.PreservePath)

	for _, domainName := range *redirectionHost.DomainNames {
		plan.DomainNames = append(plan.DomainNames, types.StringValue(domainName))
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *redirectionHostResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	var state models.RedirectionHostModel
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

	err = r.client.RedirectionHosts.Delete(id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting redirection host",
			fmt.Sprintf("Unable to delete redirection host, unexpected error: %s", err.Error()),
		)
	}
}

func (r *redirectionHostResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureProvider(req, resp)
}

func (r *redirectionHostResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
