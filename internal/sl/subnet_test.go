package sl

import (
	"testing"

	"github.com/softlayer/softlayer-go/datatypes"
)

func strPtr(s string) *string { return &s }
func intPtr(n int) *int       { return &n }
func boolPtr(b bool) *bool    { return &b }

func TestIsPrivateIP(t *testing.T) {
	cases := []struct {
		ip   string
		want bool
	}{
		{"10.0.0.1", true},
		{"10.114.123.123", true},
		{"172.16.0.1", true},
		{"172.31.255.254", true},
		{"192.168.1.1", true},
		{"210.10.1.1", false}, // used to be misdetected as private by the old regexp
		{"95.10.2.3", false},  // ditto
		{"8.8.8.8", false},
		{"172.32.0.1", false},
		{"not-an-ip", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := IsPrivateIP(tc.ip); got != tc.want {
			t.Errorf("IsPrivateIP(%q) = %v, want %v", tc.ip, got, tc.want)
		}
	}
}

func TestLastOctet(t *testing.T) {
	cases := []struct {
		ip   string
		want string
	}{
		{"192.0.2.15", "15"},
		{"10.0.0.255", "255"},
		{"noip", ""},
		{"1.2.3.", ""},
	}
	for _, tc := range cases {
		if got := lastOctet(tc.ip); got != tc.want {
			t.Errorf("lastOctet(%q) = %q, want %q", tc.ip, got, tc.want)
		}
	}
}

func TestBuildPTRMap(t *testing.T) {
	domains := []datatypes.Dns_Domain{
		{ResourceRecords: []datatypes.Dns_Domain_ResourceRecord{
			{Host: strPtr("10"), Data: strPtr("a.example.com")},
			{Host: strPtr("11"), Data: strPtr("b.example.com")},
			{Host: nil, Data: strPtr("orphan")},
			{Host: strPtr("12"), Data: nil},
		}},
		{ResourceRecords: []datatypes.Dns_Domain_ResourceRecord{
			{Host: strPtr("13"), Data: strPtr("c.example.com")},
		}},
	}
	ptrs := buildPTRMap(domains)
	if len(ptrs) != 3 {
		t.Fatalf("got %d records, want 3: %v", len(ptrs), ptrs)
	}
	if ptrs["10"] != "a.example.com" || ptrs["13"] != "c.example.com" {
		t.Errorf("unexpected map contents: %v", ptrs)
	}
}

func TestBuildSubnetInfo(t *testing.T) {
	subnet := datatypes.Network_Subnet{
		Id:                intPtr(42),
		NetworkIdentifier: strPtr("198.51.100.0"),
		Cidr:              intPtr(29),
		Gateway:           strPtr("198.51.100.1"),
		IpAddresses: []datatypes.Network_Subnet_IpAddress{
			{IpAddress: strPtr("198.51.100.0"), IsNetwork: boolPtr(true)},
			{IpAddress: strPtr("198.51.100.1"), IsGateway: boolPtr(true)},
			// Out of order on purpose: output must be sorted.
			{IpAddress: strPtr("198.51.100.4"), Note: strPtr("in use")},
			{IpAddress: strPtr("198.51.100.2")},
			{IpAddress: strPtr("198.51.100.3")},
			{IpAddress: strPtr("198.51.100.5"), IsReserved: boolPtr(true)},
			{IpAddress: strPtr("198.51.100.7"), IsBroadcast: boolPtr(true)},
			{IpAddress: strPtr("")},
			{IpAddress: nil},
		},
	}
	ptrs := map[string]string{"3": "host3.example.com"}

	info := buildSubnetInfo(subnet, ptrs)

	if info.ID != 42 || info.CIDR != "198.51.100.0/29" || info.Gateway != "198.51.100.1" {
		t.Errorf("unexpected subnet metadata: %+v", info)
	}
	if info.Private {
		t.Error("198.51.100.0/29 must not be private")
	}
	if len(info.IPs) != 3 {
		t.Fatalf("got %d IPs, want 3 (network/gateway/broadcast/reserved/empty excluded): %+v", len(info.IPs), info.IPs)
	}
	if info.IPs[0].IP != "198.51.100.2" || info.IPs[1].IP != "198.51.100.3" || info.IPs[2].IP != "198.51.100.4" {
		t.Errorf("IPs not sorted: %+v", info.IPs)
	}
	if !info.IPs[0].Free {
		t.Error("198.51.100.2 has no PTR and no note, must be free")
	}
	if info.IPs[1].Free || info.IPs[1].PTR != "host3.example.com" {
		t.Errorf("198.51.100.3 must carry PTR and not be free: %+v", info.IPs[1])
	}
	if info.IPs[2].Free || info.IPs[2].Note != "in use" {
		t.Errorf("198.51.100.4 must carry note and not be free: %+v", info.IPs[2])
	}
}

func TestBuildSubnetInfoPrivate(t *testing.T) {
	subnet := datatypes.Network_Subnet{
		Id:                intPtr(7),
		NetworkIdentifier: strPtr("10.20.30.0"),
		Cidr:              intPtr(26),
		IpAddresses: []datatypes.Network_Subnet_IpAddress{
			{IpAddress: strPtr("10.20.30.5")},
		},
	}
	info := buildSubnetInfo(subnet, nil)
	if !info.Private {
		t.Error("10.20.30.0/26 must be private")
	}
	if len(info.IPs) != 1 || !info.IPs[0].Free || info.IPs[0].PTR != "" {
		t.Errorf("unexpected IPs: %+v", info.IPs)
	}
}

func TestFirstFree(t *testing.T) {
	subnets := []SubnetInfo{
		{CIDR: "198.51.100.0/29", IPs: []IPInfo{
			{IP: "198.51.100.2", Note: "taken", Free: false},
			{IP: "198.51.100.3", PTR: "x.example.com", Free: false},
		}},
		{CIDR: "203.0.113.0/29", MaskBits: 29, Gateway: "203.0.113.1", IPs: []IPInfo{
			{IP: "203.0.113.2", Free: true},
		}},
	}
	subnet, ip, ok := FirstFree(subnets)
	if !ok || ip.IP != "203.0.113.2" || subnet.Gateway != "203.0.113.1" || subnet.MaskBits != 29 {
		t.Errorf("FirstFree = %+v, %+v, %v", subnet, ip, ok)
	}

	if _, _, ok := FirstFree(subnets[:1]); ok {
		t.Error("FirstFree must report ok=false when no IP is free")
	}
	if _, _, ok := FirstFree(nil); ok {
		t.Error("FirstFree(nil) must report ok=false")
	}
}

func TestSubnetFilter(t *testing.T) {
	got := subnetFilter([]string{"ONLY FOR METAL", "FULL"})
	want := `{"subnets":{"note":{"operation":"and","options":[{"name":"data","value":["!~ ONLY FOR METAL","!~ FULL"]}]}}}`
	if got != want {
		t.Errorf("subnetFilter:\n got  %s\n want %s", got, want)
	}
}

func TestBuildSubnetInfoEmpty(t *testing.T) {
	// A subnet with no IP addresses must not panic (the old code sliced
	// IpAddresses[2:len-1] and crashed on short slices).
	info := buildSubnetInfo(datatypes.Network_Subnet{}, nil)
	if len(info.IPs) != 0 {
		t.Errorf("expected no IPs, got %+v", info.IPs)
	}
}
