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
	"strings"

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

	ip := flag.String("ip", "", "ip address to delete in x.x.x.x form. default ''")
	ptr := flag.String("ptr", "", "ip address ptr [hostname]. default 'free'")
	ttl := flag.Int("ttl", 3600, "ttl for ptr. default 3600")
	note := flag.String("note", "", "note about ip in ibm cloud [host.domain.com]. default ''")
	force := flag.Bool("force", false, "force yes to rename prompt. Use with caution!!!. default false")

	flag.Parse()

	if flag.NFlag() == 0 || *ip == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	ipToDelete := net.ParseIP(*ip)
	if ipToDelete.To4() == nil {
		fmt.Printf("error: ip '%s' is not valid ipv4 address \n", *ip)
		os.Exit(127)
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

	address.updatePTR(*force)
	address.updateIPNote(*force)
}

func printJSON(obj interface{}) {
	jsonFormat, jsonErr := json.Marshal(obj)
	if jsonErr != nil {
		fmt.Println(jsonErr)
		os.Exit(130)
	}
	fmt.Println("resulting ipObject: ", string(jsonFormat))

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
