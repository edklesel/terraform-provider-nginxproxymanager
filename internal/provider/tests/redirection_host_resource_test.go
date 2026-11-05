// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRedirectionHostResource(t *testing.T) {

	var id = rand.IntN(1e4)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_redirection_host" "test" {
  forward_domain_name = "test_%d.com"
  forward_http_code = 301
  forward_scheme = "https"
  # Not in the sorted order NPM stores them in
  domain_names = [
	"domain2_%d.com",
	"domain1_%d.com",
  ]

  preserve_path = false
}
`, id, id, id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nginxproxymanager_redirection_host.test", "id"),
					resource.TestCheckResourceAttr("nginxproxymanager_redirection_host.test", "forward_domain_name", fmt.Sprintf("test_%d.com", id)),
					resource.TestCheckResourceAttr("nginxproxymanager_redirection_host.test", "forward_http_code", "301"),
					resource.TestCheckResourceAttr("nginxproxymanager_redirection_host.test", "domain_names.#", "2"),
					resource.TestCheckTypeSetElemAttr("nginxproxymanager_redirection_host.test", "domain_names.*", fmt.Sprintf("domain1_%d.com", id)),
					resource.TestCheckTypeSetElemAttr("nginxproxymanager_redirection_host.test", "domain_names.*", fmt.Sprintf("domain2_%d.com", id)),
					resource.TestCheckResourceAttr("nginxproxymanager_redirection_host.test", "preserve_path", "false"),
				),
			},
			{
				ResourceName:      "nginxproxymanager_redirection_host.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_redirection_host" "test" {
  forward_domain_name = "test123_%d.com"
  forward_http_code = 301
  forward_scheme = "https"
  domain_names = ["domain3_%d.com"]
  
  ssl_forced = true
}`, id, id),
				Check:       resource.ComposeAggregateTestCheckFunc(),
				ExpectError: regexp.MustCompile("ssl_forced cannot be true when no certificate_id is given"),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_redirection_host" "test" {
  forward_domain_name = "test123_%d.com"
  forward_http_code = 301
  forward_scheme = "https"
  domain_names = ["domain3_%d.com"]
  
  http2_support = true
}`, id, id),
				Check:       resource.ComposeAggregateTestCheckFunc(),
				ExpectError: regexp.MustCompile("http2_support cannot be true when no certificate_id is given"),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_redirection_host" "test" {
  forward_domain_name = "test123_%d.com"
  forward_http_code = 301
  forward_scheme = "https"
  domain_names = ["domain3_%d.com"]
  
  hsts_enabled = true
}`, id, id),
				Check:       resource.ComposeAggregateTestCheckFunc(),
				ExpectError: regexp.MustCompile("hsts_enabled cannot be true when ssl_forced is false"),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_redirection_host" "test" {
  forward_domain_name = "test123_%d.com"
  forward_http_code = 301
  forward_scheme = "https"
  domain_names = ["domain3_%d.com"]
  
  hsts_subdomains = true
}`, id, id),
				Check:       resource.ComposeAggregateTestCheckFunc(),
				ExpectError: regexp.MustCompile("hsts_subdomains cannot be true when hsts_enabled is false"),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_redirection_host" "test" {
  forward_domain_name = "test123_%d.com"
  forward_http_code = 302
  forward_scheme = "http"
  domain_names = [
	"domain2_%d.com",
	"domain3_%d.com",
	"domain4_%d.com",
  ]

  preserve_path = true
  enabled       = false
}`, id, id, id, id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nginxproxymanager_redirection_host.test", "enabled", "false"),
					resource.TestCheckResourceAttr("nginxproxymanager_redirection_host.test", "forward_domain_name", fmt.Sprintf("test123_%d.com", id)),
					resource.TestCheckResourceAttr("nginxproxymanager_redirection_host.test", "forward_http_code", "302"),
					resource.TestCheckResourceAttr("nginxproxymanager_redirection_host.test", "domain_names.#", "3"),
					resource.TestCheckResourceAttr("nginxproxymanager_redirection_host.test", "domain_names.0", fmt.Sprintf("domain2_%d.com", id)),
					resource.TestCheckResourceAttr("nginxproxymanager_redirection_host.test", "domain_names.1", fmt.Sprintf("domain3_%d.com", id)),
					resource.TestCheckResourceAttr("nginxproxymanager_redirection_host.test", "domain_names.2", fmt.Sprintf("domain4_%d.com", id)),
					resource.TestCheckResourceAttr("nginxproxymanager_redirection_host.test", "preserve_path", "true"),
				),
			},
		},
	})
}
