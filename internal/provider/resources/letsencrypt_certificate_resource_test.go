// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// A certificate made with an HTTP challenge, e.g in the NPM UI, has no DNS
// settings in its meta, which used to crash the provider when read.
func TestCertificateMetaValues(t *testing.T) {
	provider, seconds, challenge := "powerdns", 5, true
	cases := map[string]struct {
		meta      *sdk.CertificateMeta
		challenge types.Bool
		provider  types.String
		seconds   types.Int64
	}{
		"no meta":        {nil, types.BoolValue(false), types.StringNull(), types.Int64Null()},
		"http challenge": {&sdk.CertificateMeta{}, types.BoolValue(false), types.StringNull(), types.Int64Null()},
		"dns challenge": {
			&sdk.CertificateMeta{DNSChallenge: &challenge, DNSProvider: &provider, PropagationSeconds: &seconds},
			types.BoolValue(true), types.StringValue("powerdns"), types.Int64Value(5),
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			gotChallenge, gotProvider, gotSeconds := certificateMetaValues(c.meta)
			if !gotChallenge.Equal(c.challenge) || !gotProvider.Equal(c.provider) || !gotSeconds.Equal(c.seconds) {
				t.Errorf("got %s %s %s, want %s %s %s", gotChallenge, gotProvider, gotSeconds, c.challenge, c.provider, c.seconds)
			}
		})
	}
}
