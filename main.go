package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/l2thet/Gator/internal/config"
	"github.com/l2thet/Gator/internal/database"
	_ "github.com/lib/pq"
)

type State struct {
	db  *database.Queries
	cfg *config.Config
}

type Command struct {
	Name string
	Args []string
}

type Commands struct {
	callback map[string]func(*State, Command) error
}

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	s := &State{cfg: &cfg}

	db, err := sql.Open("postgres", s.cfg.DbURL)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	s.db = database.New(db)

	cmds := &Commands{
		callback: make(map[string]func(*State, Command) error),
	}

	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)

	args := os.Args
	if len(args) < 2 {
		log.Fatalf("usage: %s <command> [args]", args[0])
		os.Exit(0)
	}

	cmd := Command{Name: args[1], Args: args[2:]}

	err = cmds.run(s, cmd)
	if err != nil {
		log.Fatalf("Error running command: %v", err)
	}
}

func (c *Commands) register(name string, f func(*State, Command) error) {
	c.callback[name] = f
}

func (c *Commands) run(s *State, cmd Command) error {
	if f, ok := c.callback[cmd.Name]; ok {
		return f(s, cmd)
	}

	err := fmt.Errorf("command not found: %s", cmd.Name)
	return err
}

func handlerLogin(s *State, cmd Command) error {
	if len(cmd.Args) != 1 {
		err := fmt.Errorf("usage: %s <username>", cmd.Name)
		return err
	}

	_, err := s.db.GetUser(context.Background(), cmd.Args[0])
	if err != nil {
		log.Fatalf("Error getting user: %v", err)
		os.Exit(1)
	}

	err = s.cfg.SetUser(cmd.Args[0])
	if err != nil {
		log.Fatalf("Error setting user: %v", err)
		return err
	}

	fmt.Printf("%s has been set as the current user\n", cmd.Args[0])

	return nil
}

func handlerRegister(s *State, cmd Command) error {
	if len(cmd.Args) != 1 {
		err := fmt.Errorf("usage: %s <username>", cmd.Name)
		return err
	}

	user, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.Args[0],
	})
	if err != nil {
		log.Fatalf("Error creating user: %v", err)
		os.Exit(1)
	}

	s.cfg.SetUser(user.Name)
	fmt.Printf("User %s has been created and set as the current user\n", user.Name)
	fmt.Printf("User: %+v\n", user)

	return nil
}
