// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"testing"

	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccUserPermissionsResource(t *testing.T) {

	id := rand.IntN(1e4)
	email := fmt.Sprintf("test_%d@example.com", id)
	permissions := map[string]string{
		"visibility":        "all",
		"proxy_hosts":       "manage",
		"redirection_hosts": "view",
		"dead_hosts":        "hidden",
		"streams":           "view",
		"access_lists":      "hidden",

		// This is set to aid with the tests,
		// but is not set in the resource definition
		"certificates": "manage",
	}

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
resource "nginxproxymanager_user_permissions" "test" {
  user_id = nginxproxymanager_user.test.id
}
`, email),
				ExpectError: regexp.MustCompile("Invalid Block"),
			},
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_user" "test" {
  name     = "test_name"
  nickname = "test_nickname"
  email    = "%s"
  roles    = ["admin"]
}
resource "nginxproxymanager_user_permissions" "test" {
  user_id = nginxproxymanager_user.test.id
  permissions {
    visibility = "%s"
	proxy_hosts = "%s"
	redirection_hosts = "%s"
	dead_hosts = "%s"
	streams = "%s"
	access_lists = "%s"
	# Certificates are left as default
  }
}`, email, permissions["visibility"], permissions["proxy_hosts"], permissions["redirection_hosts"], permissions["dead_hosts"], permissions["streams"], permissions["access_lists"]),
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						// Find the user created in the test config
						client := sdk.NewClient(
							fmt.Sprintf("http://%s", providerHost),
							sdk.AuthStruct{
								Identity: providerUsername,
								Secret:   providerPassword,
							},
						)
						err := client.GetToken()
						if err != nil {
							t.Errorf("Unable to get client token: %v", err)
						}

						users, err := client.Users.List()
						if err != nil {
							return fmt.Errorf("Error listing users: %s", err)
						}

						var user sdk.User
						for _, u := range users {
							if *u.Email == email {
								user = u
								break
							}
						}
						if user.ID == 0 {
							return fmt.Errorf("Unable to find created user.")
						}

						permission_pairs := [7][2]string{
							{*user.Permissions.Visibility, "visibility"},
							{*user.Permissions.ProxyHosts, "proxy_hosts"},
							{*user.Permissions.RedirectionHosts, "redirection_hosts"},
							{*user.Permissions.DeadHosts, "dead_hosts"},
							{*user.Permissions.Streams, "streams"},
							{*user.Permissions.AccessLists, "access_lists"},
							{*user.Permissions.Certificates, "certificates"},
						}

						for _, pair := range permission_pairs {
							if pair[0] != permissions[pair[1]] {
								return fmt.Errorf(
									"Incorrect value for \"%s\" permission: wanted \"%s\" but got \"%s\"",
									pair[1],
									pair[0],
									permissions[pair[1]],
								)
							}
						}

						return nil
					},
				),
			},
			// Deleting the user outside Terraform takes its permissions with it,
			// and Terraform plans to create both again instead of failing
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_user" "test" {
  name     = "test_name"
  nickname = "test_nickname"
  email    = "%s"
  roles    = ["admin"]
}
resource "nginxproxymanager_user_permissions" "test" {
  user_id = nginxproxymanager_user.test.id
  permissions {
    visibility = "%s"
	proxy_hosts = "%s"
	redirection_hosts = "%s"
	dead_hosts = "%s"
	streams = "%s"
	access_lists = "%s"
	# Certificates are left as default
  }
}`, email, permissions["visibility"], permissions["proxy_hosts"], permissions["redirection_hosts"], permissions["dead_hosts"], permissions["streams"], permissions["access_lists"]),
				Check:              deleteInNPM("nginxproxymanager_user.test", func(c *sdk.Client, id int) error { return c.Users.Delete(id) }),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
