package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/bytefmt"

	"gopkg.in/dfsr.v0/core"
	"gopkg.in/dfsr.v0/dfsrflag"
	"gopkg.in/dfsr.v0/manifest"
)

const bufferSize = 67108864 // 64 mebibytes

func main() {
	fs := flag.NewFlagSet("", flag.ExitOnError)
	usage := func(errmsg string) {
		fmt.Fprintf(os.Stderr, "%s\n\n", errmsg)
		fs.Usage()
		os.Exit(2)
	}

	var (
		include      dfsrflag.RegexpSlice
		exclude      dfsrflag.RegexpSlice
		types        dfsrflag.RegexpSlice
		after        string
		before       string
		when         string
		domain       string
		resolv       bool
		domainConfig core.Domain
	)

	fs.Var(&include, "i", "regular expression for file match (inclusion)")
	fs.Var(&exclude, "e", "regular expression for file match (exclusion)")
	fs.Var(&types, "t", "resource type (delete, conflict)")
	fs.StringVar(&after, "after", "", "start date/time (YYYY-MM-DD[ H:M:S])")
	fs.StringVar(&before, "before", "", "end date/time (YYYY-MM-DD[ H:M:S])")
	fs.StringVar(&when, "when", "", "day to include (today, yesterday, YYYY-MM-DD)")
	fs.StringVar(&domain, "domain", "", "Active Directory domain to query for partner resolution")
	fs.BoolVar(&resolv, "r", false, "Perform partner resolution by querying Active Directory domain via ADSI")
	fs.Usage = makeUsageFunc(fs, os.Args[0], "")

	if len(os.Args) < 2 {
		usage("No command specified.")
	}

	var (
		command = strings.ToLower(os.Args[1])
		args    = os.Args[2:]
		list    bool
		dump    bool
	)

	switch command {
	case "summary":
	case "list":
		list = true
	case "dump":
		list = true
		dump = true
	default:
		usage(fmt.Sprintf("Unknown command \"%s\".", os.Args[1]))
	}

	fs.Usage = makeUsageFunc(fs, os.Args[0], command)
	fs.Parse(args)

	paths := fs.Args()
	total := len(paths)
	if total == 0 {
		usage("No paths specified.")
	}

	filter := parseFilter(include, exclude, types, after, before, when, usage)

	if resolv {
		var err error
		domain, domainConfig, err = resolve(domain)
		if err != nil {
			fmt.Printf("Unable to retrieve Active Directory domain configuration: %v\n", err)
			os.Exit(2)
		}
	}

	results := make([]Output, total)
	for i := 0; i < total; i++ {
		results[i] = make(Output, bufferSize)
	}

	for i, path := range paths {
		go run(path, filter, list, dump, &domainConfig, results[i])
	}

	for i := 0; i < total; i++ {
		for line := range results[i] {
			fmt.Print(line)
		}
	}
}

func run(path string, filter manifest.Filter, list, dump bool, domain *core.Domain, output Output) {
	defer close(output)

	mpath := manifest.Find(path)
	if mpath == "" {
		output.Printf("Manifest not found for %s\n", path)
		return
	}

	output.Printf("-------- %s --------\n", mpath)
	defer output.Printf("-------- %s --------\n", mpath)

	m := manifest.New(mpath)

	var total, filtered manifest.Stats

	var err error
	if !list {
		filtered, total, err = m.Stats(filter)
		if err != nil {
			output.Printf("%v\n", err)
			return
		}
	} else {
		filtered, total, err = enumerate(m, filter, dump, domain, output)
		if err != nil {
			output.Printf("%v\n", err)
			return
		}
		if filtered.Entries > 0 {
			output.Printf("\n")
		}
	}

	info, err := m.Info()
	if err != nil {
		output.Printf("%v\n", err)
		return
	}

	modified := info.Modified.In(time.Local).Format(time.RFC3339)
	output.Printf("Manifest\n")
	output.Printf("  Size: %s, Updated: %s\n", bytefmt.ByteSize(uint64(info.Size)), modified)
	output.Printf("Manifest Data\n")
	output.Printf("  TOTAL    %s\n", total.Summary())
	if filter != nil {
		output.Printf("  MATCHING %s\n", filtered.Summary())
	}
}

func enumerate(m *manifest.Manifest, filter manifest.Filter, dump bool, domain *core.Domain, output Output) (total, filtered manifest.Stats, err error) {
	members := domain.MemberInfoMap()

	c, err := m.AdvancedCursor(members.Resolve, filter)
	if err != nil {
		return
	}
	defer c.Close()

	for {
		var r manifest.Resource
		r, err = c.Read()
		if err == io.EOF {
			err = nil
			break
		}
		if err != nil {
			return
		}

		t := r.Time.In(time.Local).Format(time.RFC3339)

		if dump {
			b, mErr := xml.MarshalIndent(&r, "", "  ")
			if mErr == nil {
				output.Printf("%s\n", string(b))
			}
		} else if r.PartnerHost != "" {
			output.Printf("%s [%s:%s]: %s\n", t, r.PartnerHost, r.Type, r.Path)
		} else {
			output.Printf("%s [%s:%s]: %s\n", t, r.PartnerGUID, r.Type, r.Path)
		}
	}
	total, filtered = c.Stats()
	return
}
