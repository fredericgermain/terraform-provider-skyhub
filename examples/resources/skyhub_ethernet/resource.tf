# Changing these values reboots the hub.
resource "skyhub_ethernet" "this" {
  type = "Gigabit"
  eee  = false
}
