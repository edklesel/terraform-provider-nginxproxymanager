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
	_ datasource.DataSource              = &customCertificateDataSource{}
	_ datasource.DataSourceWithConfigure = &customCertificateDataSource{}
)

type customCertificateDataSource struct {
	client *sdk.Client
}

func NewCustomCertificateDataSource() datasource.DataSource {
	return &customCertificateDataSource{}
}

func (r *customCertificateDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_certificate"
}

func (r *customCertificateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the certificate.",
			},
			"nice_name": schema.StringAttribute{
				Computed:    true,
				Description: "Nice name for the certificate.",
			},
			"certificate": schema.StringAttribute{
				Computed:    true,
				Description: "Body of the certificate in PEM format.",
			},
			"certificate_key": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "Private key of the certificate in PEM format.",
			},
			// Not implemented in the return from the API request
			"intermediate_certificate": schema.StringAttribute{
				Optional:    true,
				Description: "Intermediate certificate in the chain. **This is implemented to be consistent with the Certificate Resource, but will always be blank as it's not returned by the API.**",
			},
		},
	}
}

func (r *customCertificateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	var state models.CustomCertificateModel
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

	certificate, err := r.client.Certificates.Get(id)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read certificate with ID %s.", state.ID.ValueString()),
			err.Error(),
		)
		return
	}

	state.NiceName = types.StringValue(*certificate.NiceName)
	state.Certificate = types.StringValue(*certificate.Meta.Certificate)
	state.CertificateKey = types.StringValue(*certificate.Meta.CertificateKey)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *customCertificateDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	r.client = configureProvider(req, resp)
}
