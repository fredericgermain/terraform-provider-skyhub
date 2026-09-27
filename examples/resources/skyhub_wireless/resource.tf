# The key is write-only: it is sent on create and whenever psk_wo_version
# changes, and never stored in state. Using the Vault secret's version makes
# a key rotation in Vault reach the hub on the next apply.
data "vault_kv_secret_v2" "skyhub" {
  mount = "secret"
  name  = "home/skyhub"
}

resource "skyhub_wireless" "band24" {
  band           = "2.4"
  enabled        = true
  ssid           = data.vault_kv_secret_v2.skyhub.data["wifi_ssid"]
  channel        = 0 # auto
  bandwidth      = "20"
  psk_wo         = data.vault_kv_secret_v2.skyhub.data["wifi_psk"]
  psk_wo_version = data.vault_kv_secret_v2.skyhub.version
}

resource "skyhub_wireless" "band5" {
  band           = "5"
  enabled        = true
  ssid           = data.vault_kv_secret_v2.skyhub.data["wifi_ssid"]
  channel        = 36
  bandwidth      = "80"
  psk_wo         = data.vault_kv_secret_v2.skyhub.data["wifi_psk"]
  psk_wo_version = data.vault_kv_secret_v2.skyhub.version
}
