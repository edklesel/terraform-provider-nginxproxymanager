// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"math/rand/v2"
	"testing"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccUserPasswordResource(t *testing.T) {

	id := rand.IntN(1e4)
	username := fmt.Sprintf("test_%d@example.com", id)
	password := "password12345"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_user" "test" {
  name     = "test_name"
  nickname = "test_nickname"
  email    = "%s"
  roles    = ["admin"]
}
resource "nginxproxymanager_user_password" "test" {
  user_id  = nginxproxymanager_user.test.id
  password = "%s"
}`, username, password),
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {

						client := sdk.NewClient(
							fmt.Sprintf("http://%s", providerHost),
							sdk.AuthStruct{
								Identity: username,
								Secret:   password,
							},
						)
						err := client.GetToken()

						if err != nil {
							return fmt.Errorf("Error authenticating as user: %s", err)
						} else if client.Token == "" {
							return fmt.Errorf("Unable to obtain token for user.")
						}

						return nil
					},
				),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_user" "test" {
  name     = "test_name"
  nickname = "test_nickname"
  email    = "%s"
  roles    = ["admin"]
}
resource "nginxproxymanager_user_password" "test" {
  user_id  = nginxproxymanager_user.test.id
  password = "%s_updated"
}`, username, password),
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {

						client := sdk.NewClient(
							fmt.Sprintf("http://%s", providerHost),
							sdk.AuthStruct{
								Identity: username,
								Secret:   fmt.Sprintf("%s_updated", password),
							},
						)
						err := client.GetToken()

						if err != nil {
							return fmt.Errorf("Error authenticating as user after updating password: %s", err)
						} else if client.Token == "" {
							return fmt.Errorf("Unable to obtain token for user after updating password.")
						}

						return nil
					},
				),
			},
			// Deleting the user outside Terraform takes its password with it,
			// and Terraform plans to create both again instead of failing
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_user" "test" {
  name     = "test_name"
  nickname = "test_nickname"
  email    = "%s"
  roles    = ["admin"]
}
resource "nginxproxymanager_user_password" "test" {
  user_id  = nginxproxymanager_user.test.id
  password = "%s_updated"
}`, username, password),
				Check:              deleteInNPM("nginxproxymanager_user.test", func(c *sdk.Client, id int) error { return c.Users.Delete(id) }),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
