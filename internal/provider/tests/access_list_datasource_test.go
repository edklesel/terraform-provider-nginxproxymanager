// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAccessListDataSource(t *testing.T) {

	id := rand.IntN(1e4)
	name := fmt.Sprintf("test_%d", id)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_access_list" "test" {
  name = "%s"
  satisfy_any = true
  pass_auth = false

  item {
    username = "username1"
	password = "password1"
  }

  item {
    username = "username2"
	password = "password2"
  }

  client {
    address = "1.2.3.4"
	directive = "allow" 
  }

  client {
    address = "2.3.4.5"
	directive = "deny" 
  }

  client {
    address = "3.4.5.6"
	directive = "allow" 
  }
}

data "nginxproxymanager_access_list" "test" {
  id = nginxproxymanager_access_list.test.id
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.nginxproxymanager_access_list.test", "name", name),
					resource.TestCheckResourceAttr("data.nginxproxymanager_access_list.test", "item.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("data.nginxproxymanager_access_list.test", "item.*", map[string]string{"username": "username1"}),
					resource.TestCheckResourceAttr("data.nginxproxymanager_access_list.test", "client.#", "3"),
					resource.TestCheckResourceAttr("data.nginxproxymanager_access_list.test", "satisfy_any", "true"),
					resource.TestCheckResourceAttr("data.nginxproxymanager_access_list.test", "pass_auth", "false"),
				),
			},
		},
	})
}
