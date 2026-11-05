// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"testing"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "nginxproxymanager_user" "test" {
  name     = "test_name"
  nickname = "test_nickname"
  email    = "test@example.com"
  roles    = ["admin"]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nginxproxymanager_user.test", "id"),
					resource.TestCheckResourceAttr("nginxproxymanager_user.test", "name", "test_name"),
					resource.TestCheckResourceAttr("nginxproxymanager_user.test", "nickname", "test_nickname"),
					resource.TestCheckResourceAttr("nginxproxymanager_user.test", "email", "test@example.com"),
					resource.TestCheckResourceAttr("nginxproxymanager_user.test", "roles.0", "admin"),
				),
			},
			{
				ResourceName:      "nginxproxymanager_user.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: providerConfig + `
resource "nginxproxymanager_user" "test" {
  name     = "test_name_updated"
  nickname = "test_nickname_updated"
  email    = "test_updated@example.com"
  roles    = ["admin"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nginxproxymanager_user.test", "name", "test_name_updated"),
					resource.TestCheckResourceAttr("nginxproxymanager_user.test", "nickname", "test_nickname_updated"),
					resource.TestCheckResourceAttr("nginxproxymanager_user.test", "email", "test_updated@example.com"),
					resource.TestCheckResourceAttr("nginxproxymanager_user.test", "roles.0", "admin"),
				),
			},
			// Deleting the user outside Terraform, e.g in the UI, makes
			// Terraform plan to create it again instead of failing
			{
				Config: providerConfig + `
resource "nginxproxymanager_user" "test" {
  name     = "test_name_updated"
  nickname = "test_nickname_updated"
  email    = "test_updated@example.com"
  roles    = ["admin"]
}
`,
				Check:              deleteInNPM("nginxproxymanager_user.test", func(c *sdk.Client, id int) error { return c.Users.Delete(id) }),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
