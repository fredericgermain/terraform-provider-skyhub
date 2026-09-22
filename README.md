# terraform-provider-skyhub

Terraform provider for the Sky Hub (Sagemcom SR200 family) home router. The hub has no API;
the provider drives its admin web UI (HTTP Digest auth, HTML forms, per-page CSRF `sessionKey`)
through the [`skyhub`](https://github.com/fredericgermain/skyhub) Go library.

Every request to the hub is serialised by the client library, so Terraform's parallel applies
are safe without `-parallelism=1`.

## Provider configuration

```hcl
terraform {
  required_providers {
    skyhub = { source = "fredericgermain/skyhub" }
  }
}

provider "skyhub" {
  endpoint = "http://192.168.50.1/"   # default
  # username / password default to SKYHUB_USER / SKYHUB_PASSWORD,
  # then the credentials file (~/skyhub with USER= and PASSWORD= lines).
  # credentials_file = "/etc/skyhub-creds"
  # timeout = 30
}
```

## Resources

| Resource | Purpose | Import id |
|---|---|---|
| `skyhub_dhcp_reservation` | Address reservation (MAC to IPv4) | `aa:bb:cc:dd:ee:ff` |
| `skyhub_service` | Custom port service used by firewall rules | service name |
| `skyhub_firewall_rule` | Inbound/outbound rule (v4 or v6), action, addresses, logging, enabled, position | `in/4/SvcC` |
| `skyhub_lan` | LAN IP, netmask, DHCP pool (singleton; changes restart the hub) | `default` |
| `skyhub_wan` | Router mode, MTU, DMZ, WAN ping | `default` |
| `skyhub_upnp` | UPnP on/off and advertisement settings | `default` |
| `skyhub_alg` | SIP / H.323 ALG | `default` |
| `skyhub_ethernet` | LAN port mode (changes reboot the hub) | `default` |

Data sources: `skyhub_attached_devices`, `skyhub_system_stats`, `skyhub_wan_status`.

```hcl
resource "skyhub_dhcp_reservation" "nas" {
  mac  = "aa:bb:cc:dd:ee:ff"
  ip   = "192.168.50.20"
  name = "nas"
}

resource "skyhub_service" "ssh_alt" {
  name       = "SvcC"
  protocol   = "tcp"
  start_port = 7003
}

resource "skyhub_firewall_rule" "ssh_in" {
  direction    = "in"
  service      = skyhub_service.ssh_alt.name
  action       = "allow_always"
  lan_start_ip = "192.168.50.20"
}

resource "skyhub_upnp" "this" {
  enabled = false
}
```

Singleton resources (`skyhub_lan`, `skyhub_wan`, `skyhub_upnp`, `skyhub_alg`, `skyhub_ethernet`)
always have id `default`; destroying them only removes them from state. Firewall rules are
identified by direction, IP version and service name; changing `action`, addresses or `logging`
deletes and re-adds the rule, while `enabled` and `position` are applied through the list page.
The hub renders every rule as enabled; the provider reads the real flag from the hub's stored
bitmask.

Nothing in this provider reboots, factory-resets or upgrades the hub. `skyhub_ethernet` and
`skyhub_lan` changes are the only ones that make the hub restart itself, and the docs say so.

## Local development

```hcl
# ~/.terraformrc
provider_installation {
  dev_overrides {
    "fredericgermain/skyhub" = "/home/fred/go/bin"
  }
  direct {}
}
```

```sh
go install .                       # builds terraform-provider-skyhub into $GOPATH/bin
go test ./...                      # against the fake hub (fixtures from ../skyhub)
TF_ACC=1 SKYHUB_LIVE=1 go test ./internal/provider -run TestAccLive -v -timeout 20m   # live, throwaway entries only
go generate ./...                  # regenerate docs/
```

With `dev_overrides`, `terraform plan` in `examples/provider` needs no `terraform init`.
