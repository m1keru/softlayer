// Package sl wraps the SoftLayer API for managing IP notes and PTR records.
package sl

import (
	"bytes"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/softlayer/softlayer-go/datatypes"
)

// IPInfo describes a single usable IP address inside a subnet.
type IPInfo struct {
	IP   string `json:"ip"`
	PTR  string `json:"ptr,omitempty"`
	Note string `json:"note,omitempty"`
	Free bool   `json:"free"`
}

// SubnetInfo describes a subnet together with its usable IP addresses.
type SubnetInfo struct {
	ID       int      `json:"id"`
	CIDR     string   `json:"cidr"`
	MaskBits int      `json:"mask_bits"`
	Gateway  string   `json:"gateway,omitempty"`
	Private  bool     `json:"private"`
	IPs      []IPInfo `json:"ips"`
}

// FirstFree returns the first free IP across the given subnets along with
// its subnet, or ok=false when everything is taken.
func FirstFree(subnets []SubnetInfo) (SubnetInfo, IPInfo, bool) {
	for _, subnet := range subnets {
		for _, ip := range subnet.IPs {
			if ip.Free {
				return subnet, ip, true
			}
		}
	}
	return SubnetInfo{}, IPInfo{}, false
}

// IsPrivateIP reports whether the address belongs to a private range
// (RFC 1918: 10/8, 172.16/12, 192.168/16).
func IsPrivateIP(ip string) bool {
	parsed := net.ParseIP(ip)
	return parsed != nil && parsed.IsPrivate()
}

// lastOctet returns the last dot-separated part of an IPv4 address.
// SoftLayer reverse zones key PTR records by this value.
func lastOctet(ip string) string {
	i := strings.LastIndexByte(ip, '.')
	if i < 0 || i+1 >= len(ip) {
		return ""
	}
	return ip[i+1:]
}

// buildPTRMap flattens reverse DNS domains into a host -> PTR-target map.
func buildPTRMap(domains []datatypes.Dns_Domain) map[string]string {
	ptrs := make(map[string]string)
	for _, domain := range domains {
		for _, record := range domain.ResourceRecords {
			if record.Host == nil || record.Data == nil {
				continue
			}
			ptrs[*record.Host] = *record.Data
		}
	}
	return ptrs
}

// buildSubnetInfo converts an API subnet into SubnetInfo, skipping
// network/gateway/broadcast/reserved addresses and resolving PTRs via ptrs
// (nil for private subnets which have no public reverse zone).
func buildSubnetInfo(subnet datatypes.Network_Subnet, ptrs map[string]string) SubnetInfo {
	info := SubnetInfo{Private: subnet.NetworkIdentifier != nil && IsPrivateIP(*subnet.NetworkIdentifier)}
	if subnet.Id != nil {
		info.ID = *subnet.Id
	}
	if subnet.Cidr != nil {
		info.MaskBits = *subnet.Cidr
	}
	if subnet.NetworkIdentifier != nil && subnet.Cidr != nil {
		info.CIDR = *subnet.NetworkIdentifier + "/" + strconv.Itoa(*subnet.Cidr)
	}
	if subnet.Gateway != nil {
		info.Gateway = *subnet.Gateway
	}
	for _, addr := range subnet.IpAddresses {
		if addr.IpAddress == nil || *addr.IpAddress == "" {
			continue
		}
		if isTrue(addr.IsNetwork) || isTrue(addr.IsGateway) || isTrue(addr.IsBroadcast) || isTrue(addr.IsReserved) {
			continue
		}
		ip := IPInfo{IP: *addr.IpAddress}
		if addr.Note != nil {
			ip.Note = *addr.Note
		}
		if ptrs != nil {
			ip.PTR = ptrs[lastOctet(ip.IP)]
		}
		ip.Free = ip.Note == "" && ip.PTR == ""
		info.IPs = append(info.IPs, ip)
	}
	sortIPs(info.IPs)
	return info
}

func sortIPs(ips []IPInfo) {
	sort.Slice(ips, func(i, j int) bool {
		a, b := net.ParseIP(ips[i].IP), net.ParseIP(ips[j].IP)
		if a == nil || b == nil {
			return ips[i].IP < ips[j].IP
		}
		return bytes.Compare(a.To16(), b.To16()) < 0
	})
}

func isTrue(b *bool) bool {
	return b != nil && *b
}
