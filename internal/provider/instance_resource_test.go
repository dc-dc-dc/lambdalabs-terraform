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
			{
				Config: testAccExampleResourceConfig("gpu_1x_a10", "us-west-1", "laptop"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("lambdalabs_instance.test", "instance_type_name", "gpu_1x_a10"),
					resource.TestCheckResourceAttr("lambdalabs_instance.test", "region_name", "us-west-1"),
					resource.TestCheckResourceAttrSet("lambdalabs_instance.test", "id"),
				),
			},
			{
				ResourceName:      "lambdalabs_instance.test",
				ImportState:       true,
				ImportStateVerify: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("lambdalabs_instance.test", "ip"),
					resource.TestCheckResourceAttrSet("lambdalabs_instance.test", "status"),
				),
			},
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
