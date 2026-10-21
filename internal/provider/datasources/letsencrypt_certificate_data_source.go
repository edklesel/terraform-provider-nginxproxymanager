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

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &letsencryptCertificateDataSource{}
	_ datasource.DataSourceWithConfigure = &letsencryptCertificateDataSource{}
)

type letsencryptCertificateDataSource struct {
	client *sdk.Client
}

func NewletsencryptCertificateDataSource() datasource.DataSource {
	return &letsencryptCertificateDataSource{}
}

func (r *letsencryptCertificateDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_letsencrypt_certificate"
}

func (r *letsencryptCertificateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the certificate.",
			},
			"domain_names": schema.ListAttribute{
				Computed:    true,
				Description: "Domain names for the certificate.",
				ElementType: types.StringType,
			},
			"dns_challenge": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether or not this certificate uses a DNS challenge.",
			},
			"dns_provider": schema.StringAttribute{
				Computed:    true,
				Description: "Which DNS provider is used for the DNS challenge..",
			},
			"dns_provider_credentials": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "Credentials for the chosen DNS provider.",
			},
			"letsencrypt_email": schema.StringAttribute{
				Computed:    true,
				Description: "Email address used for the Letsencrypt certificate.",
			},
			"propagation_seconds": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of seconds waited for the DNS challenge to propagate. Defaults to the provider default if left empty.",
			},
			"nice_name": schema.StringAttribute{
				Computed:    true,
				Description: "Nice name for the certificate.",
			},
		},
	}
}

func (r *letsencryptCertificateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	var state models.LetsencryptCertificateModel
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
	state.DNSChallenge = types.BoolValue(*certificate.Meta.DNSChallenge)
	state.DNSProvider = types.StringValue(*certificate.Meta.DNSProvider)
	state.PropagationSeconds = types.Int64Value(int64(*certificate.Meta.PropagationSeconds))

	// The DNS provider credentials are only returned from the API in <2.13.0
	if utils.CheckAttribute(r.client, &utils.ReturnDNSProviderCredentials) {
		state.DNSProviderCredentials = types.StringValue(*certificate.Meta.DNSProviderCredentials)
	}

	// The LetsencryptEmail and LetsencryptAgree are only present i
	if utils.CheckAttribute(r.client, &utils.FieldLetsencryptEmail) {
		state.LetsencryptEmail = types.StringValue(*certificate.Meta.LetsencryptEmail)
	}

	// These aren't yet implemented in the GET request to the certificate
	// plan.CN = types.StringValue(*certificate.Meta.LetsencryptCertificate.CN)
	// plan.Issuer = types.StringValue(*certificate.Meta.LetsencryptCertificate.Issuer)
	// plan.From = types.Int64Value(int64(*certificate.Meta.LetsencryptCertificate.Dates.From))
	// plan.To = types.Int64Value(int64(*certificate.Meta.LetsencryptCertificate.Dates.To))

	state.DomainNames = []types.String{}
	for _, dn := range *certificate.DomainNames {
		state.DomainNames = append(state.DomainNames, types.StringValue(dn))
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *letsencryptCertificateDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	r.client = configureProvider(req, resp)
}
