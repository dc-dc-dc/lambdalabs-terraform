package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccInstanceResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccExampleResourceConfig("gpu_1x_a10", "us-west-1", "laptop"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("lambdalabs_instance.test", "instance_type_name", "gpu_1x_a10"),
					resource.TestCheckResourceAttr("lambdalabs_instance.test", "region_name", "us-west-1"),
					// resource.TestCheckResourceAttr("lambdalabs_instance.test", "id", "example-id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "lambdalabs_instance.test",
				ImportState:       true,
				ImportStateVerify: true,
				// This is not normally necessary, but is here because this
				// example code does not have an actual upstream service.
				// Once the Read method is able to refresh information from
				// the upstream service, this can be removed.
				// ImportStateVerifyIgnore: []string{"configurable_attribute"},
			},
			// Update and Read testing
			// {
			// 	Config: testAccExampleResourceConfig("two"),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		resource.TestCheckResourceAttr("lambdalabs_instance.test", "configurable_attribute", "two"),
			// 	),
			// },
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccExampleResourceConfig(instance, region, ssh_key string) string {
	return fmt.Sprintf(`
resource "lambdalabs_instance" "test" {
  region_name = %[1]q
  instance_type_name = %[2]q
  ssh_key_names = [%[3]q]
}
`, region, instance, ssh_key)
}
