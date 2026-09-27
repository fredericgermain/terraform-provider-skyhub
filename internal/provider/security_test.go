package provider

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

const fakeSecurityConfig = `
resource "skyhub_firewall_globals" "g" {
  ipv6_firewall             = true
  ipsec_passthrough         = true
  allow_inbound_icmpv6_echo = true
}
resource "skyhub_wireless" "b24" {
  band      = "2.4"
  enabled   = true
  ssid      = "TestSSID"
  channel   = 0
  bandwidth = "20"
}
resource "skyhub_wireless" "b5" {
  band      = "5"
  enabled   = true
  ssid      = "TestSSID"
  channel   = 36
  bandwidth = "80"
}
`

// Values match the fixtures: create adopts without posting, import and a
// second plan show no difference.
func TestAccFakeGlobalsAndWirelessNoDiff(t *testing.T) {
	h := fakeHub(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testFactories,
		Steps: []resource.TestStep{
			{
				Config: fakeSecurityConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("skyhub_wireless.b5", tfjsonpath.New("psk_set"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue("skyhub_wireless.b24", tfjsonpath.New("hidden"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue("skyhub_wireless.b24", tfjsonpath.New("id"), knownvalue.StringExact("2.4")),
				},
			},
			{ResourceName: "skyhub_firewall_globals.g", ImportState: true, ImportStateId: "default", ImportStateVerify: true},
			{ResourceName: "skyhub_wireless.b24", ImportState: true, ImportStateId: "2.4", ImportStateVerify: true, ImportStateVerifyIgnore: []string{"psk_wo_version"}},
			{ResourceName: "skyhub_wireless.b5", ImportState: true, ImportStateId: "5", ImportStateVerify: true, ImportStateVerifyIgnore: []string{"psk_wo_version"}},
			{Config: fakeSecurityConfig, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}},
		},
	})
	for _, p := range h.Posts() {
		t.Errorf("unexpected POST to %s", p.Handler)
	}
}

// A key in config is sent on create even when everything else matches;
// it never reaches the state.
func TestAccFakeWirelessSendsKeyOnCreate(t *testing.T) {
	h := fakeHub(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testFactories,
		Steps: []resource.TestStep{{
			Config: `
resource "skyhub_wireless" "b24" {
  band           = "2.4"
  enabled        = true
  ssid           = "TestSSID"
  psk_wo         = "correct horse battery"
  psk_wo_version = 1
}
`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue("skyhub_wireless.b24", tfjsonpath.New("psk_wo"), knownvalue.Null()),
			},
		}},
	})
	posts := h.Posts()
	if len(posts) != 1 || posts[0].Handler != "sky_wireless_settings.cgi" || posts[0].Values.Get("wlWpaPsk") != "correct horse battery" {
		t.Fatalf("posts = %+v", posts)
	}
}

func TestAccFakeAdminPassword(t *testing.T) {
	h := fakeHub(t)
	h.Handle("sky_password.scgi", func(w http.ResponseWriter, r *http.Request, v url.Values) bool {
		if v.Get("inOrgPassword") == h.Pass {
			h.Pass = v.Get("inPassword")
		}
		return false
	})
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testFactories,
		Steps: []resource.TestStep{
			{
				// Same as the provider's own password: adopted without a POST.
				Config: `resource "skyhub_admin_password" "a" { password_wo = "secret12" }`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("skyhub_admin_password.a", tfjsonpath.New("id"), knownvalue.StringExact("admin")),
					statecheck.ExpectKnownValue("skyhub_admin_password.a", tfjsonpath.New("password_wo"), knownvalue.Null()),
				},
			},
		},
	})
	if n := len(h.Posts()); n != 0 {
		t.Fatalf("adoption posted %d times", n)
	}
	h2 := fakeHub(t)
	h2.Handle("sky_password.scgi", func(w http.ResponseWriter, r *http.Request, v url.Values) bool {
		if v.Get("inOrgPassword") == h2.Pass {
			h2.Pass = v.Get("inPassword")
		}
		return false
	})
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testFactories,
		Steps: []resource.TestStep{{
			// A different password is sent. (Reading the hub afterwards in the
			// same process is covered by the library test; the harness's next
			// commands start a provider that still has the old password.)
			Config: `resource "skyhub_admin_password" "a" { password_wo = "N3wPassword!" }`,
		}},
	})
	if h2.Pass != "N3wPassword!" {
		t.Fatalf("hub password = %q", h2.Pass)
	}
}
