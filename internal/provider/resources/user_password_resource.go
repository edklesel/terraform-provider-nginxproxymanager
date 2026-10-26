// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	models "github.com/edklesel/terraform-provider-nginxproxymanager/internal/models"
	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var (
	_ resource.Resource              = &userPasswordResource{}
	_ resource.ResourceWithConfigure = &userPasswordResource{}
)

type userPasswordResource struct {
	client *sdk.Client
}

func NewUserPasswordResource() resource.Resource {
	return &userPasswordResource{}
}

func (r *userPasswordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_password"
}

func (r *userPasswordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"user_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the user (required).",
			},
			"password": schema.StringAttribute{
				Required:    true,
				Description: "Password to set for the user.",
				Sensitive:   true,
			},
		},
	}
}

func (r *userPasswordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.UserPasswordModel
	diags := req.Plan.Get(ctx, &plan)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.Users.Get(int(plan.UserID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error locating user.",
			fmt.Sprintf("Unable to find user, unexpected error: %s", err.Error()),
		)
		return
	}

	auth := sdk.UserCreateAuth{
		Type:   "password",
		Secret: plan.Password.ValueString(),
	}
	err = r.client.Users.SetPassword(user.ID, auth)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error setting password for user %d.", user.ID),
			fmt.Sprintf("Unable to set password for user %d, unexpected error: %s", user.ID, err.Error()),
		)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// The password isn't returned from the API, so this only checks the user
// still exists.
func (r *userPasswordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.UserPasswordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Users.Get(int(state.UserID.ValueInt64()))
	if sdk.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read user with ID %d.", state.UserID.ValueInt64()),
			err.Error(),
		)
	}
}

func (r *userPasswordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.UserPasswordModel
	diags := req.Plan.Get(ctx, &plan)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.Users.Get(int(plan.UserID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error locating user.",
			fmt.Sprintf("Unable to find user, unexpected error: %s", err.Error()),
		)
		return
	}

	auth := sdk.UserCreateAuth{
		Type:   "password",
		Secret: plan.Password.ValueString(),
	}
	err = r.client.Users.SetPassword(user.ID, auth)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error setting password for user %d.", user.ID),
			fmt.Sprintf("Unable to set password for user %d, unexpected error: %s", user.ID, err.Error()),
		)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *userPasswordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

func (r *userPasswordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureProvider(req, resp)
}

func (r *userPasswordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
}
