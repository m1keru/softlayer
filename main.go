/*
List subnets. It retrieves all network subnets associated with an account.

Important manual pages:
http://sldn.softlayer.com/reference/services/SoftLayer_Account/getSubnets
http://sldn.softlayer.com/reference/datatypes/SoftLayer_Network_Subnet

License: http://sldn.softlayer.com/article/License
Author: SoftLayer Technologies, Inc. <sldn@softlayer.com>
*/
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
)

func main() {

	ip := flag.String("ip", "", "ip address to delete in x.x.x.x form")
	flag.Parse()
	ipToDelete := net.ParseIP(*ip)
	if ipToDelete.To4() == nil {
		fmt.Printf("error: ip '%s' is not valid ipv4 address \n", *ip)
		os.Exit(127)
	}
	//regex := regexp.MustCompile(`\.(\d+)$`)
	//searchPattern := regex.ReplaceAllString(*ip, ".*")
	//fmt.Println(searchPattern)
	// SoftLayer API username and key
	username := os.Getenv("SL_USER")
	apikey := os.Getenv("SL_APIKEY")
	//fmt.Printf("user: %s, key: %s", username, apikey)

	// Create SoftLayer API session
	// endpoint := "https://api.softlayer.com/rest/v3"
	sess := session.New(username, apikey)
	// Get SoftLayer_Account service
	/*
		Filter all subnets by subnetType, following are possible values:
		      PRIMARY_6, GLOBAL_IP, PRIMARY, STATIC_IP_ROUTED,
		      SECONDARY_ON_VLAN, ADDITIONAL_PRIMARY
		On this case we'll filter by 'PRIMARY' type.
	*/

	//service := services.GetAccountService(sess)
	//filter := filter.Path("subnets.networkIdentifier").Like(searchPattern).Build()

	// Call method getSubnets() in order to get all subnets in the Account.
	//mask := "id;networkIdentifier;cidr;subnetType;totalIpAddresses;usableIpAddressCount"
	ipservice := services.GetNetworkSubnetIpAddressService(sess)
	ipObject, _ := ipservice.GetByIpAddress(ip)
	note := "test"
	ipObject.Note = &note
	ipservice.Id(*ipObject.Id).EditObject(&ipObject)
	ipObject2, _ := ipservice.GetByIpAddress(ip)
	printJSON(ipObject2)

}

func printJSON(obj interface{}) {
	jsonFormat, jsonErr := json.Marshal(obj)
	if jsonErr != nil {
		fmt.Println(jsonErr)
		return
	}
	fmt.Println(string(jsonFormat))

}
