// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import "github.com/hashicorp/terraform-plugin-framework/types"

type ProxyHostLocationModel struct {
	ForwardScheme types.String `tfsdk:"forward_scheme"`
	ForwardHost   types.String `tfsdk:"forward_host"`
	ForwardPort   types.Int64  `tfsdk:"forward_port"`
	Path          types.String `tfsdk:"path"`
}

type ProxyHostModel struct {
	ID                    types.String             `tfsdk:"id"`
	ForwardScheme         types.String             `tfsdk:"forward_scheme"`
	DomainNames           []types.String           `tfsdk:"domain_names"`
	ForwardHost           types.String             `tfsdk:"forward_host"`
	ForwardPort           types.Int64              `tfsdk:"forward_port"`
	AccessListID          types.Int64              `tfsdk:"access_list_id"`
	CertificateID         types.Int64              `tfsdk:"certificate_id"`
	SSLForced             types.Bool               `tfsdk:"ssl_forced"`
	CachingEnabled        types.Bool               `tfsdk:"caching_enabled"`
	BlockExploits         types.Bool               `tfsdk:"block_exploits"`
	AdvancedConfig        types.String             `tfsdk:"advanced_config"`
	AllowWebsocketUpgrade types.Bool               `tfsdk:"allow_websocket_upgrade"`
	HTTP2Support          types.Bool               `tfsdk:"http2_support"`
	Enabled               types.Bool               `tfsdk:"enabled"`
	Locations             []ProxyHostLocationModel `tfsdk:"location"`
	HSTSEnabled           types.Bool               `tfsdk:"hsts_enabled"`
	HSTSSubdomains        types.Bool               `tfsdk:"hsts_subdomains"`
	OwnerUserID           types.Int64              `tfsdk:"owner_user_id"`
}
