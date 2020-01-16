package main

import (
	"strings"
)

type Command struct {
	Names       []string
	Description string
	Usage       string
	Fn          func() error
}

func (c *Command) Match(val string) bool {
	for _, name := range c.Names {
		if name == val {
			return true
		}
	}
	return false
}
func (c *Command) MatchPrefix(prefix string) bool {
	for _, name := range c.Names {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

type CommandManager struct {
	// we don't use a map since we want to keep order
	commands []*Command
}

func MakeManager() *CommandManager {
	return &CommandManager{}
}

func (m *CommandManager) AddCommand(names []string, description, usage string, fn func() error) {
	for _, name := range names {
		if m.GetByName(name) != nil {
			panic("command already exist")
		}
	}

	cmd := &Command{
		Names:       names,
		Description: description,
		Usage:       usage,
		Fn:          fn,
	}
	m.commands = append(m.commands, cmd)
}

func (m *CommandManager) GetByPrefix(prefix string) []*Command {
	var res []*Command

	for _, cmd := range m.commands {
		hasPrefix := cmd.MatchPrefix(prefix)

		if hasPrefix {
			res = append(res, cmd)
		}
	}

	return res
}

func (m *CommandManager) GetByName(name string) *Command {
	for _, cmd := range m.commands {
		if cmd.Match(name) {
			return cmd
		}
	}
	return nil
}
