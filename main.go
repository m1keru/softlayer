package main

/* TODO:
*  Rework -list to less ugly style
 */

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"

	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
)

type softlayer struct {
	id          string
	ip          net.IP
	ptr         string
	ttl         int
	note        string
	sess        *session.Session
	listPublic  bool
	listPrivate bool
}

func (cli softlayer) isNetworkLocal() bool {
	network := "10.0.0.0/8"
	_, subnet, _ := net.ParseCIDR(network)
	if subnet.Contains(cli.ip) {
		return true
	}
	return false
}

func (cli softlayer) getPTR() (datatypes.Dns_Domain_ResourceRecord, error) {
	re := regexp.MustCompile(".*\\.(.*)$")
	ipservice := services.GetNetworkSubnetIpAddressService(cli.sess)
	ipObject, err := ipservice.GetByIpAddress(&cli.id)
	if ipObject.Id == nil {
		fmt.Println("error: unable to find cli in IBM subnets")
		if err != nil {
			fmt.Printf("error: %s\n", err.Error())
		}
		os.Exit(2)
	}
	subnetService := services.GetNetworkSubnetService(cli.sess)
	rawPTRPool, _ := subnetService.Id(*ipObject.SubnetId).GetReverseDomainRecords()
	strippedPTRPool := rawPTRPool[0].ResourceRecords
	match := re.FindStringSubmatch(cli.id)
	host := match[1]
	var ptr string
	for _, ptrRecord := range strippedPTRPool {
		if ptrRecord.Host == nil {
			ptr = ""
		} else {
			ptr = *ptrRecord.Host
		}
		if ptr == string(host) {
			return ptrRecord, nil
		}
	}
	return datatypes.Dns_Domain_ResourceRecord{}, errors.New("PTR not found")
}

func (cli softlayer) findIPS(networkType string) {
	var filterNet string
	mask := "id,networkIdentifier,note,cidr,gateway"
	accountService := services.GetAccountService(cli.sess)
	subnetService := services.GetNetworkSubnetService(cli.sess)
	re := regexp.MustCompile(".*\\.(.*)$")
	if networkType == "public" {
		filterNet = `{"subnets":
			{
			 "networkIdentifier":{"operation":"!~ ^10\\..*"},
			 "note":{"operation":"!~ ONLY FOR METAL"}
			 }
		}`
		subnets, err := accountService.Mask(mask).Filter(filterNet).GetSubnets()
		if err != nil {
			fmt.Println("error: unable to get Subnets from Softlayer", err)
			os.Exit(1)
		}

		fmt.Println("==============public networks============")
		for _, subnet := range subnets {
			var resultingArray [][3]string // [[ip,ptr,note],[ip,ptr,note]]
			var tmpArray [3]string
			fmt.Printf("----- %s/%d,%s -----\n", *subnet.NetworkIdentifier, *subnet.Cidr, *subnet.Gateway)
			rawPTRPool, _ := subnetService.Id(*subnet.Id).GetReverseDomainRecords()
			strippedPTRPool := rawPTRPool[0].ResourceRecords
			rawIPPool, _ := subnetService.Id(*subnet.Id).GetIpAddresses()
			strippedIPPool := rawIPPool[0].Subnet.IpAddresses[2 : len(rawIPPool[0].Subnet.IpAddresses)-1]
			for _, ibmcli := range strippedIPPool {
				if *ibmcli.IpAddress == "" {
					continue
				}
				var ip, ptr, note string
				ip = *ibmcli.IpAddress
				if ibmcli.Note == nil {
					note = ""
				} else {
					note = *ibmcli.Note
				}
				match := re.FindStringSubmatch(*ibmcli.IpAddress)
				host := match[1]
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
				if tmpArray == [3]string{} {
					resultingArray = append(resultingArray, [3]string{ip, "", note})
				} else {
					resultingArray = append(resultingArray, tmpArray)
				}
				tmpArray = [3]string{}
			}
			for _, row := range resultingArray {
				if row[1] == "" && row[2] == "" {
					fmt.Println(row[0])
				}
			}

		}

	} else {
		filterNet = `{"subnets":
			{
			 "networkIdentifier":{"operation":"~ ^10\\..*"},
			 "note":{"operation":"!~ ONLY FOR METAL"}
			 }
		}`
		subnets, err := accountService.Mask(mask).Filter(filterNet).GetSubnets()
		if err != nil {
			fmt.Println("error: unable to get Subnets from Softlayer", err)
			os.Exit(1)
		}
		fmt.Println("==============private networks============")
		for _, subnet := range subnets {
			var resultingArray [][3]string // [[ip,ptr,note],[ip,ptr,note]]
			var tmpArray [3]string
			fmt.Printf("---------- %s/%d,%s ----------\n", *subnet.NetworkIdentifier, *subnet.Cidr, *subnet.Gateway)
			rawIPPool, _ := subnetService.Id(*subnet.Id).GetIpAddresses()
			strippedIPPool := rawIPPool[0].Subnet.IpAddresses[2 : len(rawIPPool[0].Subnet.IpAddresses)-1]
			for _, ibmcli := range strippedIPPool {
				if *ibmcli.IpAddress == "" {
					continue
				}
				var ip, note string
				ip = *ibmcli.IpAddress
				if ibmcli.Note == nil {
					note = ""
				} else {
					note = *ibmcli.Note
				}
				tmpArray = [3]string{ip, note}
				resultingArray = append(resultingArray, tmpArray)
			}
			for _, row := range resultingArray {
				if row[1] == "" {
					fmt.Println(row[0])
				}
			}
		}

	}
}

func (cli softlayer) listFreeIPS() {
	/// Find public IP
	if cli.listPublic {
		cli.findIPS("public")
	}
	if cli.listPrivate {
		cli.findIPS("private")
	}
}

func (cli softlayer) deletePTR() {
	dnsservice := services.GetDnsDomainResourceRecordService(cli.sess)
	ptrObject, err := cli.getPTR()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Print("DELETE PTR: ")
	printJSON(ptrObject)
	_, err = dnsservice.Id(*ptrObject.Id).DeleteObject()
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
	}
}

func (cli softlayer) pushPTR() {
	dnsservice := services.GetDnsDomainService(cli.sess)
	record, err := dnsservice.CreatePtrRecord(&cli.id, &cli.ptr, &cli.ttl)
	if err != nil {
		fmt.Printf("error: unable to update ptr %s ", err)
		os.Exit(127)
	}
	printJSON(record)
}

func (cli softlayer) updatePTR(force bool) {
	if cli.isNetworkLocal() {
		fmt.Println("this is internal cli. no ptr will be assigned")
		return
	}

	fmt.Printf("update PTR for cli %s to '%s'\n", cli.id, cli.ptr)

	if force == false {
		// remove without a prompt
		confirm()
	}
	if cli.ptr == "" {
		cli.deletePTR()
	} else {
		cli.pushPTR()
	}

}

func (cli softlayer) updateIPNote(force bool) {

	ipservice := services.GetNetworkSubnetIpAddressService(cli.sess)
	ipObject, err := ipservice.GetByIpAddress(&cli.id)
	if ipObject.Id == nil {
		fmt.Println("error: unable to find cli in IBM subnets")
		if err != nil {
			fmt.Printf("error: %s\n", err.Error())
		}
		os.Exit(2)
	}
	//if cli.note == "" {
	//	cli.note = "free"
	//}
	currnetNote := ""
	if ipObject.Note != nil {
		currnetNote = *ipObject.Note
	}
	fmt.Printf("update cli note for %s from: '%s' to '%s'\n", *ipObject.IpAddress, currnetNote, cli.note)
	if force == false {
		confirm()
	}
	ipObject.Note = &cli.note
	ipservice.Id(*ipObject.Id).EditObject(&ipObject)
	ipObject2, err := ipservice.GetByIpAddress(&cli.id)
	if err != nil {
		fmt.Println("updated, but we can not get new cli description form api")
	}
	printJSON(ipObject2)
}

func main() {

	force := flag.Bool("force", false, "force yes to rename prompt. Use with caution!!!. default false")
	note := flag.String("note", "", "note about cli in ibm cloud [host.domain.com]. default ''")
	ttl := flag.Int("ttl", 3600, "ttl for ptr. default 3600")
	ptr := flag.String("ptr", "", "cli address ptr [hostname]. default ''")
	cli := flag.String("ip", "", "ip address to delete in x.x.x.x form. default ''")
	list := flag.Bool("list", false, "list free public and private ips")
	listPublic := flag.Bool("public", false, "list only free public ips [use only with -list]")
	listPrivate := flag.Bool("private", false, "list only free private ips [use only with -list]")

	flag.Parse()

	if *list != false && *listPrivate == false && *listPublic == false {
		*listPublic = true
		*listPrivate = true
	}

	if *list == false {
		if flag.NFlag() == 0 || *cli == "" {
			flag.PrintDefaults()
			os.Exit(1)
		}

		ipToDelete := net.ParseIP(*cli)
		if ipToDelete.To4() == nil {
			fmt.Printf("error: cli '%s' is not valid ipv4 address \n", *cli)
			os.Exit(127)
		}
	}
	username := os.Getenv("SL_USER")
	apikey := os.Getenv("SL_APIKEY")
	sess := session.New(username, apikey)
	address := softlayer{
		id:          *cli,
		ptr:         *ptr,
		ttl:         *ttl,
		note:        *note,
		sess:        sess,
		listPublic:  *listPublic,
		listPrivate: *listPrivate,
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
