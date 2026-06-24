package models

import (
	"blogaggregator/internal/config"
	"blogaggregator/internal/database"
	"fmt"
)

type Command struct {
	Name string
	Args []string
}

type CommandRegistry struct {
	Cmds map[string]func(*State, Command) error
}

func (c *CommandRegistry) Run(state *State, cmd Command) error {
	if handler, exists := c.Cmds[cmd.Name]; exists {
		return handler(state, cmd)
	}
	return fmt.Errorf("unknown command: %s", cmd.Name)
}

func (c *CommandRegistry) Register(name string, f func(*State, Command) error) {
	c.Cmds[name] = f
}

func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		Cmds: make(map[string]func(*State, Command) error),
	}
}

type State struct {
	Config *config.Config
	DB     *database.Queries
}
