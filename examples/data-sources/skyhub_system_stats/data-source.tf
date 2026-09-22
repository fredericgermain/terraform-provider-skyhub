data "skyhub_system_stats" "s" {}

output "dsl_down_kbps" {
  value = data.skyhub_system_stats.s.dsl_down_kbps
}
