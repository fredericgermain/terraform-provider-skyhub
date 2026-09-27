package provider

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/fredericgermain/skyhub/pkg/skyhubtest"
)

// The fake hub serves the library's captured hub pages. They are found, in
// order: SKYHUB_FIXTURES; a sibling checkout of the library (../skyhub, for
// working on both at once); the library module at the version go.mod pins,
// which `go test` has already downloaded -- that last one is what CI uses.
const fixtureSubdir = "pkg/skyhub/testdata/7.04.0208.R"

var fixtureDir = sync.OnceValue(func() string {
	if d := os.Getenv("SKYHUB_FIXTURES"); d != "" {
		return d
	}
	if d := filepath.Join("../../../skyhub", fixtureSubdir); isDir(d) {
		return d
	}
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/fredericgermain/skyhub").Output()
	if err != nil {
		return ""
	}
	return filepath.Join(strings.TrimSpace(string(out)), fixtureSubdir)
})

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

var testFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"skyhub": providerserver.NewProtocol6WithError(New("test")()),
}

// fakeHub points the provider at an in-process fake hub for this test.
func fakeHub(t *testing.T) *skyhubtest.FakeHub {
	t.Helper()
	h := skyhubtest.NewFakeHub(t, fixtureDir(), "admin", "secret12")
	// Every terraform command gets a fresh provider (and digest nc counter);
	// rotating the nonce per request keeps the fake's nc check happy.
	h.RotateNonceEvery = 1
	t.Setenv("SKYHUB_URL", h.URL())
	t.Setenv("SKYHUB_USER", "admin")
	t.Setenv("SKYHUB_PASSWORD", "secret12")
	t.Setenv("SKYHUB_CREDENTIALS_FILE", "/nonexistent")
	t.Setenv("TF_ACC", "1")
	return h
}

// liveHub skips unless TF_ACC and SKYHUB_LIVE are set; credentials come
// from the environment or ~/skyhub as for the CLI.
func liveHub(t *testing.T) {
	t.Helper()
	if os.Getenv("TF_ACC") == "" || os.Getenv("SKYHUB_LIVE") == "" {
		t.Skip("TF_ACC and SKYHUB_LIVE must be set")
	}
}
