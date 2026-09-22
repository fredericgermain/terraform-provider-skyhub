data "skyhub_wan_status" "w" {}

output "public_ipv4" {
  value = data.skyhub_wan_status.w.ipv4
}
