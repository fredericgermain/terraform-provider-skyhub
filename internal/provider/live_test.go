package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccLiveDHCPReservation(t *testing.T) {
	liveHub(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "skyhub_dhcp_reservation" "t" {
  mac  = "02:00:00:00:00:01"
  ip   = "192.168.50.250"
  name = "tftest"
}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("skyhub_dhcp_reservation.t", tfjsonpath.New("name"), knownvalue.StringExact("tftest")),
				},
			},
			{ResourceName: "skyhub_dhcp_reservation.t", ImportState: true, ImportStateId: "02:00:00:00:00:01", ImportStateVerify: true},
			{
				Config: `resource "skyhub_dhcp_reservation" "t" {
  mac  = "02:00:00:00:00:01"
  ip   = "192.168.50.250"
  name = "tftest2"
}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("skyhub_dhcp_reservation.t", tfjsonpath.New("name"), knownvalue.StringExact("tftest2")),
				},
			},
		},
	})
}

func TestAccLiveServiceAndFirewallRule(t *testing.T) {
	liveHub(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "skyhub_service" "t" {
  name       = "tfTest"
  protocol   = "tcp"
  start_port = 65001
}
resource "skyhub_firewall_rule" "t" {
  direction    = "in"
  service      = skyhub_service.t.name
  action       = "allow_always"
  lan_start_ip = "192.168.50.250"
  enabled      = false
}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("skyhub_service.t", tfjsonpath.New("end_port"), knownvalue.Int64Exact(65001)),
					statecheck.ExpectKnownValue("skyhub_firewall_rule.t", tfjsonpath.New("enabled"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue("skyhub_firewall_rule.t", tfjsonpath.New("id"), knownvalue.StringExact("in/4/tfTest")),
				},
			},
			{ResourceName: "skyhub_firewall_rule.t", ImportState: true, ImportStateId: "in/4/tfTest", ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"lan_start_ip", "lan_type", "wan_type", "lan_end_ip", "wan_start_ip", "wan_end_ip", "lan_ipv6", "wan_start_ipv6", "wan_end_ipv6"}},
			{
				Config: `
resource "skyhub_service" "t" {
  name       = "tfTest"
  protocol   = "tcp"
  start_port = 65001
}
resource "skyhub_firewall_rule" "t" {
  direction    = "in"
  service      = skyhub_service.t.name
  action       = "allow_always"
  lan_start_ip = "192.168.50.250"
  enabled      = true
  position     = 1
}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("skyhub_firewall_rule.t", tfjsonpath.New("enabled"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue("skyhub_firewall_rule.t", tfjsonpath.New("position"), knownvalue.Int64Exact(1)),
				},
			},
		},
	})
}
