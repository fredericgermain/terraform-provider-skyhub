package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccFakeDataSources(t *testing.T) {
	fakeHub(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testFactories,
		Steps: []resource.TestStep{{
			Config: `
data "skyhub_attached_devices" "all" {}
data "skyhub_system_stats" "s" {}
data "skyhub_wan_status" "w" {}
`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue("data.skyhub_system_stats.s", tfjsonpath.New("ports"), knownvalue.ListSizeExact(4)),
				statecheck.ExpectKnownValue("data.skyhub_system_stats.s", tfjsonpath.New("ports").AtSliceIndex(0).AtMapKey("key"), knownvalue.StringExact("wan")),
				statecheck.ExpectKnownValue("data.skyhub_system_stats.s", tfjsonpath.New("dsl_down_kbps"), knownvalue.Int64Func(func(v int64) error {
					if v < 1000 {
						return errRange("dsl_down_kbps", v)
					}
					return nil
				})),
				statecheck.ExpectKnownValue("data.skyhub_wan_status.w", tfjsonpath.New("up"), knownvalue.Bool(true)),
				statecheck.ExpectKnownValue("data.skyhub_wan_status.w", tfjsonpath.New("firmware"), knownvalue.StringExact("7.04.0208.R")),
				statecheck.ExpectKnownValue("data.skyhub_wan_status.w", tfjsonpath.New("dns"), knownvalue.ListSizeExact(2)),
				statecheck.ExpectKnownValue("data.skyhub_attached_devices.all", tfjsonpath.New("devices").AtSliceIndex(0).AtMapKey("mac"), knownvalue.StringRegexp(regexp.MustCompile(`^02:00:00:00:00:`))),
			},
		}},
	})
}

func TestAccFakeSingletonsImportNoDiff(t *testing.T) {
	fakeHub(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testFactories,
		Steps: []resource.TestStep{
			{
				// Values match the fixture, so Create adopts without posting.
				Config: `
resource "skyhub_upnp" "u" { enabled = true }
resource "skyhub_alg" "a" {
  sip  = true
  h323 = true
}
resource "skyhub_ethernet" "e" { type = "Gigabit" }
resource "skyhub_wan" "w" { mtu = 1500 }
resource "skyhub_lan" "l" {
  ip              = "192.168.50.1"
  netmask         = "255.255.255.0"
  dhcp_pool_start = "192.168.50.2"
  dhcp_pool_end   = "192.168.50.254"
}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("skyhub_upnp.u", tfjsonpath.New("advertise_interval"), knownvalue.Int64Exact(30)),
					statecheck.ExpectKnownValue("skyhub_ethernet.e", tfjsonpath.New("eee"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue("skyhub_wan.w", tfjsonpath.New("dmz_ip"), knownvalue.StringExact("192.168.50.200")),
					statecheck.ExpectKnownValue("skyhub_lan.l", tfjsonpath.New("lease_hours"), knownvalue.Int64Exact(24)),
				},
			},
			{ResourceName: "skyhub_upnp.u", ImportState: true, ImportStateId: "default", ImportStateVerify: true},
			{ResourceName: "skyhub_alg.a", ImportState: true, ImportStateId: "default", ImportStateVerify: true},
			{ResourceName: "skyhub_ethernet.e", ImportState: true, ImportStateId: "default", ImportStateVerify: true},
			{ResourceName: "skyhub_wan.w", ImportState: true, ImportStateId: "default", ImportStateVerify: true},
			{ResourceName: "skyhub_lan.l", ImportState: true, ImportStateId: "default", ImportStateVerify: true},
			{
				Config: `
resource "skyhub_upnp" "u" { enabled = true }
resource "skyhub_alg" "a" {
  sip  = true
  h323 = true
}
resource "skyhub_ethernet" "e" { type = "Gigabit" }
resource "skyhub_wan" "w" { mtu = 1500 }
resource "skyhub_lan" "l" {
  ip              = "192.168.50.1"
  netmask         = "255.255.255.0"
  dhcp_pool_start = "192.168.50.2"
  dhcp_pool_end   = "192.168.50.254"
}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			},
		},
	})
}

func TestAccFakeDHCPReservationAdopt(t *testing.T) {
	h := fakeHub(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "skyhub_dhcp_reservation" "r" {
  mac  = "02:00:00:00:00:0a"
  ip   = "192.168.50.10"
  name = "nanobox"
}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("skyhub_dhcp_reservation.r", tfjsonpath.New("index"), knownvalue.Int64Exact(1)),
					statecheck.ExpectKnownValue("skyhub_dhcp_reservation.r", tfjsonpath.New("id"), knownvalue.StringExact("02:00:00:00:00:0a")),
				},
			},
			{ResourceName: "skyhub_dhcp_reservation.r", ImportState: true, ImportStateId: "02:00:00:00:00:0a", ImportStateVerify: true},
			{
				Config: `
resource "skyhub_firewall_rule" "existing" {
  direction    = "in"
  service      = "SvcC"
  action       = "allow_always"
  lan_start_ip = "192.168.0.4"
}
`,
				ImportState: true, ImportStateId: "in/4/SvcC", ResourceName: "skyhub_firewall_rule.existing",
				ImportStatePersist: false,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 imported state, got %d", len(states))
					}
					a := states[0].Attributes
					want := map[string]string{"direction": "in", "ip_version": "4", "service": "SvcC", "action": "allow_always", "position": "4", "enabled": "true", "logging": "never"}
					for k, v := range want {
						if a[k] != v {
							return fmt.Errorf("%s = %q, want %q", k, a[k], v)
						}
					}
					return nil
				},
			},
		},
	})
	// The fake records POSTs: adopting an identical reservation must not post.
	for _, p := range h.Posts() {
		if p.Handler == "sky_lanaddmac.sky" && p.Values.Get("action") == "add" {
			t.Errorf("unexpected POST to %s: %v", p.Handler, p.Values)
		}
	}
}

type rangeErr struct {
	what string
	v    int64
}

func (e rangeErr) Error() string          { return e.what + " out of range" }
func errRange(what string, v int64) error { return rangeErr{what, v} }
