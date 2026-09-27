package provider

import "testing"

func TestLANHostFromCell(t *testing.T) {
	for cell, want := range map[string]string{
		"192.168.50.200 (7003:7003)": "192.168.50.200",
		"192.168.0.4 (500)":          "192.168.0.4",
		" 192.168.50.200 (1:65535)":  "192.168.50.200",
		"Any":                        "",
		"":                           "",
		"2001:db8::1 (22)":           "",
	} {
		if got := lanHostFromCell(cell); got != want {
			t.Errorf("lanHostFromCell(%q) = %q, want %q", cell, got, want)
		}
	}
}
