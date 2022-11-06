package main

import (
	"got/flag"
	"got/formatters"
	"got/types"
	"strconv"
	"strings"
	"time"

	nd "github.com/tj/go-naturaldate"
)

type Input struct {
	Raw map[string]string

	ID        uint64
	Start     time.Time
	End       time.Time
	At        time.Time
	Filter    string
	Formatter types.Formatter

	Command string
	Note    string
}

func GetInput() (Input, error) {
	var res Input

	fs := flag.MakeFlagSet(map[string]string{
		"id": "0",

		"start": "",
		"end":   "",
		"at":    "",

		"filter": "",

		"formatter": "human",
	})
	if err := fs.Parse(); err != nil {
		return res, err
	}

	var err error
	res.ID, err = strconv.ParseUint(fs.Values["id"], 10, 64)
	if err != nil {
		return res, err
	}
	startString := fs.Values["start"]
	endString := fs.Values["end"]
	atString := fs.Values["at"]

	res.Start, err = nd.Parse(startString, time.Now())
	if err != nil {
		res.Start = time.Time{}
	}
	res.End, err = nd.Parse(endString, time.Now())
	if err != nil {
		res.End = time.Time{}
	}
	res.At, err = nd.Parse(atString, time.Now())
	if err != nil {
		res.At = time.Time{}
	}
	res.Filter = fs.Values["filter"]
	switch fs.Values["formatter"] {
	case "human":
		res.Formatter = &formatters.Human{}
	case "json", "JSON":
		res.Formatter = &formatters.JSON{}

	default:
		panic("invalid formatter " + fs.Values["formatter"])
	}

	if len(fs.Strings) > 0 {
		res.Command = fs.Strings[0]
	}
	for i := 1; i < len(fs.Strings); i++ {
		if i > 1 {
			res.Note += " "
		}

		res.Note += fs.Strings[i]
	}
	res.Note = strings.TrimSpace(res.Note)

	res.Raw = fs.Values

	return res, nil
}
