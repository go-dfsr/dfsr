package main

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/dfsr.v0/dfsrflag"
	"gopkg.in/dfsr.v0/manifest"
	"gopkg.in/dfsr.v0/manifest/mfilter"
)

func parseFilter(include, exclude, types dfsrflag.RegexpSlice, after, before, when string, usage func(string)) manifest.Filter {
	var filters []manifest.Filter

	if len(include) != 0 {
		subfilters := make([]manifest.Filter, 0, len(include))
		for _, i := range include {
			subfilters = append(subfilters, mfilter.PathRegexp(i))
		}
		filters = append(filters, mfilter.Or(subfilters...))
	}

	if len(exclude) != 0 {
		subfilters := make([]manifest.Filter, 0, len(exclude))
		for _, e := range exclude {
			subfilters = append(subfilters, mfilter.Not(mfilter.PathRegexp(e)))
		}
		filters = append(filters, mfilter.And(subfilters...))
	}

	if len(types) != 0 {
		subfilters := make([]manifest.Filter, 0, len(types))
		for _, t := range types {
			subfilters = append(subfilters, mfilter.TypeRegexp(t))
		}
		filters = append(filters, mfilter.And(subfilters...))
	}

	if when != "" {
		if after != "" || before != "" {
			usage("Cannot use when flag in combination with after or before.")
		}
		a, b, err := parseWhen(when)
		if err != nil {
			fmt.Printf("Invalid start/end date: %v\n", err)
			os.Exit(2)
		}
		filters = append(filters, mfilter.After(a))
		filters = append(filters, mfilter.Before(b))
	}

	if after != "" {
		a, err := parseStart(after)
		if err != nil {
			fmt.Printf("Invalid start time: %v\n", err)
			os.Exit(2)
		}
		filters = append(filters, mfilter.After(a))
	}

	if before != "" {
		b, err := parseEnd(before)
		if err != nil {
			fmt.Printf("Invalid end time: %v\n", err)
			os.Exit(2)
		}
		filters = append(filters, mfilter.Before(b))
	}

	return mfilter.And(filters...)
}

func compileRegex(re string, usage func(string)) *regexp.Regexp {
	if re == "" {
		return nil
	}

	c, err := regexp.Compile(re)
	if err != nil {
		usage(fmt.Sprintf("Unable to compile regular expression \"%s\": %v\n", re, err))
	}
	return c
}
