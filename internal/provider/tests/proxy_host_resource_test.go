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

func TestAccProxyHostResource(t *testing.T) {

	var id = rand.IntN(1e4)
	certificate := customCertificateConfig(fmt.Sprintf("proxy_host_%d", id))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_proxy_host" "test" {
  forward_host = "test_%d.com"
  forward_port = 8080
  forward_scheme = "https"
  # Not in the sorted order NPM stores them in
  domain_names = [
	"domain2_%d.com",
	"domain1_%d.com",
  ]
  
  block_exploits          = true
  allow_websocket_upgrade = false

  location {
    path           = "/api"
    forward_scheme = "http"
    forward_host   = "10.0.0.11"
    forward_port   = 9000
  }

  location {
    path           = "/static"
    forward_scheme = "https"
    forward_host   = "10.0.0.12"
    forward_port   = 443
  }
}
`, id, id, id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nginxproxymanager_proxy_host.test", "id"),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "location.#", "2"),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "location.0.path", "/api"),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "location.0.forward_port", "9000"),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "location.1.forward_host", "10.0.0.12"),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "forward_host", fmt.Sprintf("test_%d.com", id)),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "forward_port", "8080"),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "forward_scheme", "https"),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "domain_names.#", "2"),
					resource.TestCheckTypeSetElemAttr("nginxproxymanager_proxy_host.test", "domain_names.*", fmt.Sprintf("domain1_%d.com", id)),
					resource.TestCheckTypeSetElemAttr("nginxproxymanager_proxy_host.test", "domain_names.*", fmt.Sprintf("domain2_%d.com", id)),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "block_exploits", "true"),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "allow_websocket_upgrade", "false"),
				),
			},
			{
				ResourceName:      "nginxproxymanager_proxy_host.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_proxy_host" "test" {
  forward_host = "test123_%d.com"
  forward_port = 8081
  forward_scheme = "https"
  domain_names = ["domain3_%d.com"]
  
  ssl_forced = true
}`, id, id),
				Check:       resource.ComposeAggregateTestCheckFunc(),
				ExpectError: regexp.MustCompile("ssl_forced cannot be true when no certificate_id is given"),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_proxy_host" "test" {
  forward_host = "test123_%d.com"
  forward_port = 8081
  forward_scheme = "https"
  domain_names = ["domain3_%d.com"]
  
  http2_support = true
}`, id, id),
				Check:       resource.ComposeAggregateTestCheckFunc(),
				ExpectError: regexp.MustCompile("http2_support cannot be true when no certificate_id is given"),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_proxy_host" "test" {
  forward_host = "test123_%d.com"
  forward_port = 8081
  forward_scheme = "https"
  domain_names = ["domain3_%d.com"]
  
  hsts_enabled = true
}`, id, id),
				Check:       resource.ComposeAggregateTestCheckFunc(),
				ExpectError: regexp.MustCompile("hsts_enabled cannot be true when ssl_forced is false"),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_proxy_host" "test" {
  forward_host = "test123_%d.com"
  forward_port = 8081
  forward_scheme = "https"
  domain_names = ["domain3_%d.com"]
  
  hsts_subdomains = true
}`, id, id),
				Check:       resource.ComposeAggregateTestCheckFunc(),
				ExpectError: regexp.MustCompile("hsts_subdomains cannot be true when hsts_enabled is false"),
			},
			{
				Config: providerConfig + certificate + fmt.Sprintf(`
resource "nginxproxymanager_proxy_host" "test" {
  forward_host = "test123_%d.com"
  forward_port = 8081
  forward_scheme = "https"
  domain_names = [
	"domain2_%d.com",
	"domain3_%d.com",
	"domain4_%d.com",
  ]
  
  allow_websocket_upgrade = true
  caching_enabled         = true

  location {
    path           = "/api"
    forward_scheme = "http"
    forward_host   = "10.0.0.11"
    forward_port   = 9001
  }

  certificate_id = nginxproxymanager_custom_certificate.proxy_host_%d.id
  ssl_forced     = true
}`, id, id, id, id, id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "location.#", "1"),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "location.0.forward_port", "9001"),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "ssl_forced", "true"),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "forward_host", fmt.Sprintf("test123_%d.com", id)),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "forward_port", "8081"),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "forward_scheme", "https"),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "domain_names.#", "3"),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "domain_names.0", fmt.Sprintf("domain2_%d.com", id)),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "domain_names.1", fmt.Sprintf("domain3_%d.com", id)),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "domain_names.2", fmt.Sprintf("domain4_%d.com", id)),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "block_exploits", "false"),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "allow_websocket_upgrade", "true"),
					resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "caching_enabled", "true"),
				),
			},
		},
	})
}
