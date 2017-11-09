package main

import (
	"time"

	"github.com/jinzhu/now"
)

func parseStart(s string) (start time.Time, err error) {
	start, ref := parseTimeReference(s)
	start = now.New(start).BeginningOfDay()
	if ref {
		return
	}
	start, err = now.New(start).Parse(s) // FIXME: This is overwriting start when it shouldn't
	return
}

func parseEnd(s string) (end time.Time, err error) {
	end, ref := parseTimeReference(s)
	end = now.New(end).EndOfDay()
	if ref {
		return
	}
	end, err = now.New(end).Parse(s) // FIXME: This is overwriting end when it shouldn't
	return
}

func parseWhen(s string) (start, end time.Time, err error) {
	when, ref := parseTimeReference(s)
	start = now.New(when).BeginningOfDay()
	end = now.New(when).EndOfDay()
	if ref {
		return
	}
	start, err = now.New(start).Parse(s)
	if err != nil {
		return
	}
	end, err = now.New(end).Parse(s)
	return
}

func parseTimeReference(s string) (when time.Time, ref bool) {
	switch s {
	case "today":
		return time.Now(), true
	case "yesterday":
		return time.Now().AddDate(0, 0, -1), true
	default:
		return time.Now(), false
	}
}
