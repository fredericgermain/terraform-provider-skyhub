# AGENTS.md

Terraform provider for the Sky Hub router, on the Registry as `fredericgermain/skyhub`, built on
`github.com/fredericgermain/skyhub` (`pkg/skyhub`). Resources and their quirks: README; `docs/`
is generated (`go generate ./...`), edit the templates and schemas, not the output.

- **Public repo: nothing from a real network goes in**, not in code, tests, examples, docs or
  commit messages: no SSID, LAN subnet, IPv6 prefix, MAC, host or device name, custom service
  name or port, secret-store path, public IP. Stand-ins: `192.168.50.0/24`, `2001:db8::/32`,
  MACs `02:00:00:…`, services `SvcA`–`SvcD` on ports 7001–7110, secret path `home/skyhub`.
  Grep the diff and the commit message before committing.
- Tests: `go test ./...` runs against the library's fake hub (fixtures from the module cache;
  `SKYHUB_FIXTURES` overrides). Live:
  `TF_ACC=1 SKYHUB_LIVE=1 go test ./internal/provider -run TestAccLive -v -timeout 20m`,
  throwaway entries only (`tfTest`, `tfA`–`tfC`, MAC `02:00:00:00:00:01`); addresses and rule
  positions come from the hub (`liveHub`, `liveInboundCount`). Credentials as for the
  library: `SKYHUB_*` env or `~/skyhub`.
- Library bump: `go get github.com/fredericgermain/skyhub@vX.Y.Z && go mod tidy`, then the
  tests. `skyhub_lan` relies on the library's `ErrHubRestarting` after a LAN change.
- Releasing and the Registry: `docs/RELEASING.md` (tag → signed GitHub release → Registry
  webhook; the HCP Terraform namespace that owns the listing).
