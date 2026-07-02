package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func showCommandHelp(name, summary string, fs *flag.FlagSet) {
	fmt.Fprintf(os.Stdout, "Usage: softlayer %s [options]\n\n%s\n\nOptions:\n", name, summary)
	fs.SetOutput(os.Stdout)
	fs.PrintDefaults()
}

func helpRequested(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "-h", "--help", "-help", "help":
			return true
		}
	}
	return false
}

func stripHelpArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		switch arg {
		case "-h", "--help", "-help", "help":
			continue
		default:
			out = append(out, arg)
		}
	}
	return out
}

func isLegacyInvocation(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "list", "set", "clear", "lease", "stale", "completion", "version", "help",
		"-h", "--help", "-version", "--version":
		return false
	}
	return strings.HasPrefix(args[0], "-")
}

// translateLegacyArgs maps the old flat-flag CLI to subcommands.
func translateLegacyArgs(args []string) []string {
	args = stripHelpArgs(args)
	fs := flag.NewFlagSet("softlayer", flag.ContinueOnError)
	fs.SetOutput(ioDiscard{})

	force := fs.Bool("force", false, "")
	note := fs.String("note", "", "")
	ttl := fs.Int("ttl", 3600, "")
	ptr := fs.String("ptr", "", "")
	ip := fs.String("ip", "", "")
	list := fs.Bool("list", false, "")
	listPublic := fs.Bool("public", false, "")
	listPrivate := fs.Bool("private", false, "")
	one := fs.Bool("one", false, "")
	lease := fs.Bool("lease", false, "")
	listStale := fs.Bool("liststale", false, "")
	versionFlag := fs.Bool("version", false, "")

	// Old binary accepted both -liststale and --liststale style names.
	_ = fs.Bool("debug", false, "")

	if err := fs.Parse(args); err != nil {
		return args
	}

	if *versionFlag {
		return []string{"version"}
	}
	if *listStale {
		return append([]string{"stale"}, passthroughFlags(fs)...)
	}
	if *lease {
		out := []string{"lease"}
		if flagPassed(fs, "ptr") {
			out = append(out, "-ptr", *ptr)
		}
		if flagPassed(fs, "note") {
			out = append(out, "-note", *note)
		}
		if flagPassed(fs, "ttl") {
			out = append(out, "-ttl", fmt.Sprintf("%d", *ttl))
		}
		if *force {
			out = append(out, "-force")
		}
		out = append(out, passthroughFlags(fs)...)
		return out
	}
	if *list {
		out := []string{"list"}
		if *listPublic {
			out = append(out, "-public")
		}
		if *listPrivate {
			out = append(out, "-private")
		}
		if *one {
			out = append(out, "-one")
		}
		out = append(out, passthroughFlags(fs)...)
		return out
	}
	if *ip != "" {
		out := []string{"set", "-ip", *ip}
		if flagPassed(fs, "ptr") {
			out = append(out, "-ptr", *ptr)
		}
		if flagPassed(fs, "note") {
			out = append(out, "-note", *note)
		}
		if flagPassed(fs, "ttl") {
			out = append(out, "-ttl", fmt.Sprintf("%d", *ttl))
		}
		if *force {
			out = append(out, "-force")
		}
		return out
	}

	return args
}

func showLegacyHelp(args []string) error {
	translated := translateLegacyArgs(args)
	if len(translated) == 0 {
		fmt.Println(usage)
		return nil
	}
	switch translated[0] {
	case "list":
		return cmdList([]string{"-h"})
	case "set":
		return cmdSet([]string{"-h"})
	case "clear":
		return cmdClear([]string{"-h"})
	case "lease":
		return cmdLease([]string{"-h"})
	case "stale":
		return cmdStale([]string{"-h"})
	case "version":
		fmt.Println("Version:", version)
		return nil
	default:
		fmt.Println(usage)
		return nil
	}
}

func passthroughFlags(fs *flag.FlagSet) []string {
	var out []string
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "list", "lease", "liststale", "public", "private", "one", "version", "debug",
			"ip", "ptr", "note", "ttl", "force":
			return
		}
		if f.Value.String() == "true" {
			out = append(out, "-"+f.Name)
			return
		}
		if f.Value.String() != "false" {
			out = append(out, "-"+f.Name, f.Value.String())
		}
	})
	return out
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
