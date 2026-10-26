// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// The DNS provider was checked against a fixed list, which rejected
// providers NPM supports, e.g hetzner, ovh and desec.
func TestAccLetsencryptCertificateResourceDNSProviders(t *testing.T) {

	config := func(provider string) string {
		return providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_letsencrypt_certificate" "test" {
  domain_names             = ["dns-provider-check.example.com"]
  dns_challenge            = true
  dns_provider             = "%s"
  dns_provider_credentials = "dns_example_token = abc"
  letsencrypt_email        = "test@test.com"
}
`, provider)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             config("hetzner"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config:             config("ovh"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config:      config("not-a-dns-provider"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`doesn't support the DNS provider "not-a-dns-provider"`),
			},
		},
	})
}
