package sl

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
	"golang.org/x/sync/errgroup"
)

// ErrPTRNotFound is returned when an IP has no matching PTR record.
var ErrPTRNotFound = errors.New("PTR record not found")

// subnetMask fetches subnets together with their IP addresses in a single
// API call instead of one GetIpAddresses call per subnet.
const subnetMask = "id,networkIdentifier,cidr,gateway," +
	"ipAddresses[id,ipAddress,note,isNetwork,isGateway,isBroadcast,isReserved]"

// Client is a thin wrapper around the SoftLayer API session.
type Client struct {
	sess *session.Session
	// concurrency limits parallel reverse-zone lookups to stay below the
	// SoftLayer API rate limit.
	concurrency int
}

// NewClientFromEnv builds a Client using SL_USER / SL_APIKEY credentials.
func NewClientFromEnv() (*Client, error) {
	username := os.Getenv("SL_USER")
	apikey := os.Getenv("SL_APIKEY")
	if username == "" || apikey == "" {
		return nil, errors.New("SL_USER and SL_APIKEY environment variables must be set")
	}
	sess := session.New(username, apikey)
	sess.Timeout = 90 * time.Second
	return &Client{sess: sess, concurrency: 8}, nil
}

// ListSubnets returns account subnets with their IPs, PTRs and notes.
func (c *Client) ListSubnets(opts ListOpts) ([]SubnetInfo, error) {
	opts.WithPTRs = true
	return c.listSubnets(opts, DefaultListExcludeNotes)
}

func (c *Client) listSubnets(opts ListOpts, defaultNotes []string) ([]SubnetInfo, error) {
	excludeNotes := ResolveExcludeNotes(opts, defaultNotes)
	accountService := services.GetAccountService(c.sess)
	query := accountService.Mask(subnetMask)
	if filter := subnetFilter(excludeNotes); filter != "" {
		query = query.Filter(filter)
	}
	subnets, err := query.GetSubnets()
	if err != nil {
		return nil, fmt.Errorf("get subnets: %w", err)
	}

	// Reverse zones exist only for public subnets; fetch them in parallel.
	ptrMaps := make([]map[string]string, len(subnets))
	if opts.WithPTRs {
		var group errgroup.Group
		group.SetLimit(c.concurrency)
		for i, subnet := range subnets {
			if subnet.Id == nil || subnet.NetworkIdentifier == nil || IsPrivateIP(*subnet.NetworkIdentifier) {
				continue
			}
			group.Go(func() error {
				domains, err := services.GetNetworkSubnetService(c.sess).Id(*subnet.Id).GetReverseDomainRecords()
				if err != nil {
					return fmt.Errorf("get reverse records for subnet %s: %w", *subnet.NetworkIdentifier, err)
				}
				ptrMaps[i] = buildPTRMap(domains)
				return nil
			})
		}
		if err := group.Wait(); err != nil {
			return nil, err
		}
	}

	var result []SubnetInfo
	for i, subnet := range subnets {
		info := buildSubnetInfo(subnet, ptrMaps[i])
		if (info.Private && opts.IncludePrivate) || (!info.Private && opts.IncludePublic) {
			result = append(result, info)
		}
	}
	return filterSubnetsByCIDR(result, opts.ExcludeCIDRs), nil
}

// StaleIPs returns public IP addresses that do not answer pings.
func (c *Client) StaleIPs(opts ListOpts) ([]IPInfo, error) {
	opts.IncludePublic, opts.IncludePrivate, opts.WithPTRs = true, false, false
	subnets, err := c.listSubnets(opts, DefaultStaleExcludeNotes)
	if err != nil {
		return nil, err
	}

	var candidates []IPInfo
	for _, subnet := range subnets {
		candidates = append(candidates, subnet.IPs...)
	}

	alive := make([]bool, len(candidates))
	var group errgroup.Group
	group.SetLimit(pingConcurrency)
	for i, ip := range candidates {
		group.Go(func() error {
			alive[i] = isAlive(ip.IP)
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}

	var stale []IPInfo
	for i, ip := range candidates {
		if !alive[i] {
			stale = append(stale, ip)
		}
	}
	return stale, nil
}

// getIPObject looks up the SoftLayer IP address object for a literal IP.
func (c *Client) getIPObject(ip string) (datatypes.Network_Subnet_IpAddress, error) {
	ipService := services.GetNetworkSubnetIpAddressService(c.sess)
	ipObject, err := ipService.GetByIpAddress(&ip)
	if err != nil {
		return datatypes.Network_Subnet_IpAddress{}, fmt.Errorf("look up IP %s: %w", ip, err)
	}
	if ipObject.Id == nil {
		return datatypes.Network_Subnet_IpAddress{}, fmt.Errorf("IP %s not found in account subnets", ip)
	}
	return ipObject, nil
}

// GetPTR returns the PTR record for the given IP, or ErrPTRNotFound.
func (c *Client) GetPTR(ip string) (datatypes.Dns_Domain_ResourceRecord, error) {
	ipObject, err := c.getIPObject(ip)
	if err != nil {
		return datatypes.Dns_Domain_ResourceRecord{}, err
	}
	domains, err := services.GetNetworkSubnetService(c.sess).Id(*ipObject.SubnetId).GetReverseDomainRecords()
	if err != nil {
		return datatypes.Dns_Domain_ResourceRecord{}, fmt.Errorf("get reverse records: %w", err)
	}
	host := lastOctet(ip)
	for _, domain := range domains {
		for _, record := range domain.ResourceRecords {
			if record.Host != nil && *record.Host == host {
				return record, nil
			}
		}
	}
	return datatypes.Dns_Domain_ResourceRecord{}, ErrPTRNotFound
}

// SetPTR creates (or replaces) the PTR record for an IP.
func (c *Client) SetPTR(ip, ptr string, ttl int) (datatypes.Dns_Domain_ResourceRecord, error) {
	dnsService := services.GetDnsDomainService(c.sess)
	record, err := dnsService.CreatePtrRecord(&ip, &ptr, &ttl)
	if err != nil {
		return datatypes.Dns_Domain_ResourceRecord{}, fmt.Errorf("create PTR for %s: %w", ip, err)
	}
	return record, nil
}

// DeletePTR removes the PTR record of an IP. Deleting a missing record is
// reported via ErrPTRNotFound so callers can treat it as a no-op.
func (c *Client) DeletePTR(ip string) (datatypes.Dns_Domain_ResourceRecord, error) {
	record, err := c.GetPTR(ip)
	if err != nil {
		return datatypes.Dns_Domain_ResourceRecord{}, err
	}
	if _, err := services.GetDnsDomainResourceRecordService(c.sess).Id(*record.Id).DeleteObject(); err != nil {
		return datatypes.Dns_Domain_ResourceRecord{}, fmt.Errorf("delete PTR for %s: %w", ip, err)
	}
	return record, nil
}

// GetNote returns the current note of an IP address.
func (c *Client) GetNote(ip string) (string, error) {
	ipObject, err := c.getIPObject(ip)
	if err != nil {
		return "", err
	}
	if ipObject.Note == nil {
		return "", nil
	}
	return *ipObject.Note, nil
}

// SetNote updates the note attached to an IP address.
func (c *Client) SetNote(ip, note string) error {
	ipObject, err := c.getIPObject(ip)
	if err != nil {
		return err
	}
	ipObject.Note = &note
	ok, err := services.GetNetworkSubnetIpAddressService(c.sess).Id(*ipObject.Id).EditObject(&ipObject)
	if err != nil {
		return fmt.Errorf("update note for %s: %w", ip, err)
	}
	if !ok {
		return fmt.Errorf("update note for %s: API returned false", ip)
	}
	return nil
}
