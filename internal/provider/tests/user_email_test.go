// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// NPM lowercases the email when a user is updated, so an email with
// capitals failed with an inconsistent result on the first change.
func TestAccUserResourceEmailCase(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "nginxproxymanager_user" "test" {
  email    = "Mixed.Case@example.com"
  name     = "Mixed Case"
  nickname = "mixed"
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("must be lowercase"),
			},
		},
	})
}
