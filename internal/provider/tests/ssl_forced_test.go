// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"encoding/base64"
	"fmt"
	"math/rand/v2"
	"testing"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// ssl_forced wasn't sent to NPM when creating or updating proxy and dead
// hosts, or when updating redirection hosts.
func TestAccSSLForced(t *testing.T) {

	var id = rand.IntN(1e4)
	cert, key := sdk.NewCertWithKey()

	config := func(sslForced bool) string {
		return providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_custom_certificate" "test" {
  nice_name       = "ssl_forced_%[1]d"
  certificate     = base64decode("%[2]s")
  certificate_key = base64decode("%[3]s")
}

resource "nginxproxymanager_proxy_host" "test" {
  forward_host   = "10.0.0.10"
  forward_port   = 8080
  forward_scheme = "http"
  domain_names   = ["proxy_%[1]d.example.com"]
  certificate_id = nginxproxymanager_custom_certificate.test.id
  ssl_forced     = %[4]t
}

resource "nginxproxymanager_dead_host" "test" {
  domain_names   = ["dead_%[1]d.example.com"]
  certificate_id = nginxproxymanager_custom_certificate.test.id
  ssl_forced     = %[4]t
}

resource "nginxproxymanager_redirection_host" "test" {
  domain_names        = ["redirection_%[1]d.example.com"]
  forward_domain_name = "target_%[1]d.example.com"
  forward_scheme      = "https"
  forward_http_code   = 301
  certificate_id      = nginxproxymanager_custom_certificate.test.id
  ssl_forced          = %[4]t
}
`, id, base64.StdEncoding.EncodeToString([]byte(cert)), base64.StdEncoding.EncodeToString([]byte(key)), sslForced)
	}

	checks := func(sslForced bool) resource.TestCheckFunc {
		value := fmt.Sprintf("%t", sslForced)
		return resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "ssl_forced", value),
			resource.TestCheckResourceAttr("nginxproxymanager_dead_host.test", "ssl_forced", value),
			resource.TestCheckResourceAttr("nginxproxymanager_redirection_host.test", "ssl_forced", value),
		)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Created with ssl_forced
			{
				Config: config(true),
				Check:  checks(true),
			},
			// Updated to turn it off, then on again
			{
				Config: config(false),
				Check:  checks(false),
			},
			{
				Config: config(true),
				Check:  checks(true),
			},
			{
				Config: config(true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}
