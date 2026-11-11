// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	models "github.com/edklesel/terraform-provider-nginxproxymanager/internal/models"
	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
	utils "github.com/edklesel/terraform-provider-nginxproxymanager/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &userPermissionsResource{}
	_ resource.ResourceWithConfigure = &userPermissionsResource{}
)

type userPermissionsResource struct {
	client *sdk.Client
}

func NewUserPermissionsResource() resource.Resource {
	return &userPermissionsResource{}
}

func (r *userPermissionsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_permissions"
}

func (r *userPermissionsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"user_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the user (required).",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"permissions": schema.SingleNestedBlock{
				Description: "Permissions to grant the user.",
				Attributes: map[string]schema.Attribute{
					"visibility": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Visibility permission.",
						Validators: []validator.String{
							stringvalidator.OneOf("user", "all"),
						},
						Default: stringdefault.StaticString("all"),
					},
					"proxy_hosts": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Proxy Hosts permission.",
						Validators: []validator.String{
							stringvalidator.OneOf("manage", "view", "hidden"),
						},
						Default: stringdefault.StaticString("manage"),
					},
					"redirection_hosts": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Redirection Hosts permission.",
						Validators: []validator.String{
							stringvalidator.OneOf("manage", "view", "hidden"),
						},
						Default: stringdefault.StaticString("manage"),
					},
					"dead_hosts": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Dead Hosts permission.",
						Validators: []validator.String{
							stringvalidator.OneOf("manage", "view", "hidden"),
						},
						Default: stringdefault.StaticString("manage"),
					},
					"streams": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Streams permission.",
						Validators: []validator.String{
							stringvalidator.OneOf("manage", "view", "hidden"),
						},
						Default: stringdefault.StaticString("manage"),
					},
					"access_lists": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Access lists permission.",
						Validators: []validator.String{
							stringvalidator.OneOf("manage", "view", "hidden"),
						},
						Default: stringdefault.StaticString("manage"),
					},
					"certificates": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Certificates permission.",
						Validators: []validator.String{
							stringvalidator.OneOf("manage", "view", "hidden"),
						},
						Default: stringdefault.StaticString("manage"),
					},
				},
				Validators: []validator.Object{
					objectvalidator.IsRequired(),
				},
			},
		},
	}
}

func (r *userPermissionsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.UserPermissionsModel
	diags := req.Plan.Get(ctx, &plan)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.Users.Get(int(plan.UserID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error finding user.",
			fmt.Sprintf("Unable to find user %d, unexpected error: %s", plan.UserID.ValueInt64(), err.Error()),
		)
		return
	}

	var permissions = sdk.Permissions{
		Visibility:       plan.Permissions.Visibility.ValueStringPointer(),
		ProxyHosts:       plan.Permissions.ProxyHosts.ValueStringPointer(),
		RedirectionHosts: plan.Permissions.RedirectionHosts.ValueStringPointer(),
		DeadHosts:        plan.Permissions.DeadHosts.ValueStringPointer(),
		Streams:          plan.Permissions.Streams.ValueStringPointer(),
		AccessLists:      plan.Permissions.AccessLists.ValueStringPointer(),
		Certificates:     plan.Permissions.Certificates.ValueStringPointer(),
	}

	err = r.client.Users.UpdatePermissions(user.ID, permissions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error setting user permissions.",
			fmt.Sprintf("Unable to set permissions for user %d, unexpected error: %s", plan.UserID.ValueInt64(), err.Error()),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *userPermissionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	var state models.UserPermissionsModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.Users.Get(int(state.UserID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read user with ID %d.", state.UserID.ValueInt64()),
			err.Error(),
		)
		return
	}

	state = models.UserPermissionsModel{
		UserID: types.Int64Value(int64(user.ID)),
		Permissions: models.UserPermissionModel{
			Visibility:       types.StringValue(*user.Permissions.Visibility),
			ProxyHosts:       types.StringValue(*user.Permissions.ProxyHosts),
			RedirectionHosts: types.StringValue(*user.Permissions.RedirectionHosts),
			DeadHosts:        types.StringValue(*user.Permissions.DeadHosts),
			Streams:          types.StringValue(*user.Permissions.Streams),
			AccessLists:      types.StringValue(*user.Permissions.AccessLists),
			Certificates:     types.StringValue(*user.Permissions.Certificates),
		},
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *userPermissionsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.UserPermissionsModel
	diags := req.Plan.Get(ctx, &plan)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.Users.Get(int(plan.UserID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error finding user.",
			fmt.Sprintf("Unable to find user %d, unexpected error: %s", plan.UserID.ValueInt64(), err.Error()),
		)
		return
	}

	var permissions = sdk.Permissions{
		Visibility:       plan.Permissions.Visibility.ValueStringPointer(),
		ProxyHosts:       plan.Permissions.ProxyHosts.ValueStringPointer(),
		RedirectionHosts: plan.Permissions.RedirectionHosts.ValueStringPointer(),
		DeadHosts:        plan.Permissions.DeadHosts.ValueStringPointer(),
		Streams:          plan.Permissions.Streams.ValueStringPointer(),
		AccessLists:      plan.Permissions.AccessLists.ValueStringPointer(),
		Certificates:     plan.Permissions.Certificates.ValueStringPointer(),
	}

	err = r.client.Users.UpdatePermissions(user.ID, permissions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error setting user permissions.",
			fmt.Sprintf("Unable to set permissions for user %d, unexpected error: %s", plan.UserID.ValueInt64(), err.Error()),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *userPermissionsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.UserPermissionsModel
	diags := req.State.Get(ctx, &state)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.Users.Get(int(state.UserID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error finding user.",
			fmt.Sprintf("Unable to find user %d, unexpected error: %s", state.UserID.ValueInt64(), err.Error()),
		)
		return
	}

	var permissions = sdk.Permissions{
		Visibility:       utils.ReturnPointer("user"),
		ProxyHosts:       utils.ReturnPointer("manage"),
		RedirectionHosts: utils.ReturnPointer("manage"),
		DeadHosts:        utils.ReturnPointer("manage"),
		Streams:          utils.ReturnPointer("manage"),
		AccessLists:      utils.ReturnPointer("manage"),
		Certificates:     utils.ReturnPointer("manage"),
	}

	err = r.client.Users.UpdatePermissions(user.ID, permissions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error resetting user permissions.",
			fmt.Sprintf("Unable to reset permissions for user %d, unexpected error: %s", state.UserID.ValueInt64(), err.Error()),
		)
		return
	}
}

func (r *userPermissionsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureProvider(req, resp)
}
