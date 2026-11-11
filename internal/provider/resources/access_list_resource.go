// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	models "github.com/edklesel/terraform-provider-nginxproxymanager/internal/models"
	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &accessListResource{}
	_ resource.ResourceWithConfigure   = &accessListResource{}
	_ resource.ResourceWithImportState = &accessListResource{}
)

type accessListResource struct {
	client *sdk.Client
}

func NewAccessListResource() resource.Resource {
	return &accessListResource{}
}

func (r *accessListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_list"
}

func (r *accessListResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the access list.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the access list.",
			},
			"satisfy_any": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether a request should pass if it satisfies any of the rules.",
			},
			"pass_auth": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the to pass auth to the host.",
			},
			"owner_user_id": schema.Int64Attribute{
				Computed:    true,
				Description: "ID of the user which owns the access list.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			// NPM inserts the users in parallel, so it doesn't keep their order
			"item": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"username": schema.StringAttribute{
							Description: "Basic auth username.",
							Required:    true,
						},
						"password": schema.StringAttribute{
							Description: "Basic auth password. NPM doesn't return passwords, so a password changed outside Terraform isn't detected.",
							Required:    true,
							Sensitive:   true,
						},
					},
				},
			},
			"client": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"address": schema.StringAttribute{
							Description: "IP Address of a client.",
							Required:    true,
						},
						"directive": schema.StringAttribute{
							Description: "Directive, allow or deny.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("allow", "deny"),
							},
						},
					},
				},
			},
		},
	}
}

func (r *accessListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.AccessListModel
	diags := req.Plan.Get(ctx, &plan)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var _accessList = sdk.AccessListBase{
		Name:       plan.Name.ValueStringPointer(),
		SatisfyAny: plan.SatisfyAny.ValueBoolPointer(),
		PassAuth:   plan.PassAuth.ValueBoolPointer(),
	}

	_accessList.Items = &[]sdk.AccessListItem{}
	for _, item := range plan.Items {
		_item := sdk.AccessListItem{
			Username: item.Username.ValueStringPointer(),
			Password: item.Password.ValueStringPointer(),
		}
		*_accessList.Items = append(*_accessList.Items, _item)
	}

	_accessList.Clients = &[]sdk.AccessListClient{}
	for _, client := range plan.Clients {
		_client := sdk.AccessListClient{
			Address:   client.Address.ValueStringPointer(),
			Directive: client.Directive.ValueStringPointer(),
		}
		*_accessList.Clients = append(*_accessList.Clients, _client)
	}

	access_list, err := r.client.AccessLists.Create(_accessList)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating access list.",
			fmt.Sprintf("Unable to create access list, unexpected error: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(access_list.ID))
	plan.Name = types.StringValue(*access_list.Name)
	plan.SatisfyAny = types.BoolValue(*access_list.SatisfyAny)
	plan.PassAuth = types.BoolValue(*access_list.PassAuth)
	plan.OwnerUserID = types.Int64Value(int64(access_list.OwnerUserID))

	// We don't do items as they aren't returned from the API

	plan.Clients = []models.AccessListClientModel{}
	for _, client := range *access_list.Clients {
		_client := models.AccessListClientModel{
			Address:   types.StringValue(*client.Address),
			Directive: types.StringValue(*client.Directive),
		}
		plan.Clients = append(plan.Clients, _client)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *accessListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	var state models.AccessListModel
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

	access_list, err := r.client.AccessLists.Get(id)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read access list with ID %s.", state.ID.ValueString()),
			err.Error(),
		)
		return
	}

	state.Name = types.StringValue(*access_list.Name)
	state.PassAuth = types.BoolValue(*access_list.PassAuth)
	state.SatisfyAny = types.BoolValue(*access_list.SatisfyAny)
	state.OwnerUserID = types.Int64Value(int64(access_list.OwnerUserID))

	state.Items = accessListItems(*access_list.Items, state.Items)

	state.Clients = []models.AccessListClientModel{}
	for _, client := range *access_list.Clients {
		_client := models.AccessListClientModel{
			Address:   types.StringValue(*client.Address),
			Directive: types.StringValue(*client.Directive),
		}
		state.Clients = append(state.Clients, _client)
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *accessListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.AccessListModel
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

	var _accessList = sdk.AccessListBase{
		Name:       plan.Name.ValueStringPointer(),
		SatisfyAny: plan.SatisfyAny.ValueBoolPointer(),
		PassAuth:   plan.PassAuth.ValueBoolPointer(),
	}

	_accessList.Items = &[]sdk.AccessListItem{}
	for _, item := range plan.Items {
		_item := sdk.AccessListItem{
			Username: item.Username.ValueStringPointer(),
			Password: item.Password.ValueStringPointer(),
		}
		*_accessList.Items = append(*_accessList.Items, _item)
	}

	_accessList.Clients = &[]sdk.AccessListClient{}
	for _, client := range plan.Clients {
		_client := sdk.AccessListClient{
			Address:   client.Address.ValueStringPointer(),
			Directive: client.Directive.ValueStringPointer(),
		}
		*_accessList.Clients = append(*_accessList.Clients, _client)
	}

	access_list, err := r.client.AccessLists.Update(id, _accessList)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating updating access list.",
			fmt.Sprintf("Unable to update access list %d, unexpected error: %s", id, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(access_list.ID))
	plan.Name = types.StringValue(*access_list.Name)
	plan.SatisfyAny = types.BoolValue(*access_list.SatisfyAny)
	plan.PassAuth = types.BoolValue(*access_list.PassAuth)
	plan.OwnerUserID = types.Int64Value(int64(access_list.OwnerUserID))

	plan.Clients = []models.AccessListClientModel{}
	for _, client := range *access_list.Clients {
		_client := models.AccessListClientModel{
			Address:   types.StringValue(*client.Address),
			Directive: types.StringValue(*client.Directive),
		}
		plan.Clients = append(plan.Clients, _client)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *accessListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	var state models.AccessListModel
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

	err = r.client.AccessLists.Delete(id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting access list",
			fmt.Sprintf("Unable to delete access list, unexpected error: %s", err.Error()),
		)
	}
}

func (r *accessListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureProvider(req, resp)
}

func (r *accessListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// accessListItems returns the users of an access list as NPM has them. NPM
// never returns the passwords, so they're taken from the known users with
// the same username. Users that aren't known, e.g after an import, get an
// empty password, so the configured one is set on the next apply.
func accessListItems(items []sdk.AccessListItem, known []models.AccessListItemModel) []models.AccessListItemModel {
	passwords := map[string]types.String{}
	for _, item := range known {
		passwords[item.Username.ValueString()] = item.Password
	}

	result := []models.AccessListItemModel{}
	for _, item := range items {
		if item.Username == nil {
			continue
		}
		password, ok := passwords[*item.Username]
		if !ok || password.IsNull() || password.IsUnknown() {
			password = types.StringValue("")
		}
		result = append(result, models.AccessListItemModel{
			Username: types.StringValue(*item.Username),
			Password: password,
		})
	}
	return result
}
