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

// enabled was never sent to NPM, so disabling a host had no effect and
// showed up as an inconsistent result. Streams had no enabled attribute.
func TestAccEnabled(t *testing.T) {

	var id = rand.IntN(1e4)
	port := 20000 + id

	config := func(enabled bool) string {
		return providerConfig + fmt.Sprintf(`
resource "nginxproxymanager_proxy_host" "test" {
  forward_host   = "10.0.0.10"
  forward_port   = 8080
  forward_scheme = "http"
  domain_names   = ["proxy_%[1]d.example.com"]
  enabled        = %[3]t
}

resource "nginxproxymanager_dead_host" "test" {
  domain_names = ["dead_%[1]d.example.com"]
  enabled      = %[3]t
}

resource "nginxproxymanager_redirection_host" "test" {
  domain_names        = ["redirection_%[1]d.example.com"]
  forward_domain_name = "target_%[1]d.example.com"
  forward_scheme      = "https"
  forward_http_code   = 301
  enabled             = %[3]t
}

resource "nginxproxymanager_stream" "test" {
  incoming_port   = %[2]d
  forwarding_host = "10.0.0.20"
  forwarding_port = 5432
  enabled         = %[3]t
}
`, id, port, enabled)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Created disabled
			{
				Config: config(false),
				Check:  checkEnabled(false),
			},
			{
				Config: config(false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			// Enabled in an update, then disabled again
			{
				Config: config(true),
				Check:  checkEnabled(true),
			},
			{
				Config: config(false),
				Check:  checkEnabled(false),
			},
			{
				ResourceName:      "nginxproxymanager_stream.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// checkEnabled checks both the state and what NPM itself has.
func checkEnabled(enabled bool) resource.TestCheckFunc {
	value := strconv.FormatBool(enabled)
	return resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr("nginxproxymanager_proxy_host.test", "enabled", value),
		resource.TestCheckResourceAttr("nginxproxymanager_dead_host.test", "enabled", value),
		resource.TestCheckResourceAttr("nginxproxymanager_redirection_host.test", "enabled", value),
		resource.TestCheckResourceAttr("nginxproxymanager_stream.test", "enabled", value),
		func(s *terraform.State) error {
			client := sdk.NewClient(fmt.Sprintf("http://%s", providerHost), sdk.AuthStruct{Identity: providerUsername, Secret: providerPassword})
			if err := client.GetToken(); err != nil {
				return err
			}
			ids := map[string]int{}
			for _, name := range []string{"proxy_host", "dead_host", "redirection_host", "stream"} {
				id, err := strconv.Atoi(s.RootModule().Resources["nginxproxymanager_"+name+".test"].Primary.ID)
				if err != nil {
					return err
				}
				ids[name] = id
			}
			got := map[string]*bool{}
			proxy, err := client.ProxyHosts.Get(ids["proxy_host"])
			if err != nil {
				return err
			}
			got["proxy_host"] = proxy.Enabled
			dead, err := client.DeadHosts.Get(ids["dead_host"])
			if err != nil {
				return err
			}
			got["dead_host"] = dead.Enabled
			redirection, err := client.RedirectionHosts.Get(ids["redirection_host"])
			if err != nil {
				return err
			}
			got["redirection_host"] = redirection.Enabled
			stream, err := client.Streams.Get(ids["stream"])
			if err != nil {
				return err
			}
			got["stream"] = stream.Enabled
			for name, e := range got {
				if e == nil || *e != enabled {
					return fmt.Errorf("expected %s to be enabled=%t in NPM", name, enabled)
				}
			}
			return nil
		},
	)
}
