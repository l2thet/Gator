package main

import (
	"fmt"
	"log"
	"os"

	"github.com/l2thet/Gator/internal/config"
)

type State struct {
	*config.Config
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

	s := &State{Config: &cfg}

	cmds := &Commands{
		callback: make(map[string]func(*State, Command) error),
	}

	cmds.register("login", handlerLogin)

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

	err := s.SetUser(cmd.Args[0])
	if err != nil {
		log.Fatalf("Error setting user: %v", err)
		return err
	}

	fmt.Printf("%s has been set as the current user\n", cmd.Args[0])

	return nil
}
