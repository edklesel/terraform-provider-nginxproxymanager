// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"testing"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
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
			// Deleting the letsencrypt certificate outside Terraform, e.g in the UI, makes
			// Terraform plan to create it again instead of failing
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
				Check:              deleteInNPM("nginxproxymanager_letsencrypt_certificate.test", func(c *sdk.Client, id int) error { return c.Certificates.Delete(id) }),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
