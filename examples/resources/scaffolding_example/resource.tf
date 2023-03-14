resource "lambdalabs_instance" "instance1" {
  region_name = "us-west-1"
  instance_type_name = "gpu_1x_a10"
  ssh_key_names = ["laptop"]
}
