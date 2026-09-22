resource "skyhub_dhcp_reservation" "nas" {
  mac  = "aa:bb:cc:dd:ee:ff"
  ip   = "192.168.50.20"
  name = "nas"
}
