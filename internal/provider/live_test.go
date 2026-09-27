package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
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

// liveOrderedConfig appends three throwaway rules after the hub's existing
// inbound rules (assumes six, as on the reference hub), created in order
// through a depends_on chain the way layer2_network does it, and adopts the
// firewall globals with their current values (no POST).
const liveOrderedConfig = `
resource "skyhub_firewall_globals" "g" {
  ipv6_firewall             = true
  ipsec_passthrough         = true
  allow_inbound_icmpv6_echo = true
}
resource "skyhub_service" "a" {
  name       = "tfA"
  protocol   = "udp"
  start_port = 65011
}
resource "skyhub_service" "b" {
  name       = "tfB"
  protocol   = "udp"
  start_port = 65012
}
resource "skyhub_service" "c" {
  name       = "tfC"
  protocol   = "udp"
  start_port = 65013
}
resource "skyhub_firewall_rule" "a" {
  direction    = "in"
  service      = skyhub_service.a.name
  action       = "allow_always"
  lan_start_ip = "192.168.50.250"
  enabled      = false
  position     = 7
}
resource "skyhub_firewall_rule" "b" {
  direction    = "in"
  service      = skyhub_service.b.name
  action       = "allow_always"
  lan_start_ip = "192.168.50.250"
  enabled      = false
  position     = 8
  depends_on   = [skyhub_firewall_rule.a]
}
resource "skyhub_firewall_rule" "c" {
  direction    = "in"
  service      = skyhub_service.c.name
  action       = "allow_always"
  lan_start_ip = "192.168.50.250"
  enabled      = false
  position     = 9
  depends_on   = [skyhub_firewall_rule.b]
}
`

func TestAccLiveOrderedRulesAndGlobals(t *testing.T) {
	liveHub(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testFactories,
		Steps: []resource.TestStep{
			{
				Config: liveOrderedConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("skyhub_firewall_rule.a", tfjsonpath.New("position"), knownvalue.Int64Exact(7)),
					statecheck.ExpectKnownValue("skyhub_firewall_rule.b", tfjsonpath.New("position"), knownvalue.Int64Exact(8)),
					statecheck.ExpectKnownValue("skyhub_firewall_rule.c", tfjsonpath.New("position"), knownvalue.Int64Exact(9)),
					statecheck.ExpectKnownValue("skyhub_firewall_rule.c", tfjsonpath.New("enabled"), knownvalue.Bool(false)),
				},
			},
			{ResourceName: "skyhub_firewall_globals.g", ImportState: true, ImportStateId: "default", ImportStateVerify: true},
			{Config: liveOrderedConfig, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}},
		},
	})
}
