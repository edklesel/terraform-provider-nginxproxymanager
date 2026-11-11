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
	_ resource.Resource                = &deadHostResource{}
	_ resource.ResourceWithConfigure   = &deadHostResource{}
	_ resource.ResourceWithImportState = &deadHostResource{}
)

type deadHostResource struct {
	client *sdk.Client
}

func NewDeadHostResource() resource.Resource {
	return &deadHostResource{}
}

func (r *deadHostResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dead_host"
}

func (r *deadHostResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the dead host.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
			"http2_support": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to enable HTTP2 support.",
				Default:     booldefault.StaticBool(false),
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable or disable the dead host.",
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
				Description: "ID of the user which owns the dead host.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *deadHostResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data models.DeadHostModel
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

func (r *deadHostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.DeadHostModel
	diags := req.Plan.Get(ctx, &plan)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var _deadHost = sdk.DeadHostBase{
		DomainNames:    utils.ReturnPointer([]string{}),
		CertificateID:  utils.ReturnPointer(int(plan.CertificateID.ValueInt64())),
		AdvancedConfig: plan.AdvancedConfig.ValueStringPointer(),
		HTTP2Support:   plan.HTTP2Support.ValueBoolPointer(),
		HSTSEnabled:    plan.HSTSEnabled.ValueBoolPointer(),
		HSTSSubdomains: plan.HSTSSubdomains.ValueBoolPointer(),
	}

	for _, domainName := range plan.DomainNames {
		*_deadHost.DomainNames = append(*_deadHost.DomainNames, domainName.ValueString())
	}

	deadHost, err := r.client.DeadHosts.Create(_deadHost)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating dead host.",
			fmt.Sprintf("Unable to create dead host, unexpected error: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(deadHost.ID))
	plan.OwnerUserID = types.Int64Value(int64(deadHost.OwnerUserID))
	plan.DomainNames = []types.String{}
	plan.CertificateID = types.Int64Value(int64(*deadHost.CertificateID))
	plan.SSLForced = types.BoolValue(*deadHost.SSLForced)
	plan.AdvancedConfig = types.StringValue(*deadHost.AdvancedConfig)
	plan.HTTP2Support = types.BoolValue(*deadHost.HTTP2Support)
	plan.HSTSEnabled = types.BoolValue(*deadHost.HSTSEnabled)
	plan.HSTSSubdomains = types.BoolValue(*deadHost.HSTSSubdomains)
	plan.Enabled = types.BoolValue(*deadHost.Enabled)

	for _, domainName := range *deadHost.DomainNames {
		plan.DomainNames = append(plan.DomainNames, types.StringValue(domainName))
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *deadHostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	var state models.DeadHostModel
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

func (r *deadHostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.DeadHostModel
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

	var _deadHost = sdk.DeadHostBase{
		DomainNames:    utils.ReturnPointer([]string{}),
		CertificateID:  utils.ReturnPointer(int(plan.CertificateID.ValueInt64())),
		AdvancedConfig: plan.AdvancedConfig.ValueStringPointer(),
		HTTP2Support:   plan.HTTP2Support.ValueBoolPointer(),
		HSTSEnabled:    plan.HSTSEnabled.ValueBoolPointer(),
		HSTSSubdomains: plan.HSTSSubdomains.ValueBoolPointer(),
	}

	for _, domainName := range plan.DomainNames {
		*_deadHost.DomainNames = append(*_deadHost.DomainNames, domainName.ValueString())
	}

	deadHost, err := r.client.DeadHosts.Update(id, _deadHost)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating dead host.",
			fmt.Sprintf("Unable to update dead host %d, unexpected error: %s", id, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(deadHost.ID))
	plan.OwnerUserID = types.Int64Value(int64(deadHost.OwnerUserID))
	plan.DomainNames = []types.String{}
	plan.CertificateID = types.Int64Value(int64(*deadHost.CertificateID))
	plan.SSLForced = types.BoolValue(*deadHost.SSLForced)
	plan.AdvancedConfig = types.StringValue(*deadHost.AdvancedConfig)
	plan.HTTP2Support = types.BoolValue(*deadHost.HTTP2Support)
	plan.HSTSEnabled = types.BoolValue(*deadHost.HSTSEnabled)
	plan.HSTSSubdomains = types.BoolValue(*deadHost.HSTSSubdomains)
	plan.Enabled = types.BoolValue(*deadHost.Enabled)

	for _, domainName := range *deadHost.DomainNames {
		plan.DomainNames = append(plan.DomainNames, types.StringValue(domainName))
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *deadHostResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	var state models.DeadHostModel
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

	err = r.client.DeadHosts.Delete(id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting dead host",
			fmt.Sprintf("Unable to delete dead host, unexpected error: %s", err.Error()),
		)
	}
}

func (r *deadHostResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureProvider(req, resp)
}

func (r *deadHostResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
