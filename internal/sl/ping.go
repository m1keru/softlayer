package sl

import (
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

// pingConcurrency limits simultaneous ICMP probes in StaleIPs.
const pingConcurrency = 32

// isAlive reports whether the IP answers an ICMP echo within a second.
func isAlive(ip string) bool {
	pinger, err := probing.NewPinger(ip)
	if err != nil {
		return false
	}
	pinger.Count = 2
	pinger.Timeout = time.Second
	// Unprivileged UDP mode: no root required on macOS; on Linux it needs
	// the net.ipv4.ping_group_range sysctl to cover the current group.
	pinger.SetPrivileged(false)
	if err := pinger.Run(); err != nil {
		return false
	}
	return pinger.Statistics().PacketsRecv > 0
}
