resource "skyhub_wan" "this" {
  mtu              = 1500
  dmz_enabled      = true
  dmz_ip           = "192.168.50.200"
  respond_to_ping  = false
  respond_to_ping6 = false
}
