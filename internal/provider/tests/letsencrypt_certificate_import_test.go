// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Importing a certificate and applying the config it came from replaced
// (reissued) it, as NPM doesn't return the DNS provider credentials or the
// Letsencrypt email, and both forced a replacement.
func TestAccLetsencryptCertificateResourceImportNoReplace(t *testing.T) {

	const name = "nginxproxymanager_letsencrypt_certificate.test"
	const imported = "nginxproxymanager_letsencrypt_certificate.imported"
	credentials := "dns_powerdns_api_url = http://pdns:8081\r\ndns_powerdns_api_key = npm"
	certificate := func(name string) string {
		return fmt.Sprintf(`
resource "nginxproxymanager_letsencrypt_certificate" "%s" {
  domain_names             = ["2.example.com", "www.example.com"]
  dns_challenge            = true
  dns_provider             = "powerdns"
  dns_provider_credentials = base64decode("%s")
  letsencrypt_email        = "test@test.com"
  propagation_seconds      = 5
}
`, name, base64.StdEncoding.EncodeToString([]byte(credentials)))
	}
	config := providerConfig + certificate("test")
	// The same certificate imported a second time, as it would be by
	// someone starting to manage an existing NPM with Terraform
	importConfig := config + certificate("imported")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				Config:             importConfig,
				ResourceName:       imported,
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return s.RootModule().Resources[name].Primary.ID, nil
				},
			},
			// The imported certificate has no credentials or email in
			// state, which must be an update, not a replacement
			{
				Config: importConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(imported, plancheck.ResourceActionUpdate),
					},
				},
			},
			{
				Config: importConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}
