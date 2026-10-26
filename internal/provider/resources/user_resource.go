// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	models "github.com/edklesel/terraform-provider-nginxproxymanager/internal/models"
	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
	utils "github.com/edklesel/terraform-provider-nginxproxymanager/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &userResource{}
	_ resource.ResourceWithConfigure   = &userResource{}
	_ resource.ResourceWithImportState = &userResource{}
)

type userResource struct {
	client *sdk.Client
}

func NewUserResource() resource.Resource {
	return &userResource{}
}

func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the user.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"email": schema.StringAttribute{
				Required:    true,
				Description: "Email of the user, used for logging in. Must be lowercase, as NPM lowercases it when updating a user.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[^A-Z]*$`), "must be lowercase, as NPM lowercases email addresses when updating a user"),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Username of the user.",
			},
			"nickname": schema.StringAttribute{
				Required:    true,
				Description: "Nickname for the user.",
			},
			"is_disabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the user is disabled or not.",
				Default:     booldefault.StaticBool(false),
			},
			"roles": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "A list of roles that the user has.",
				Default: listdefault.StaticValue(
					types.ListValueMust(
						types.StringType,
						[]attr.Value{},
					),
				),
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.OneOf("admin", "user"),
					),
				},
			},
			"avatar": schema.StringAttribute{
				Computed:    true,
				Description: "URL to the avatar for the user.",
			},
		},
	}
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.UserModel
	diags := req.Plan.Get(ctx, &plan)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var _user = sdk.UserCreate{
		UserBase: sdk.UserBase{
			Name:       plan.Name.ValueStringPointer(),
			Nickname:   plan.Nickname.ValueStringPointer(),
			Email:      plan.Email.ValueStringPointer(),
			IsDisabled: plan.IsDisabled.ValueBoolPointer(),
			Roles:      utils.ReturnPointer([]string{}),
		},
	}
	for _, role := range plan.Roles {
		*_user.Roles = append(*_user.Roles, role.ValueString())
	}

	user, err := r.client.Users.Create(_user)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating user.",
			fmt.Sprintf("Unable to create user, unexpected error: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(user.ID))
	plan.Email = types.StringValue(*user.Email)
	plan.Name = types.StringValue(*user.Name)
	plan.Nickname = types.StringValue(*user.Nickname)
	plan.IsDisabled = types.BoolValue(*user.IsDisabled)
	plan.Avatar = types.StringValue(user.Avatar)

	plan.Roles = []types.String{}
	for _, role := range *user.Roles {
		plan.Roles = append(plan.Roles, types.StringValue(role))
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	var state models.UserModel
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

	user, err := r.client.Users.Get(id)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read user with ID %s.", state.ID.ValueString()),
			err.Error(),
		)
		return
	}

	state.ID = types.StringValue(strconv.Itoa(user.ID))
	state.Email = types.StringValue(*user.Email)
	state.Name = types.StringValue(*user.Name)
	state.Nickname = types.StringValue(*user.Nickname)
	state.IsDisabled = types.BoolValue(*user.IsDisabled)
	state.Avatar = types.StringValue(user.Avatar)

	state.Roles = []types.String{}
	for _, role := range *user.Roles {
		state.Roles = append(state.Roles, types.StringValue(role))
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.UserModel
	diags := req.Plan.Get(ctx, &plan)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var newProperties = sdk.UserBase{
		Name:       plan.Name.ValueStringPointer(),
		Nickname:   plan.Nickname.ValueStringPointer(),
		Email:      plan.Email.ValueStringPointer(),
		IsDisabled: plan.IsDisabled.ValueBoolPointer(),
		Roles:      utils.ReturnPointer([]string{}),
	}
	for _, role := range plan.Roles {
		*newProperties.Roles = append(*newProperties.Roles, role.ValueString())
	}

	id, err := strconv.Atoi(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to convert ID value %s to int type", plan.ID),
			err.Error(),
		)
		return
	}

	user, err := r.client.Users.Update(id, newProperties)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating user",
			fmt.Sprintf("Unable to update user, unexpected error: %s", err.Error()),
		)
	}

	plan.ID = types.StringValue(strconv.Itoa(user.ID))
	plan.Email = types.StringValue(*user.Email)
	plan.Name = types.StringValue(*user.Name)
	plan.Nickname = types.StringValue(*user.Nickname)
	plan.IsDisabled = types.BoolValue(*user.IsDisabled)
	plan.Avatar = types.StringValue(user.Avatar)

	plan.Roles = []types.String{}
	for _, role := range *user.Roles {
		plan.Roles = append(plan.Roles, types.StringValue(role))
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	var state models.UserModel
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

	err = r.client.Users.Delete(id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting user",
			fmt.Sprintf("Unable to delete user, unexpected error: %s", err.Error()),
		)
	}
}

func (r *userResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureProvider(req, resp)
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
