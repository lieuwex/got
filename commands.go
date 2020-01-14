package main

import (
	"strings"
)

type Command struct {
	Name        string
	Description string
	Usage       string
	Fn          func() error
}

type CommandManager struct {
	// we don't use a map since we want to keep order
	commands []*Command
}

func MakeManager() *CommandManager {
	return &CommandManager{}
}

func (m *CommandManager) AddCommand(name, description, usage string, fn func() error) {
	if m.GetByName(name) != nil {
		panic("command already exist")
	}

	cmd := &Command{
		Name:        name,
		Description: description,
		Usage:       usage,
		Fn:          fn,
	}
	m.commands = append(m.commands, cmd)
}

func (m *CommandManager) GetByPrefix(prefix string) []*Command {
	var res []*Command

	for _, cmd := range m.commands {
		hasPrefix := strings.HasPrefix(cmd.Name, prefix)
		if hasPrefix {
			res = append(res, cmd)
		}
	}

	return res
}

func (m *CommandManager) GetByName(name string) *Command {
	for _, cmd := range m.commands {
		if cmd.Name == name {
			return cmd
		}
	}
	return nil
}
