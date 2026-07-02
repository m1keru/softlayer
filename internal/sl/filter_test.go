package sl

import "testing"

func TestResolveExcludeNotes(t *testing.T) {
	defaults := []string{"A", "B"}
	opts := ListOpts{ExcludeNotes: []string{"C"}}
	got := ResolveExcludeNotes(opts, defaults)
	want := []string{"A", "B", "C"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("ResolveExcludeNotes = %v, want %v", got, want)
	}

	opts = ListOpts{NoDefaultExcludes: true, ExcludeNotes: []string{"X"}}
	got = ResolveExcludeNotes(opts, defaults)
	if len(got) != 1 || got[0] != "X" {
		t.Errorf("NoDefaultExcludes = %v, want [X]", got)
	}
}

func TestSubnetFilterEmpty(t *testing.T) {
	if subnetFilter(nil) != "" {
		t.Error("empty exclude list must produce no filter")
	}
}

func TestFilterSubnetsByCIDR(t *testing.T) {
	subnets := []SubnetInfo{
		{CIDR: "10.0.0.0/24"},
		{CIDR: "10.1.0.0/24"},
		{CIDR: "198.51.100.0/29"},
	}
	got := filterSubnetsByCIDR(subnets, []string{"10.0.0.0/24", "198.51.100.0"})
	if len(got) != 1 || got[0].CIDR != "10.1.0.0/24" {
		t.Errorf("filterSubnetsByCIDR = %+v", got)
	}
}

func TestSubnetExcludedByCIDR(t *testing.T) {
	cases := []struct {
		cidr string
		ex   string
		want bool
	}{
		{"10.0.0.0/24", "10.0.0.0/24", true},
		{"10.0.0.0/24", "10.0.0.0", true},
		{"10.0.0.0/24", "10.1.0.0", false},
	}
	for _, tc := range cases {
		if got := subnetExcludedByCIDR(tc.cidr, []string{tc.ex}); got != tc.want {
			t.Errorf("subnetExcludedByCIDR(%q, %q) = %v, want %v", tc.cidr, tc.ex, got, tc.want)
		}
	}
}
