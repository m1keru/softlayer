package main

import (
	"flag"
	"strings"

	"github.com/m1keru/softlayer/internal/sl"
)

// stringList is a repeatable/comma-separated flag value.
type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }

func (s *stringList) Set(v string) error {
	for part := range strings.SplitSeq(v, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			*s = append(*s, part)
		}
	}
	return nil
}

func listOptsFromFlags(public, private, noDefault *bool, excludeNotes, excludeCIDRs *stringList) sl.ListOpts {
	return sl.ListOpts{
		IncludePublic:     *public,
		IncludePrivate:    *private,
		ExcludeNotes:      *excludeNotes,
		ExcludeCIDRs:      *excludeCIDRs,
		NoDefaultExcludes: *noDefault,
	}
}

func registerExcludeFlags(fs *flag.FlagSet, excludeNotes, excludeCIDRs *stringList) *bool {
	fs.Var(excludeNotes, "exclude-note", "exclude subnets whose IBM note contains this string (repeatable, comma-separated)")
	fs.Var(excludeCIDRs, "exclude-cidr", "exclude subnet by CIDR, e.g. 10.0.0.0/24 (repeatable, comma-separated)")
	return fs.Bool("no-default-excludes", false, "do not apply built-in subnet note exclusions")
}
