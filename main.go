package main

/* TODO:
* List free adddresses based on FREE record
 */

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"

	"github.com/softlayer/softlayer-go/filter"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
)

type ipAddress struct {
	id   string
	ip   net.IP
	ptr  string
	ttl  int
	note string
	sess *session.Session
}

func (ip ipAddress) isNetworkLocal() bool {
	network := "10.0.0.0/8"
	_, subnet, _ := net.ParseCIDR(network)
	if subnet.Contains(ip.ip) {
		return true
	}
	return false
}

func (ip ipAddress) findIPS(networkType string) {
	var filterNet string
	mask := "id,networkIdentifier,note,cidr,gateway"
	accountService := services.GetAccountService(ip.sess)
	subnetService := services.GetNetworkSubnetService(ip.sess)
	//ipservice := services.GetNetworkSubnetIpAddressService(ip.sess)
	//dnsservice := services.GetDnsDomainService(ip.sess)
	if networkType == "public" {
		//filterNet = filter.Path("subnets.networkIdentifier").Eq("158.85.102.*").Build()
		filterNet = `{"subnets":
			{
			 "networkIdentifier":{"operation":"!~ 10.*"},
			 "note":{"operation":"~ PARTIAL AVAILABLE"}
			 }
		}`
	} else {
		filterNet = filter.Path("subnets.networkidentifier").Like("10.*").Build()
	}
	filterNetNote := filter.Path("subnets.note").Like("PARTIAL AVAILABLE").Build()
	println(filterNetNote)
	subnets, err := accountService.Mask(mask).Filter(filterNet).GetSubnets()
	//printJSON(subnets)
	if err != nil {
		fmt.Println("error: unable to get Subnets from Softlayer", err)
		os.Exit(1)
	}
	re := regexp.MustCompile(".*\\.(.*)$")
	for _, subnet := range subnets {
		var resultingArray [][3]string // [[ip,ptr,note],[ip,ptr,note]]
		var tmpArray [3]string
		printJSON(subnet)
		fmt.Printf("========== %s/%d,%s ===========\n", *subnet.NetworkIdentifier, *subnet.Cidr, *subnet.Gateway)
		rawPTRPool, _ := subnetService.Id(*subnet.Id).GetReverseDomainRecords()
		strippedPTRPool := rawPTRPool[0].ResourceRecords
		rawIPPool, _ := subnetService.Id(*subnet.Id).GetIpAddresses()
		strippedIPPool := rawIPPool[0].Subnet.IpAddresses[2 : len(rawIPPool[0].Subnet.IpAddresses)-1]
		for _, ibmIP := range strippedIPPool {
			if *ibmIP.IpAddress == "" {
				continue
			}
			var ip, ptr, note string
			ip = *ibmIP.IpAddress
			if ibmIP.Note == nil {
				note = ""
			} else {
				note = *ibmIP.Note
			}
			match := re.FindStringSubmatch(*ibmIP.IpAddress)
			host := match[1]
			//fmt.Println(host)
			for _, ptrRecord := range strippedPTRPool {
				if ptrRecord.Host == nil {
					ptr = ""
				} else {
					ptr = *ptrRecord.Host
				}
				if ptr == string(host) {
					tmpArray = [3]string{ip, *ptrRecord.Data, note}
					break
				}
			}
			//printJSON(tmpArray)
			if tmpArray == [3]string{} {
				resultingArray = append(resultingArray, [3]string{ip, "", note})
			} else {
				resultingArray = append(resultingArray, tmpArray)
			}
			tmpArray = [3]string{}
		}
		for _, row := range resultingArray {
			printJSON(row)
		}
		//printJSON(resultingArray)
		//for _, network := range ipPool {
		//printJSON(net.Subnet.IpAddresses)
		//}
	}
}

/*

	for _, ipaddr := range network.Subnet.IpAddresses {
		if ipaddr.IpAddress != nil {
			ipObject, _ := ipservice.Id(*ipaddr.Id).GetObject()
			if *ipObject.IsBroadcast == false && *ipObject.IsNetwork == false && *ipObject.IsGateway == false {
				if ipObject.Note == nil || *ipObject.Note == "free" {
					for _, ptrObject := range ptrPool[0].ResourceRecords {
						printJSON(ptrObject)
						/*matcher := regexp.MustCompile(fmt.Sprintf(".*\\.%d$", ptrObject.Host))
						if matcher.MatchString(*ipObject.IpAddress) {
							fmt.Println(ipObject, ptrObject.Data)
						}
					}

					//for _, ptrObject := range ptrPool {
					//	printJSON(ptrObject)
					//}
					//printJSON(record)
				}
			}
		}
	}




*/

func (ip ipAddress) listFreeIPS() {
	/// Find public IP
	networkType := "public"
	ip.findIPS(networkType)

}

func (ip ipAddress) pushPTR() {
	dnsservice := services.GetDnsDomainService(ip.sess)
	record, err := dnsservice.CreatePtrRecord(&ip.id, &ip.ptr, &ip.ttl)
	if err != nil {
		fmt.Printf("error: unable to update ptr %s ", err)
		os.Exit(127)
	}
	printJSON(record)
}

func (ip ipAddress) updatePTR(force bool) {
	if ip.isNetworkLocal() {
		fmt.Println("this is internal ip. no ptr will be assigned")
		return
	}
	if ip.ptr == "" {
		ip.ptr = "FREE"
	}

	fmt.Printf("update PTR for IP %s to '%s'\n", ip.id, ip.ptr)

	if force == false {
		// remove without a prompt
		confirm()
	}
	ip.pushPTR()
}

func (ip ipAddress) updateIPNote(force bool) {

	ipservice := services.GetNetworkSubnetIpAddressService(ip.sess)
	ipObject, err := ipservice.GetByIpAddress(&ip.id)
	if ipObject.Id == nil {
		fmt.Println("error: unable to find IP in IBM subnets")
		if err != nil {
			fmt.Printf("error: %s\n", err.Error())
		}
		os.Exit(2)
	}
	if ip.note == "" {
		ip.note = "FREE"
	}
	currnetNote := "FREE"
	if ipObject.Note != nil {
		currnetNote = *ipObject.Note
	}
	fmt.Printf("update IP note for %s from: '%s' to '%s'\n", *ipObject.IpAddress, currnetNote, ip.note)
	if force == false {
		confirm()
	}
	ipObject.Note = &ip.note
	ipservice.Id(*ipObject.Id).EditObject(&ipObject)
	ipObject2, err := ipservice.GetByIpAddress(&ip.id)
	if err != nil {
		fmt.Println("updated, but we can not get new ip description form api")
	}
	printJSON(ipObject2)
}

func main() {

	list := flag.Bool("list", false, "list free ips")
	ip := flag.String("ip", "", "ip address to delete in x.x.x.x form. default ''")
	ptr := flag.String("ptr", "", "ip address ptr [hostname]. default 'free'")
	ttl := flag.Int("ttl", 3600, "ttl for ptr. default 3600")
	note := flag.String("note", "", "note about ip in ibm cloud [host.domain.com]. default ''")
	force := flag.Bool("force", false, "force yes to rename prompt. Use with caution!!!. default false")

	flag.Parse()

	if *list == false {
		if flag.NFlag() == 0 || *ip == "" {
			flag.PrintDefaults()
			os.Exit(1)
		}

		ipToDelete := net.ParseIP(*ip)
		if ipToDelete.To4() == nil {
			fmt.Printf("error: ip '%s' is not valid ipv4 address \n", *ip)
			os.Exit(127)
		}
	}
	username := os.Getenv("SL_USER")
	apikey := os.Getenv("SL_APIKEY")
	sess := session.New(username, apikey)
	address := ipAddress{
		id:   *ip,
		ptr:  *ptr,
		ttl:  *ttl,
		note: *note,
		sess: sess,
	}

	if *list {
		address.listFreeIPS()
		return
	}
	address.updatePTR(*force)
	address.updateIPNote(*force)
}

func printJSON(obj interface{}) {
	jsonFormat, jsonErr := json.Marshal(obj)
	if jsonErr != nil {
		fmt.Println(jsonErr)
		os.Exit(130)
	}
	fmt.Println("Object: ", string(jsonFormat))

}

func confirm() bool {
	var s string
	fmt.Printf("(y/N): ")
	//_, err := fmt.Scan(&s)
	reader := bufio.NewReader(os.Stdin)
	s, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("error: %s", err)
		os.Exit(128)
	}
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)

	if s == "y" || s == "yes" {
		return true
	}
	fmt.Println("canceled")
	os.Exit(128)
	return false
}
