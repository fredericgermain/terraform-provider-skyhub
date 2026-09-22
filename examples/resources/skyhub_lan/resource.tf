resource "skyhub_lan" "this" {
  ip              = "192.168.50.1"
  netmask         = "255.255.255.0"
  dhcp_enabled    = true
  dhcp_pool_start = "192.168.50.2"
  dhcp_pool_end   = "192.168.50.254"
}
