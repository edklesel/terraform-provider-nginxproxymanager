// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccStreamDataSource(t *testing.T) {

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

data "nginxproxymanager_stream" "test" {
  id = nginxproxymanager_stream.test.id
}
`, id, id, id+1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.nginxproxymanager_stream.test", "forwarding_host", fmt.Sprintf("test_%d.com", id)),
					resource.TestCheckResourceAttr("data.nginxproxymanager_stream.test", "forwarding_port", fmt.Sprintf("%d", id)),
					resource.TestCheckResourceAttr("data.nginxproxymanager_stream.test", "incoming_port", fmt.Sprintf("%d", id+1)),
					resource.TestCheckResourceAttr("data.nginxproxymanager_stream.test", "tcp_forwarding", "true"),
					resource.TestCheckResourceAttr("data.nginxproxymanager_stream.test", "udp_forwarding", "false"),
					resource.TestCheckResourceAttr("data.nginxproxymanager_stream.test", "enabled", "true"),
				),
			},
		},
	})
}
