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

func TestAccDeadHostResource(t *testing.T) {

	var id = rand.IntN(1e4)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_dead_host" "test" {
  # Not in the sorted order NPM stores them in
  domain_names = [
	"domain2_%d.com",
	"domain1_%d.com",
  ]
}
`, id, id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nginxproxymanager_dead_host.test", "id"),
					resource.TestCheckResourceAttr("nginxproxymanager_dead_host.test", "domain_names.#", "2"),
					resource.TestCheckTypeSetElemAttr("nginxproxymanager_dead_host.test", "domain_names.*", fmt.Sprintf("domain1_%d.com", id)),
					resource.TestCheckTypeSetElemAttr("nginxproxymanager_dead_host.test", "domain_names.*", fmt.Sprintf("domain2_%d.com", id)),
				),
			},
			{
				ResourceName:      "nginxproxymanager_dead_host.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_dead_host" "test" {
  domain_names = ["domain3_%d.com"]
  
  ssl_forced = true
}`, id),
				Check:       resource.ComposeAggregateTestCheckFunc(),
				ExpectError: regexp.MustCompile("ssl_forced cannot be true when no certificate_id is given"),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_dead_host" "test" {
  domain_names = ["domain3_%d.com"]
  
  http2_support = true
}`, id),
				Check:       resource.ComposeAggregateTestCheckFunc(),
				ExpectError: regexp.MustCompile("http2_support cannot be true when no certificate_id is given"),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_dead_host" "test" {
  domain_names = ["domain3_%d.com"]
  
  hsts_enabled = true
}`, id),
				Check:       resource.ComposeAggregateTestCheckFunc(),
				ExpectError: regexp.MustCompile("hsts_enabled cannot be true when ssl_forced is false"),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_dead_host" "test" {
  domain_names = ["domain3_%d.com"]
  
  hsts_subdomains = true
}`, id),
				Check:       resource.ComposeAggregateTestCheckFunc(),
				ExpectError: regexp.MustCompile("hsts_subdomains cannot be true when hsts_enabled is false"),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_dead_host" "test" {
  domain_names = [
	"domain2_%d.com",
	"domain3_%d.com",
	"domain4_%d.com",
  ]

  enabled = false
}`, id, id, id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nginxproxymanager_dead_host.test", "enabled", "false"),
					resource.TestCheckResourceAttr("nginxproxymanager_dead_host.test", "domain_names.#", "3"),
					resource.TestCheckResourceAttr("nginxproxymanager_dead_host.test", "domain_names.0", fmt.Sprintf("domain2_%d.com", id)),
					resource.TestCheckResourceAttr("nginxproxymanager_dead_host.test", "domain_names.1", fmt.Sprintf("domain3_%d.com", id)),
					resource.TestCheckResourceAttr("nginxproxymanager_dead_host.test", "domain_names.2", fmt.Sprintf("domain4_%d.com", id)),
				),
			},
		},
	})
}
