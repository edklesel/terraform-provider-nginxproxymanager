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
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccAccessListResource(t *testing.T) {

	id := rand.IntN(1e4)
	name := fmt.Sprintf("test_%d", id)
	threeUsers := fmt.Sprintf(`
resource "nginxproxymanager_access_list" "test" {
  name = "%s"

  item {
    username = "username1"
	password = "password1"
  }

  item {
    username = "username2"
	password = "password2"
  }

  item {
    username = "username3"
	password = "password3"
  }

  client {
    address = "1.2.3.4"
	directive = "allow" 
  }

  client {
    address = "3.4.5.6"
	directive = "allow"
  }
}`, name)
	imported := fmt.Sprintf(`
resource "nginxproxymanager_access_list" "imported" {
  name = "%s_imported"

  item {
    username = "imported_user"
    password = "imported_password"
  }

  client {
    address   = "10.0.0.0/8"
    directive = "allow"
  }
}
`, name)
	var importedID int

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
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nginxproxymanager_access_list.test", "id"),
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "name", name),
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "item.#", "2"),
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "client.#", "3"),
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "satisfy_any", "true"),
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "pass_auth", "false"),
				),
			},
			{
				ResourceName:      "nginxproxymanager_access_list.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"item",
				},
			},
			{
				Config: providerConfig + threeUsers,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "name", name),
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "item.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs("nginxproxymanager_access_list.test", "item.*", map[string]string{"username": "username3"}),
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "client.#", "2"),
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "satisfy_any", "false"),
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "pass_auth", "true"),
				),
			},
			// A user added outside Terraform shows up as a change
			{
				Config: providerConfig + threeUsers,
				Check: func(s *terraform.State) error {
					client, id, err := accessListClient(s)
					if err != nil {
						return err
					}
					current, err := client.AccessLists.Get(id)
					if err != nil {
						return err
					}
					// Users sent without a password keep their current one
					items := []sdk.AccessListItem{}
					for _, item := range *current.Items {
						items = append(items, sdk.AccessListItem{Username: item.Username})
					}
					username, password := "username4", "password4"
					items = append(items, sdk.AccessListItem{Username: &username, Password: &password})
					update := current.AccessListBase
					update.Items = &items
					_, err = client.AccessLists.Update(id, update)
					return err
				},
				ExpectNonEmptyPlan: true,
			},
			{
				Config: providerConfig + threeUsers,
				Check:  resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "item.#", "3"),
			},
			// An access list made outside Terraform imports with its users, and
			// setting their passwords is an update
			{
				PreConfig: func() {
					client := sdk.NewClient(fmt.Sprintf("http://%s", providerHost), sdk.AuthStruct{Identity: providerUsername, Secret: providerPassword})
					if err := client.GetToken(); err != nil {
						t.Fatal(err)
					}
					listName, username, password := name+"_imported", "imported_user", "imported_password"
					address, directive := "10.0.0.0/8", "allow"
					created, err := client.AccessLists.Create(sdk.AccessListBase{
						Name:    &listName,
						Items:   &[]sdk.AccessListItem{{Username: &username, Password: &password}},
						Clients: &[]sdk.AccessListClient{{Address: &address, Directive: &directive}},
					})
					if err != nil {
						t.Fatal(err)
					}
					importedID = created.ID
				},
				Config:             providerConfig + threeUsers + imported,
				ResourceName:       "nginxproxymanager_access_list.imported",
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateIdFunc: func(*terraform.State) (string, error) {
					return strconv.Itoa(importedID), nil
				},
			},
			{
				Config: providerConfig + threeUsers + imported,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("nginxproxymanager_access_list.imported", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.imported", "item.#", "1"),
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.imported", "item.0.username", "imported_user"),
				),
			},
			// An access list with only client rules
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_access_list" "test" {
  name = "%s"

  client {
    address = "1.2.3.4"
	directive = "allow"
  }
}`, name) + imported,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "item.#", "0"),
					resource.TestCheckResourceAttr("nginxproxymanager_access_list.test", "client.#", "1"),
				),
			},
			// Deleting the access list outside Terraform, e.g in the UI, makes
			// Terraform plan to create it again instead of failing
			{
				Config: providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_access_list" "test" {
  name = "%s"

  client {
    address = "1.2.3.4"
	directive = "allow"
  }
}`, name) + imported,
				Check:              deleteInNPM("nginxproxymanager_access_list.test", func(c *sdk.Client, id int) error { return c.AccessLists.Delete(id) }),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// accessListClient returns an NPM client and the ID of the test access list.
func accessListClient(s *terraform.State) (*sdk.Client, int, error) {
	id, err := strconv.Atoi(s.RootModule().Resources["nginxproxymanager_access_list.test"].Primary.ID)
	if err != nil {
		return nil, 0, err
	}
	client := sdk.NewClient(fmt.Sprintf("http://%s", providerHost), sdk.AuthStruct{Identity: providerUsername, Secret: providerPassword})
	return client, id, client.GetToken()
}
