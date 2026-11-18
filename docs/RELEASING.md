# Releasing

1. Pin the library version the change needs (`go get github.com/fredericgermain/skyhub@vX.Y.Z`,
   `go mod tidy`), run `go test ./...`, and the live tests when a writer changed.
2. Commit on `main`, create an annotated tag `vX.Y.Z`, push `main` and the tag.
3. GitHub Actions `release` (`.github/workflows/release.yml`) runs goreleaser: binaries for
   every platform, `terraform-registry-manifest.json` attached as
   `terraform-provider-skyhub_X.Y.Z_manifest.json`, and `SHA256SUMS` signed with the GPG key
   imported from the repo secret `GPG_PRIVATE_KEY` (no passphrase; fingerprint
   `1B8A 911E 7B86 3F3C 562D 0D0F AA5E DE96 582B 2D47`). That secret is written by the owner's
   infrastructure-as-code from the key's secret store: rotate the key there and let it
   re-apply, never with `gh secret set`.
4. The Terraform Registry ingests the tag through the GitHub webhook installed at publish time.
   `https://registry.terraform.io/v1/providers/fredericgermain/skyhub/versions` lists the new
   version within minutes; consumers pick it up per their version constraints and re-lock.

## The Registry listing

Published 2026-11-14. registry.terraform.io itself only signs in; the listing is managed in
HCP Terraform: Registry → Public namespaces → `fredericgermain`. Its Settings page holds the
namespace's GPG keys (the public part of the signing key above; the Registry verifies each
release's `SHA256SUMS.sig` against it), and Publish → **Provider** lists the repositories
named `terraform-provider-*` the Registry's GitHub app can see.

Publish → Module on this repo creates a *module* `skyhub/provider` under the Modules tab and
leaves the provider unpublished ("provider not found" from the API): pick Provider.

What the Registry requires, all in place: a public repo named `terraform-provider-skyhub`,
releases from semver tags carrying the manifest, `SHA256SUMS` and its `.sig`, and `docs/`
(tfplugindocs output) for the documentation pages.
