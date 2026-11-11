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
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &letsencryptCertificateResource{}
	_ resource.ResourceWithConfigure   = &letsencryptCertificateResource{}
	_ resource.ResourceWithImportState = &letsencryptCertificateResource{}

	dnsProviders = []string{
		"acmedns", "active24", "aliyun", "azure", "baidu", "beger", "bunny",
		"cdmon", "cloudflare", "cloudns", "cloudxns", "constellix", "corenetworks",
		"cpanel", "ddnss", "dsec", "duckdns", "digitalocean", "directadmin",
		"dnsimple", "dnsmadeeasy", "dnsmulti", "dnspod", "domainoffensive",
		"domeneshop", "dynu", "easydns", "eurodns", "firstdomains", "freedns",
		"gandi", "gcore", "godaddy", "google", "googledomains", "he", "hetner",
		"hostingnl", "hover", "infomaniak", "inwx", "ionos", "ispconfig", "isset",
		"joker", "leaseweb", "linode", "loopia", "luadns", "mijnhost", "namecheap",
		"netcup", "nictu", "njalla", "nsone", "oci", "ovn", "plesk", "porkbun",
		"powerdns", "regru", "rfc2136", "rockenstein", "route53", "spaceship",
		"strato", "selecte1v2", "timeweb", "transip", "tencentcloud", "vultr",
		"websupport", "wedos", "edgedns", "zoneedit",
	}
)

type letsencryptCertificateResource struct {
	client *sdk.Client
}

func NewLetsencryptCertificateResource() resource.Resource {
	return &letsencryptCertificateResource{}
}

func (r *letsencryptCertificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_letsencrypt_certificate"
}

func (r *letsencryptCertificateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the certificate.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_names": schema.SetAttribute{
				Required:    true,
				Description: "Domain names for the certificate. NPM stores these sorted, so the order doesn't matter.",
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
			"dns_challenge": schema.BoolAttribute{
				Required:    true,
				Description: "Whether or not to use a DNS challenge for this Letsencrypt certificate.",
				Validators: []validator.Bool{
					utils.RequireAttributesIfTrueValidator{
						RequiredAttrs: []string{"dns_provider", "dns_provider_credentials"},
					},
				},
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"dns_provider": schema.StringAttribute{
				Optional:    true,
				Description: "Which DNS provider to use for the DNS challenge. Must be set if `dns_challenge` it `true`.",
				Validators: []validator.String{
					stringvalidator.OneOf(dnsProviders...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"dns_provider_credentials": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Credentials for the provided DNS provider. Consult the Nginx Proxy Manager UI for the format for your provider.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"letsencrypt_email": schema.StringAttribute{
				Required:    true,
				Description: "Email address to use for the Letsencrypt certificate.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"propagation_seconds": schema.Int64Attribute{
				Optional:    true,
				Description: "Number of seconds to wait for the DNS challenge to propagate. Defaults to the provider default if left empty.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"nice_name": schema.StringAttribute{
				Computed:    true,
				Description: "Nice name for the certificate.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			// These aren't consistently returned by the API, so leaving them out of the
			// resource for the time being

			// "from": schema.Int64Attribute{
			// 	Computed:    true,
			// 	Description: "The date the certificate is valid from.",
			// 	PlanModifiers: []planmodifier.Int64{
			// 		int64planmodifier.UseStateForUnknown(),
			// 	},
			// },
			// "to": schema.Int64Attribute{
			// 	Computed:    true,
			// 	Description: "The date the certificate is valid until.",
			// 	PlanModifiers: []planmodifier.Int64{
			// 		int64planmodifier.UseStateForUnknown(),
			// 	},
			// },
			// "cn": schema.StringAttribute{
			// 	Computed:    true,
			// 	Description: "The Common Name of the certificate",
			// 	PlanModifiers: []planmodifier.String{
			// 		stringplanmodifier.UseStateForUnknown(),
			// 	},
			// },
			// "issuer": schema.StringAttribute{
			// 	Computed:    true,
			// 	Description: "Issuer of the certificate",
			// 	PlanModifiers: []planmodifier.String{
			// 		stringplanmodifier.UseStateForUnknown(),
			// 	},
			// },
		},
	}
}

func (r *letsencryptCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.LetsencryptCertificateModel
	diags := req.Plan.Get(ctx, &plan)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var _certificate = sdk.CertificateBase{
		DomainNames: utils.ReturnPointer([]string{}),
		Provider:    utils.ReturnPointer("letsencrypt"),
		Meta: &sdk.CertificateMeta{
			DNSChallenge: plan.DNSChallenge.ValueBoolPointer(),
		},
	}

	if utils.CheckAttribute(r.client, &utils.FieldLetsencryptEmail) {
		_certificate.Meta.LetsencryptAgree = utils.ReturnPointer(true)
		_certificate.Meta.LetsencryptEmail = plan.LetsencryptEmail.ValueStringPointer()
	}

	if plan.DNSChallenge.ValueBool() {
		_certificate.Meta.DNSProvider = plan.DNSProvider.ValueStringPointer()
		_certificate.Meta.DNSProviderCredentials = plan.DNSProviderCredentials.ValueStringPointer()

	}

	if !plan.PropagationSeconds.IsNull() {
		_certificate.Meta.PropagationSeconds = utils.ReturnPointer(int(plan.PropagationSeconds.ValueInt64()))
	}

	for _, domainName := range plan.DomainNames {
		*_certificate.DomainNames = append(*_certificate.DomainNames, domainName.ValueString())
	}

	certificate, err := r.client.Certificates.Create(_certificate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating certificate.",
			fmt.Sprintf("Unable to create certificate, unexpected error: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(certificate.ID))
	plan.NiceName = types.StringValue(*certificate.NiceName)
	plan.DNSChallenge = types.BoolValue(*certificate.Meta.DNSChallenge)
	plan.DNSProvider = types.StringValue(*certificate.Meta.DNSProvider)

	// The DNS provider credentials are only returned from the API in <2.13.0
	if utils.CheckAttribute(r.client, &utils.ReturnDNSProviderCredentials) {
		plan.DNSProviderCredentials = types.StringValue(*certificate.Meta.DNSProviderCredentials)
	}

	// The LetsencryptEmail and LetsencryptAgree are only present i
	if utils.CheckAttribute(r.client, &utils.FieldLetsencryptEmail) {
		plan.LetsencryptEmail = types.StringValue(*certificate.Meta.LetsencryptEmail)
	}

	// These aren't yet implemented in the GET request to the certificate
	// plan.CN = types.StringValue(*certificate.Meta.LetsencryptCertificate.CN)
	// plan.Issuer = types.StringValue(*certificate.Meta.LetsencryptCertificate.Issuer)
	// plan.From = types.Int64Value(int64(*certificate.Meta.LetsencryptCertificate.Dates.From))
	// plan.To = types.Int64Value(int64(*certificate.Meta.LetsencryptCertificate.Dates.To))

	if certificate.Meta.PropagationSeconds != nil {
		plan.PropagationSeconds = types.Int64Value(int64(*certificate.Meta.PropagationSeconds))
	}

	plan.DomainNames = []types.String{}
	for _, domainName := range *certificate.DomainNames {
		plan.DomainNames = append(plan.DomainNames, types.StringValue(domainName))
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *letsencryptCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	var state models.LetsencryptCertificateModel
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

	if utils.CheckAttribute(r.client, &utils.FieldLetsencryptEmail) {
		state.LetsencryptEmail = types.StringValue(*certificate.Meta.LetsencryptEmail)
	}

	// The DNS provider credentials are only returned from the API in <2.13.0
	if utils.CheckAttribute(r.client, &utils.ReturnDNSProviderCredentials) {
		state.DNSProviderCredentials = types.StringValue(*certificate.Meta.DNSProviderCredentials)
	}

	// These aren't yet implemented in the GET request to the certificate
	// state.CN = types.StringValue(*certificate.Meta.LetsencryptCertificate.CN)
	// state.Issuer = types.StringValue(*certificate.Meta.LetsencryptCertificate.Issuer)
	// state.From = types.Int64Value(int64(*certificate.Meta.LetsencryptCertificate.Dates.From))
	// state.To = types.Int64Value(int64(*certificate.Meta.LetsencryptCertificate.Dates.To))

	state.DomainNames = []types.String{}
	for _, domainName := range *certificate.DomainNames {
		state.DomainNames = append(state.DomainNames, types.StringValue(domainName))
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// The update function doesn't do anything as all attributes require replacement
// but is implements to fit the interface requirements.
func (r *letsencryptCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *letsencryptCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	var state models.LetsencryptCertificateModel
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

	err = r.client.Certificates.Delete(id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting certificate",
			fmt.Sprintf("Unable to delete certificate, unexpected error: %s", err.Error()),
		)
	}
}

func (r *letsencryptCertificateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureProvider(req, resp)
}

func (r *letsencryptCertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
