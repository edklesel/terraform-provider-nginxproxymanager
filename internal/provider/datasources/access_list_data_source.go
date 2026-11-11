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
	_ datasource.DataSource              = &accessListDataSource{}
	_ datasource.DataSourceWithConfigure = &accessListDataSource{}
)

type accessListDataSource struct {
	client *sdk.Client
}

func NewAccessListDataSource() datasource.DataSource {
	return &accessListDataSource{}
}

func (d *accessListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_list"
}

func (d *accessListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the access list.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the access list.",
			},
			"satisfy_any": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether a request passes if it satisfies any of the rules.",
			},
			"pass_auth": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether responsibility for auth passes to the host.",
			},
			"owner_user_id": schema.Int64Attribute{
				Computed:    true,
				Description: "ID of the user who owns the access list.",
			},
		},
		Blocks: map[string]schema.Block{
			"item": schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"username": schema.StringAttribute{
							Computed:    true,
							Description: "Basic auth username.",
						},
						"password": schema.StringAttribute{
							Computed:    true,
							Sensitive:   true,
							Description: "Always empty, NPM doesn't return the passwords.",
						},
					},
				},
			},
			"client": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"address": schema.StringAttribute{
							Computed:    true,
							Description: "IP Address of a client.",
						},
						"directive": schema.StringAttribute{
							Computed:    true,
							Description: "Directive, allow or deny.",
						},
					},
				},
			},
		},
	}
}

func (d *accessListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	var state models.AccessListModel
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

	access_list, err := d.client.AccessLists.Get(id)
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

	state.Items = []models.AccessListItemModel{}
	for _, item := range *access_list.Items {
		state.Items = append(state.Items, models.AccessListItemModel{
			Username: types.StringPointerValue(item.Username),
			Password: types.StringNull(),
		})
	}

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

func (d *accessListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureProvider(req, resp)
}
