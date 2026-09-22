data "skyhub_attached_devices" "all" {}

output "hostnames" {
  value = [for d in data.skyhub_attached_devices.all.devices : d.hostname]
}
