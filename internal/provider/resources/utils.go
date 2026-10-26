// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
)

func configureProvider(req resource.ConfigureRequest, resp *resource.ConfigureResponse) (client *sdk.Client) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sdk.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected configuration type",
			fmt.Sprintf("Expected *sdk.Client, got: %T. Please report this issue to the provider developer.", req.ProviderData),
		)
		return
	}

	return client
}

type enabler interface {
	SetEnabled(id int, enabled bool) error
}

// applyEnabled enables or disables an object through NPM's enable/disable
// endpoints when the planned value differs from what NPM has, and returns
// the resulting value. NPM ignores enabled in create and update requests.
func applyEnabled(client enabler, id int, current *bool, planned types.Bool) (types.Bool, error) {
	isEnabled := current != nil && *current
	if planned.IsNull() || planned.IsUnknown() || planned.ValueBool() == isEnabled {
		return types.BoolValue(isEnabled), nil
	}
	if err := client.SetEnabled(id, planned.ValueBool()); err != nil {
		return types.BoolValue(isEnabled), err
	}
	return planned, nil
}
