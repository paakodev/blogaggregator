package main

import (
	"blogaggregator/internal/config"
	"blogaggregator/internal/database"
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type command struct {
	name string
	args []string
}

type commandRegistry struct {
	cmds map[string]func(*state, command) error
}

func (c *commandRegistry) run(state *state, cmd command) error {
	if handler, exists := c.cmds[cmd.name]; exists {
		return handler(state, cmd)
	}
	return fmt.Errorf("unknown command: %s", cmd.name)
}

func (c *commandRegistry) register(name string, f func(*state, command) error) {
	c.cmds[name] = f
}

func newCommandRegistry() *commandRegistry {
	return &commandRegistry{
		cmds: make(map[string]func(*state, command) error),
	}
}

type state struct {
	config *config.Config
	db     *database.Queries
}

func main() {
	cfg, err := config.Read()
	if err != nil {
		panic(fmt.Sprintf("failed to read config: %v", err))
	}
	db, err := sql.Open("postgres", cfg.DbURL)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to database: %v", err))
	}
	defer db.Close()

	state := &state{config: cfg, db: database.New(db)}
	registry := newCommandRegistry()
	registry.register("login", handlerLogin)
	registry.register("register", handlerRegister)
	registry.register("reset", handlerReset)

	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("No command provided")
		os.Exit(1)
	}

	cmd := command{name: args[0], args: args[1:]}
	if err := registry.run(state, cmd); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func handlerLogin(state *state, cmd command) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("username is required")
	}

	username := cmd.args[0]
	user, err := state.db.GetUserByName(context.Background(), username)
	if err != nil {
		return fmt.Errorf("failed to get user: %v", err)
	}
	if err := state.config.SetUser(user.Name); err != nil {
		return fmt.Errorf("failed to set user: %v", err)
	}

	fmt.Printf("Logged in as %s\n", user.Name)
	return nil
}

func handlerRegister(state *state, cmd command) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("username is required")
	}

	username := cmd.args[0]
	userID := uuid.New()

	user, err := state.db.CreateUser(
		context.Background(),
		database.CreateUserParams{
			ID:   userID,
			Name: username,
		})
	if err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}

	if err := state.config.SetUser(user.Name); err != nil {
		return fmt.Errorf("failed to set user: %v", err)
	}

	fmt.Println("User created")
	fmt.Printf("Name:       %s\n", user.Name)
	fmt.Printf("ID:         %s\n", user.ID)
	fmt.Printf("Created:    %s\n", user.CreatedAt)
	fmt.Printf("Updated:    %s\n", user.UpdatedAt)
	return nil
}

func handlerReset(state *state, cmd command) error {
	if err := state.db.Reset(context.Background()); err != nil {
		return fmt.Errorf("failed to reset database: %v", err)
	}
	fmt.Println("Database reset successfully")
	return nil
}
