// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccLetsencryptCertificateResource(t *testing.T) {

	credentials := "dns_powerdns_api_url = http://pdns:8081\r\ndns_powerdns_api_key = npm"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
	resource "nginxproxymanager_letsencrypt_certificate" "test" {
	  domain_names = ["www.example.com"]
	  letsencrypt_email = "test@test.com"
	  dns_challenge = true
	}
`,
				ExpectError: regexp.MustCompile("Attribute 'dns_provider(|_credentials)' must be set when 'dns_challenge' is"),
			},
			// The DNS provider is checked against the providers NPM supports
			{
				Config: providerConfig + `
	resource "nginxproxymanager_letsencrypt_certificate" "test" {
	  domain_names = ["www.example.com"]
	  letsencrypt_email = "test@test.com"
	  dns_challenge = true
	  dns_provider = "hetzner"
	  dns_provider_credentials = "dns_hetzner_api_token = abc"
	}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: providerConfig + `
	resource "nginxproxymanager_letsencrypt_certificate" "test" {
	  domain_names = ["www.example.com"]
	  letsencrypt_email = "test@test.com"
	  dns_challenge = true
	  dns_provider = "not-a-dns-provider"
	  dns_provider_credentials = "dns_example_token = abc"
	}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`doesn't support the DNS provider "not-a-dns-provider"`),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
	resource "nginxproxymanager_letsencrypt_certificate" "test" {
	  domain_names = ["www.example.com", "2.example.com"]
	  dns_challenge = true
	  dns_provider = "powerdns"
	  dns_provider_credentials = base64decode("%s")
	  letsencrypt_email = "test@test.com"
	  propagation_seconds = 5
	}
`, base64.StdEncoding.EncodeToString([]byte(credentials))),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nginxproxymanager_letsencrypt_certificate.test", "dns_provider", "powerdns"),
					resource.TestCheckResourceAttr("nginxproxymanager_letsencrypt_certificate.test", "dns_challenge", "true"),
					resource.TestCheckResourceAttr("nginxproxymanager_letsencrypt_certificate.test", "domain_names.#", "2"),
					resource.TestCheckTypeSetElemAttr("nginxproxymanager_letsencrypt_certificate.test", "domain_names.*", "2.example.com"),
				),
			},
			{
				ResourceName:      "nginxproxymanager_letsencrypt_certificate.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"dns_provider_credentials",
					"letsencrypt_email",
				},
			},
		},
	})
}
