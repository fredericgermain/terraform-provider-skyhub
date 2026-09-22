resource "skyhub_firewall_rule" "ssh_in" {
  direction    = "in"
  service      = skyhub_service.ssh_alt.name
  action       = "allow_always"
  lan_start_ip = "192.168.50.20"
  wan_type     = "range"
  wan_start_ip = "203.0.113.1"
  wan_end_ip   = "203.0.113.9"
  logging      = "always"
}

resource "skyhub_firewall_rule" "block_telnet_out" {
  direction = "out"
  service   = "TELNET"
  action    = "block_always"
}
