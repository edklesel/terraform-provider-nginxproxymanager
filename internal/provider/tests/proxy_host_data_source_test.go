// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProxyHostDataSource(t *testing.T) {

	var id = rand.IntN(1e4)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_proxy_host" "test" {
  forward_host = "test_%d.com"
  forward_port = 8080
  forward_scheme = "https"
  domain_names = [
	"domain1_%d.com",
	"domain2_%d.com",
  ]
  
  block_exploits          = true
  allow_websocket_upgrade = false

  location {
    path           = "/api"
    forward_scheme = "http"
    forward_host   = "10.0.0.11"
    forward_port   = 9000
  }
}

data "nginxproxymanager_proxy_host" "test" {
  id = nginxproxymanager_proxy_host.test.id
}
`, id, id, id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.nginxproxymanager_proxy_host.test", "forward_host", fmt.Sprintf("test_%d.com", id)),
					resource.TestCheckResourceAttr("data.nginxproxymanager_proxy_host.test", "forward_port", "8080"),
					resource.TestCheckResourceAttr("data.nginxproxymanager_proxy_host.test", "forward_scheme", "https"),
					resource.TestCheckResourceAttr("data.nginxproxymanager_proxy_host.test", "domain_names.#", "2"),
					resource.TestCheckResourceAttr("data.nginxproxymanager_proxy_host.test", "domain_names.0", fmt.Sprintf("domain1_%d.com", id)),
					resource.TestCheckResourceAttr("data.nginxproxymanager_proxy_host.test", "domain_names.1", fmt.Sprintf("domain2_%d.com", id)),
					resource.TestCheckResourceAttr("data.nginxproxymanager_proxy_host.test", "block_exploits", "true"),
					resource.TestCheckResourceAttr("data.nginxproxymanager_proxy_host.test", "allow_websocket_upgrade", "false"),
					resource.TestCheckResourceAttr("data.nginxproxymanager_proxy_host.test", "location.#", "1"),
					resource.TestCheckResourceAttr("data.nginxproxymanager_proxy_host.test", "location.0.path", "/api"),
					resource.TestCheckResourceAttr("data.nginxproxymanager_proxy_host.test", "location.0.forward_host", "10.0.0.11"),
				),
			},
		},
	})
}
