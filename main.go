// Command softlayer manages IP address notes and PTR records in the
// IBM SoftLayer (Classic Infrastructure) API.
package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/m1keru/softlayer/internal/sl"
)

//go:embed all:completions
var completionsFS embed.FS

// version is injected at build time via -ldflags "-X main.version=...".
var version = "dev"

const usage = `Usage: softlayer <command> [options]

Commands:
  list        list subnets and their IPs (free ones by default)
  set         set PTR and/or note for an IP
  clear       remove PTR and note from an IP
  lease       reserve the first free private+public IP pair and print
              puppet/netplan network config
  stale       list public IPs that do not answer pings
  completion  print shell completion script (bash or zsh)
  version     print the version number

Credentials are read from the SL_USER and SL_APIKEY environment variables.

Run 'softlayer <command> -h' for command options.

Legacy flat flags (-list, -lease, -liststale, -ip …) are still accepted.`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	if isLegacyInvocation(args) {
		if helpRequested(args) {
			return showLegacyHelp(stripHelpArgs(args))
		}
		args = translateLegacyArgs(args)
	}
	switch args[0] {
	case "list":
		return cmdList(args[1:])
	case "set":
		return cmdSet(args[1:])
	case "clear":
		return cmdClear(args[1:])
	case "lease":
		return cmdLease(args[1:])
	case "stale":
		return cmdStale(args[1:])
	case "completion":
		return cmdCompletion(args[1:])
	case "version", "-version", "--version":
		fmt.Println("Version:", version)
		return nil
	case "-h", "--help", "help":
		fmt.Println(usage)
		return nil
	default:
		fmt.Fprintln(os.Stderr, usage)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func cmdList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	public := fs.Bool("public", false, "list only public subnets")
	private := fs.Bool("private", false, "list only private subnets")
	all := fs.Bool("all", false, "show all IPs, not only free ones")
	one := fs.Bool("one", false, "print only the first free IP as 'ip/mask gateway'")
	asJSON := fs.Bool("json", false, "output JSON instead of a table")
	var excludeNotes, excludeCIDRs stringList
	noDefault := registerExcludeFlags(fs, &excludeNotes, &excludeCIDRs)
	if helpRequested(args) {
		showCommandHelp("list", "List subnets and their free IPs.", fs)
		return nil
	}
	if err := fs.Parse(stripHelpArgs(args)); err != nil {
		return err
	}
	// No scope flags means both.
	if !*public && !*private {
		*public, *private = true, true
	}

	client, err := sl.NewClientFromEnv()
	if err != nil {
		return err
	}
	subnets, err := client.ListSubnets(listOptsFromFlags(public, private, noDefault, &excludeNotes, &excludeCIDRs))
	if err != nil {
		return err
	}

	if *one {
		subnet, ip, ok := sl.FirstFree(subnets)
		if !ok {
			return errors.New("no free IPs found")
		}
		if *asJSON {
			return printJSON(map[string]any{"ip": ip.IP, "mask_bits": subnet.MaskBits, "gateway": subnet.Gateway})
		}
		fmt.Printf("%s/%d %s\n", ip.IP, subnet.MaskBits, subnet.Gateway)
		return nil
	}

	if !*all {
		for i := range subnets {
			var free []sl.IPInfo
			for _, ip := range subnets[i].IPs {
				if ip.Free {
					free = append(free, ip)
				}
			}
			subnets[i].IPs = free
		}
	}

	if *asJSON {
		return printJSON(subnets)
	}
	return printSubnetsTable(subnets, *all)
}

func cmdSet(args []string) error {
	fs := flag.NewFlagSet("set", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	ip := fs.String("ip", "", "IPv4 address (required)")
	ptr := fs.String("ptr", "", "PTR hostname to assign")
	note := fs.String("note", "", "note to assign to the IP")
	ttl := fs.Int("ttl", 3600, "TTL for the PTR record")
	force := fs.Bool("force", false, "do not ask for confirmation")
	if helpRequested(args) {
		showCommandHelp("set", "Set PTR and/or note for an IP.", fs)
		return nil
	}
	if err := fs.Parse(stripHelpArgs(args)); err != nil {
		return err
	}
	if err := validateIP(*ip); err != nil {
		fs.PrintDefaults()
		return err
	}
	ptrSet, noteSet := flagPassed(fs, "ptr"), flagPassed(fs, "note")
	if !ptrSet && !noteSet {
		fs.PrintDefaults()
		return errors.New("nothing to do: pass -ptr and/or -note")
	}

	client, err := sl.NewClientFromEnv()
	if err != nil {
		return err
	}

	if ptrSet {
		if sl.IsPrivateIP(*ip) {
			fmt.Printf("%s is a private IP, skipping PTR\n", *ip)
		} else if err := applyPTR(client, *ip, *ptr, *ttl, *force); err != nil {
			return err
		}
	}
	if noteSet {
		if err := applyNote(client, *ip, *note, *force); err != nil {
			return err
		}
	}
	return nil
}

func cmdClear(args []string) error {
	fs := flag.NewFlagSet("clear", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	ip := fs.String("ip", "", "IPv4 address (required)")
	force := fs.Bool("force", false, "do not ask for confirmation")
	if helpRequested(args) {
		showCommandHelp("clear", "Remove PTR and note from an IP.", fs)
		return nil
	}
	if err := fs.Parse(stripHelpArgs(args)); err != nil {
		return err
	}
	if err := validateIP(*ip); err != nil {
		fs.PrintDefaults()
		return err
	}

	client, err := sl.NewClientFromEnv()
	if err != nil {
		return err
	}
	if !sl.IsPrivateIP(*ip) {
		if err := applyPTR(client, *ip, "", 3600, *force); err != nil {
			return err
		}
	}
	return applyNote(client, *ip, "", *force)
}

// applyPTR sets the PTR when the value is non-empty, deletes it otherwise.
func applyPTR(client *sl.Client, ip, ptr string, ttl int, force bool) error {
	if ptr == "" {
		fmt.Printf("delete PTR for %s\n", ip)
		if !force && !confirm() {
			return nil
		}
		record, err := client.DeletePTR(ip)
		if errors.Is(err, sl.ErrPTRNotFound) {
			fmt.Println("no PTR record, nothing to delete")
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Print("deleted: ")
		return printJSON(record)
	}

	fmt.Printf("set PTR for %s to %q\n", ip, ptr)
	if !force && !confirm() {
		return nil
	}
	record, err := client.SetPTR(ip, ptr, ttl)
	if err != nil {
		return err
	}
	fmt.Print("created: ")
	return printJSON(record)
}

func applyNote(client *sl.Client, ip, note string, force bool) error {
	current, err := client.GetNote(ip)
	if err != nil {
		return err
	}
	if current == note {
		fmt.Printf("note for %s already %q, skipping\n", ip, note)
		return nil
	}
	fmt.Printf("update note for %s: %q -> %q\n", ip, current, note)
	if !force && !confirm() {
		return nil
	}
	if err := client.SetNote(ip, note); err != nil {
		return err
	}
	fmt.Println("note updated")
	return nil
}

// leaseConfigTemplate renders puppet and netplan network configs for a
// leased public (eth0) + private (eth1) IP pair.
const leaseConfigTemplate = `
puppet:
eth0:
  addresses: [%[1]s/%[2]d]
  nameservers:
    - 1.1.1.1
    - 8.8.8.8
  search: ['%[7]s']
  routes:
    - to: default
      via: %[3]s
eth1:
  addresses: [%[4]s/%[5]d]
  routes:
    - to: 10.0.0.0/8
      via: %[6]s

netplan:
---
network:
  version: 2
  renderer: networkd
  ethernets:
    eth0:
      addresses:
        - %[1]s/%[2]d
      nameservers:
        addresses: [1.1.1.1, 8.8.8.8]
        search: [%[7]s]
      routes:
        - to: default
          via: %[3]s
    eth1:
      addresses:
        - %[4]s/%[5]d
      routes:
        - to: 10.0.0.0/8
          via: %[6]s

`

func cmdLease(args []string) error {
	fs := flag.NewFlagSet("lease", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	ptr := fs.String("ptr", "", "PTR hostname for the leased IPs (required)")
	note := fs.String("note", "", "note for the leased IPs (required)")
	ttl := fs.Int("ttl", 3600, "TTL for the PTR record")
	search := fs.String("search", "example.com", "DNS search domain for the generated config")
	force := fs.Bool("force", false, "do not ask for confirmation")
	var excludeNotes, excludeCIDRs stringList
	noDefault := registerExcludeFlags(fs, &excludeNotes, &excludeCIDRs)
	if helpRequested(args) {
		showCommandHelp("lease", "Reserve the first free private+public IP pair and print puppet/netplan config.", fs)
		return nil
	}
	if err := fs.Parse(stripHelpArgs(args)); err != nil {
		return err
	}
	if *ptr == "" || *note == "" {
		fs.PrintDefaults()
		return errors.New("-ptr and -note are required for lease")
	}

	client, err := sl.NewClientFromEnv()
	if err != nil {
		return err
	}
	includePublic, includePrivate := true, true
	subnets, err := client.ListSubnets(listOptsFromFlags(&includePublic, &includePrivate, noDefault, &excludeNotes, &excludeCIDRs))
	if err != nil {
		return err
	}

	var private, public []sl.SubnetInfo
	for _, subnet := range subnets {
		if subnet.Private {
			private = append(private, subnet)
		} else {
			public = append(public, subnet)
		}
	}
	privSubnet, privIP, ok := sl.FirstFree(private)
	if !ok {
		return errors.New("no free private IPs found")
	}
	pubSubnet, pubIP, ok := sl.FirstFree(public)
	if !ok {
		return errors.New("no free public IPs found")
	}

	fmt.Printf("lease private %s/%d %s\n", privIP.IP, privSubnet.MaskBits, privSubnet.Gateway)
	fmt.Printf("lease public  %s/%d %s\n", pubIP.IP, pubSubnet.MaskBits, pubSubnet.Gateway)

	// Private IPs get a note only, the public one also gets a PTR.
	if err := applyNote(client, privIP.IP, *note, *force); err != nil {
		return err
	}
	if err := applyPTR(client, pubIP.IP, *ptr, *ttl, *force); err != nil {
		return err
	}
	if err := applyNote(client, pubIP.IP, *note, *force); err != nil {
		return err
	}

	fmt.Printf(leaseConfigTemplate,
		pubIP.IP, pubSubnet.MaskBits, pubSubnet.Gateway,
		privIP.IP, privSubnet.MaskBits, privSubnet.Gateway,
		*search)
	return nil
}

func cmdStale(args []string) error {
	fs := flag.NewFlagSet("stale", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	asJSON := fs.Bool("json", false, "output JSON instead of a table")
	var excludeNotes, excludeCIDRs stringList
	noDefault := registerExcludeFlags(fs, &excludeNotes, &excludeCIDRs)
	if helpRequested(args) {
		showCommandHelp("stale", "List public IPs that do not answer pings.", fs)
		return nil
	}
	if err := fs.Parse(stripHelpArgs(args)); err != nil {
		return err
	}

	client, err := sl.NewClientFromEnv()
	if err != nil {
		return err
	}
	includePublic := true
	includePrivate := false
	stale, err := client.StaleIPs(listOptsFromFlags(&includePublic, &includePrivate, noDefault, &excludeNotes, &excludeCIDRs))
	if err != nil {
		return err
	}

	if *asJSON {
		return printJSON(stale)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	for _, ip := range stale {
		fmt.Fprintf(w, "%s\t%s\n", ip.IP, ip.Note)
	}
	return w.Flush()
}

func cmdCompletion(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: softlayer completion <bash|zsh>")
	}
	var path string
	switch args[0] {
	case "bash":
		path = "completions/softlayer.bash"
	case "zsh":
		path = "completions/_softlayer"
	default:
		return fmt.Errorf("unsupported shell %q (bash and zsh are supported)", args[0])
	}
	script, err := completionsFS.ReadFile(path)
	if err != nil {
		return err
	}
	fmt.Print(string(script))
	return nil
}

func validateIP(ip string) error {
	if ip == "" {
		return errors.New("-ip is required")
	}
	if parsed := net.ParseIP(ip); parsed == nil || parsed.To4() == nil {
		return fmt.Errorf("%q is not a valid IPv4 address", ip)
	}
	return nil
}

func flagPassed(fs *flag.FlagSet, name string) bool {
	passed := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			passed = true
		}
	})
	return passed
}

func printSubnetsTable(subnets []sl.SubnetInfo, all bool) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	for _, subnet := range subnets {
		kind := "public"
		if subnet.Private {
			kind = "private"
		}
		fmt.Fprintf(w, "----- %s\t%s\tgw %s\t-----\n", subnet.CIDR, kind, subnet.Gateway)
		for _, ip := range subnet.IPs {
			if all {
				fmt.Fprintf(w, "%s\t%s\t%s\n", ip.IP, ip.PTR, ip.Note)
			} else {
				fmt.Fprintf(w, "%s\n", ip.IP)
			}
		}
	}
	return w.Flush()
}

func printJSON(obj any) error {
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// confirm asks the user for a yes/no confirmation, defaulting to no.
func confirm() bool {
	fmt.Print("(y/N): ")
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return false
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer == "y" || answer == "yes" {
		return true
	}
	fmt.Println("canceled")
	return false
}
