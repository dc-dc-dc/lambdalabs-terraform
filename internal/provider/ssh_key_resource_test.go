package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSSHKeyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccSSHKeyResourceConfig("one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("lambdalabs_sshkey.test", "name", "one"),
					resource.TestCheckResourceAttrSet("lambdalabs_sshkey.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "lambdalabs_sshkey.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccSSHKeyResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "lambdalabs_sshkey" "test" {
  name = %[1]q
  public_key = "need some here"
}
`, name)
}
