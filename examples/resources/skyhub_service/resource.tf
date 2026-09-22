resource "skyhub_service" "ssh_alt" {
  name       = "SvcC"
  protocol   = "tcp"
  start_port = 7003
}
