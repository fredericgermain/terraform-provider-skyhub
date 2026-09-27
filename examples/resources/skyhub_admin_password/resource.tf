# The provider logs in with its own password. When this resource sets a
# different one, the provider switches to it for the rest of the run.
# Rotating: put the new password in Vault, then apply once with the old
# password given to the provider (e.g. -var skyhub_bootstrap_password=OLD).
resource "skyhub_admin_password" "this" {
  password_wo         = data.vault_kv_secret_v2.skyhub.data["skyhub_password"]
  password_wo_version = data.vault_kv_secret_v2.skyhub.version
}
