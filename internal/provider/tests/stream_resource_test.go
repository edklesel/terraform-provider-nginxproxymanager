// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccStreamResource(t *testing.T) {

	var id = rand.IntN(1e4)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_stream" "test" {
  forwarding_host = "test_%d.com"
  forwarding_port = %d
  incoming_port   = %d
}
`, id, id, id+1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nginxproxymanager_stream.test", "id"),
					resource.TestCheckResourceAttr("nginxproxymanager_stream.test", "forwarding_host", fmt.Sprintf("test_%d.com", id)),
					resource.TestCheckResourceAttr("nginxproxymanager_stream.test", "forwarding_port", fmt.Sprintf("%d", id)),
					resource.TestCheckResourceAttr("nginxproxymanager_stream.test", "incoming_port", fmt.Sprintf("%d", id+1)),
					resource.TestCheckResourceAttr("nginxproxymanager_stream.test", "tcp_forwarding", "true"),
					resource.TestCheckResourceAttr("nginxproxymanager_stream.test", "udp_forwarding", "false"),
					resource.TestCheckResourceAttr("nginxproxymanager_stream.test", "enabled", "true"),
				),
			},
			{
				ResourceName:      "nginxproxymanager_stream.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_stream" "test" {
  forwarding_host = "test_%d.com"
  forwarding_port = %d
  incoming_port   = %d
  
  tcp_forwarding  = false
  udp_forwarding = true
  enabled        = false
}`, id, id+1, id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nginxproxymanager_stream.test", "enabled", "false"),
					resource.TestCheckResourceAttr("nginxproxymanager_stream.test", "forwarding_host", fmt.Sprintf("test_%d.com", id)),
					resource.TestCheckResourceAttr("nginxproxymanager_stream.test", "forwarding_port", fmt.Sprintf("%d", id+1)),
					resource.TestCheckResourceAttr("nginxproxymanager_stream.test", "incoming_port", fmt.Sprintf("%d", id)),
					resource.TestCheckResourceAttr("nginxproxymanager_stream.test", "tcp_forwarding", "false"),
					resource.TestCheckResourceAttr("nginxproxymanager_stream.test", "udp_forwarding", "true"),
				),
			},
		},
	})
}
