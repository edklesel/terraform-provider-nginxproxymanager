// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"encoding/base64"
	"fmt"
	"strconv"

	provider "github.com/edklesel/terraform-provider-nginxproxymanager/internal/provider"
	sdk "github.com/edklesel/terraform-provider-nginxproxymanager/internal/sdk"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	providerHost     = "localhost:81"
	providerUsername = "admin@example.com"
	providerPassword = "changeme"
)

var (
	providerConfig = fmt.Sprintf(`
provider "nginxproxymanager" {
  host     = "%s"
  username = "%s"
  password = "%s"
  secure   = false
}
`, providerHost, providerUsername, providerPassword)
)

var (
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"nginxproxymanager": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
)

// customCertificateConfig returns config for a self-signed custom
// certificate, for tests that need a certificate on a host.
func customCertificateConfig(name string) string {
	cert, key := sdk.NewCertWithKey()
	return fmt.Sprintf(`
resource "nginxproxymanager_custom_certificate" "%s" {
  nice_name       = "%s"
  certificate     = base64decode("%s")
  certificate_key = base64decode("%s")
}
`, name, name, base64.StdEncoding.EncodeToString([]byte(cert)), base64.StdEncoding.EncodeToString([]byte(key)))
}

// deleteInNPM deletes a resource in state directly in NPM, to check the
// provider plans to create it again instead of failing.
func deleteInNPM(name string, del func(*sdk.Client, int) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource %s not found in state", name)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		client := sdk.NewClient(fmt.Sprintf("http://%s", providerHost), sdk.AuthStruct{Identity: providerUsername, Secret: providerPassword})
		if err := client.GetToken(); err != nil {
			return err
		}
		return del(client, id)
	}
}
