package main

import (
	"fmt"
	"os"
	"strings"
)

// FlagSet is a stupid flag system that parses long flags only and gives you
// unparsed input (free form input) and the flags with their values.
type FlagSet struct {
	Values  map[string]string
	Strings []string
}

func MakeFlagSet(Values map[string]string) *FlagSet {
	return &FlagSet{
		Values:  Values,
		Strings: []string{},
	}
}

func (s *FlagSet) Parse() error {
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "--" {
			return nil
		}

		if strings.HasPrefix(arg, "--") {
			name := arg[2:]
			if _, has := s.Values[name]; !has {
				return fmt.Errorf("unknown flag %s", name)
			}

			if i+1 >= len(os.Args) {
				return fmt.Errorf("no value for flag %s", name)
			}

			s.Values[name] = os.Args[i+1]
			i++
			continue
		}

		s.Strings = append(s.Strings, arg)
	}

	return nil
}
