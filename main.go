package main

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

func main() {

	ip := flag.String("ip", "", "ip address to delete in x.x.x.x form")
	note := flag.String("note", "", "note about ip in ibm cloud [tor1-xxx-xxx]")
	force := flag.Bool("force", false, "force yes to rename prompt. Use with caution!!!")

	flag.Parse()

	if len(os.Args) == 1 {
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
	ipservice := services.GetNetworkSubnetIpAddressService(sess)
	ipObject, err := ipservice.GetByIpAddress(ip)
	if ipObject.Id == nil {
		fmt.Println("error: unable to find IP in IBM subnets")
		if err != nil {
			fmt.Printf("error: %s\n", err.Error())
		}
		os.Exit(2)
	}
	currnetNote := ""
	if ipObject.Note != nil {
		currnetNote = *ipObject.Note
	}
	fmt.Printf("update IP note for %s from: '%s' to '%s'\n", *ipObject.IpAddress, currnetNote, *note)
	approve := false
	if *force {
		approve = true
	} else {
		approve = confirm()
	}
	if approve {
		ipObject.Note = note
		ipservice.Id(*ipObject.Id).EditObject(&ipObject)
		ipObject2, _ := ipservice.GetByIpAddress(ip)
		printJSON(ipObject2)
		return
	}
	fmt.Println("canceled")
	os.Exit(129)

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
	return false
}
