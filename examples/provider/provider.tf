terraform {
  required_providers {
    skyhub = {
      source = "fredericgermain/skyhub"
    }
  }
}

# Credentials default to SKYHUB_URL / SKYHUB_USER / SKYHUB_PASSWORD, then ~/skyhub.
provider "skyhub" {
  endpoint = "http://192.168.50.1/"
}

data "skyhub_wan_status" "wan" {}

output "public_ip" {
  value = data.skyhub_wan_status.wan.ipv4
}
