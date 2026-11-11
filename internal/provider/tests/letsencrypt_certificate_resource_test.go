// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccLetsencryptCertificateResource(t *testing.T) {

	credentials := "dns_powerdns_api_url = http://pdns:8081\r\ndns_powerdns_api_key = npm"
	certificate := func(name string) string {
		return fmt.Sprintf(`
	resource "nginxproxymanager_letsencrypt_certificate" "%s" {
	  domain_names = ["www.example.com", "2.example.com"]
	  dns_challenge = true
	  dns_provider = "powerdns"
	  dns_provider_credentials = base64decode("%s")
	  letsencrypt_email = "test@test.com"
	  propagation_seconds = 5
	}
`, name, base64.StdEncoding.EncodeToString([]byte(credentials)))
	}

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
				Config: providerConfig + certificate("test"),
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
			// Import the same certificate a second time, as someone starting
			// to manage an existing NPM would. NPM doesn't return the
			// credentials or the email, so setting them must be an update,
			// not a replacement that reissues the certificate.
			{
				Config:             providerConfig + certificate("test") + certificate("imported"),
				ResourceName:       "nginxproxymanager_letsencrypt_certificate.imported",
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return s.RootModule().Resources["nginxproxymanager_letsencrypt_certificate.test"].Primary.ID, nil
				},
			},
			{
				Config: providerConfig + certificate("test") + certificate("imported"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("nginxproxymanager_letsencrypt_certificate.imported", plancheck.ResourceActionUpdate),
					},
				},
			},
			{
				Config: providerConfig + certificate("test") + certificate("imported"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			// Both addresses point at the same certificate, so forget the
			// first one, or the certificate would be deleted twice
			{
				Config: providerConfig + certificate("imported") + `
	removed {
	  from = nginxproxymanager_letsencrypt_certificate.test
	  lifecycle {
	    destroy = false
	  }
	}
`,
			},
		},
	})
}
