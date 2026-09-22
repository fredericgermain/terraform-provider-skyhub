package provider

import "regexp"

var serviceNameRE = regexp.MustCompile(`^[A-Za-z0-9_-]{1,12}$`)
