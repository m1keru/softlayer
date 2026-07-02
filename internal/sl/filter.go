package sl

import (
	"strings"
)

// DefaultListExcludeNotes are subnet notes skipped by list and lease unless
// overridden on the command line.
var DefaultListExcludeNotes = []string{
	"ONLY FOR METAL",
	"FULL",
	"NETWORK FOR INFOSEC",
}

// DefaultStaleExcludeNotes are subnet notes skipped by stale. FULL subnets are
// kept: dead IPs there are exactly what we want to find.
var DefaultStaleExcludeNotes = []string{
	"ONLY FOR METAL",
	"NETWORK FOR INFOSEC",
}

// ListOpts controls subnet listing and search commands.
type ListOpts struct {
	IncludePublic     bool
	IncludePrivate    bool
	ExcludeNotes      []string // merged with defaults unless NoDefaultExcludes
	ExcludeCIDRs      []string // applied after the API call
	NoDefaultExcludes bool
	WithPTRs          bool
}

// ResolveExcludeNotes returns the subnet-note exclusions to send to the API.
func ResolveExcludeNotes(opts ListOpts, defaults []string) []string {
	if opts.NoDefaultExcludes {
		return append([]string(nil), opts.ExcludeNotes...)
	}
	out := append([]string(nil), defaults...)
	out = append(out, opts.ExcludeNotes...)
	return out
}

// subnetFilter builds an account subnet filter excluding the given notes.
func subnetFilter(excludeNotes []string) string {
	if len(excludeNotes) == 0 {
		return ""
	}
	values := make([]string, len(excludeNotes))
	for i, note := range excludeNotes {
		values[i] = `"!~ ` + note + `"`
	}
	return `{"subnets":{"note":{"operation":"and","options":[{"name":"data","value":[` +
		strings.Join(values, ",") + `]}]}}}`
}

// filterSubnetsByCIDR drops subnets whose CIDR matches any exclude pattern.
// Patterns may be a full CIDR (10.0.0.0/24) or a network address (10.0.0.0).
func filterSubnetsByCIDR(subnets []SubnetInfo, excludeCIDRs []string) []SubnetInfo {
	if len(excludeCIDRs) == 0 {
		return subnets
	}
	var out []SubnetInfo
	for _, subnet := range subnets {
		if !subnetExcludedByCIDR(subnet.CIDR, excludeCIDRs) {
			out = append(out, subnet)
		}
	}
	return out
}

func subnetExcludedByCIDR(cidr string, excludeCIDRs []string) bool {
	network := strings.SplitN(cidr, "/", 2)[0]
	for _, ex := range excludeCIDRs {
		ex = strings.TrimSpace(ex)
		if ex == "" {
			continue
		}
		if cidr == ex {
			return true
		}
		if !strings.Contains(ex, "/") && network == ex {
			return true
		}
	}
	return false
}
