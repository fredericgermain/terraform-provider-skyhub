resource "skyhub_firewall_globals" "this" {
  ipv6_firewall             = true
  ipsec_passthrough         = true
  allow_inbound_icmpv6_echo = true
}
