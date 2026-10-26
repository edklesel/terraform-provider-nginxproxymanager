// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"testing"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Objects deleted outside Terraform, e.g in the NPM UI, made every plan and
// destroy fail with "<object> with id <id> not found". They're now removed
// from state, so Terraform plans to create them again.
func TestAccDeletedOutsideTerraform(t *testing.T) {

	var id = rand.IntN(1e4)
	config := providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_proxy_host" "test" {
  forward_host   = "10.0.0.10"
  forward_port   = 8080
  forward_scheme = "http"
  domain_names   = ["deleted_%[1]d.example.com"]
}

resource "nginxproxymanager_stream" "test" {
  incoming_port   = %[2]d
  forwarding_host = "10.0.0.20"
  forwarding_port = 5432
}

resource "nginxproxymanager_access_list" "test" {
  name = "deleted_%[1]d"
  item {
    username = "alice"
    password = "alice-password"
  }
  client {
    address   = "10.0.0.0/8"
    directive = "allow"
  }
}

resource "nginxproxymanager_user" "test" {
  email    = "deleted_%[1]d@example.com"
  name     = "Deleted %[1]d"
  nickname = "deleted"
}

resource "nginxproxymanager_user_permissions" "test" {
  user_id = nginxproxymanager_user.test.id
  permissions {
    visibility = "user"
  }
}

resource "nginxproxymanager_user_password" "test" {
  user_id  = nginxproxymanager_user.test.id
  password = "deleted-password"
}
`, id, 20000+id)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			// Delete everything in NPM, the next plan wants to create it all
			{
				Config:             config,
				Check:              deleteInNPM(),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nginxproxymanager_proxy_host.test", "id"),
					resource.TestCheckResourceAttrSet("nginxproxymanager_user.test", "id"),
				),
			},
			// And destroying works when they're already gone
			{
				Config:             config,
				Check:              deleteInNPM(),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func deleteInNPM() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := sdk.NewClient(fmt.Sprintf("http://%s", providerHost), sdk.AuthStruct{Identity: providerUsername, Secret: providerPassword})
		if err := client.GetToken(); err != nil {
			return err
		}
		ids := map[string]int{}
		for _, name := range []string{"proxy_host", "stream", "access_list", "user"} {
			id, err := strconv.Atoi(s.RootModule().Resources["nginxproxymanager_"+name+".test"].Primary.ID)
			if err != nil {
				return err
			}
			ids[name] = id
		}
		for _, err := range []error{
			client.ProxyHosts.Delete(ids["proxy_host"]),
			client.Streams.Delete(ids["stream"]),
			client.AccessLists.Delete(ids["access_list"]),
			client.Users.Delete(ids["user"]),
		} {
			if err != nil {
				return err
			}
		}
		return nil
	}
}
