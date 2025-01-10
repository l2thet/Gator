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
	"github.com/l2thet/Gator/internal/rss"
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
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)
	cmds.register("agg", handlerAgg)
	cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmds.register("feeds", handlerFeeds)
	cmds.register("follow", middlewareLoggedIn(handlerFollow))
	cmds.register("following", middlewareLoggedIn(handlerFollowing))

	args := os.Args
	if len(args) < 2 {
		log.Fatalf("usage: %s <command> [args]", args[0])
		os.Exit(1)
	}

	cmd := Command{Name: args[1], Args: args[2:]}

	err = cmds.run(s, cmd)
	if err != nil {
		log.Fatalf("Error running command: %v", err)
		os.Exit(1)
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

func middlewareLoggedIn(handler func(s *State, cmd Command, user database.User) error) func(*State, Command) error {
	return func(s *State, cmd Command) error {
		user, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
		if err != nil {
			err := fmt.Errorf("error getting user: %v", err)
			return err
		}

		return handler(s, cmd, user)
	}
}

func handlerLogin(s *State, cmd Command) error {
	if len(cmd.Args) != 1 {
		err := fmt.Errorf("usage: %s <username>", cmd.Name)
		return err
	}

	_, err := s.db.GetUser(context.Background(), cmd.Args[0])
	if err != nil {
		err := fmt.Errorf("error getting user: %v", err)
		return err
	}

	err = s.cfg.SetUser(cmd.Args[0])
	if err != nil {
		err := fmt.Errorf("error setting user: %v", err)
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
		err := fmt.Errorf("error creating user: %v", err)
		return err
	}

	s.cfg.SetUser(user.Name)
	fmt.Printf("User %s has been created and set as the current user\n", user.Name)
	fmt.Printf("User: %+v\n", user)

	return nil
}

func handlerReset(s *State, cmd Command) error {
	if len(cmd.Args) != 0 {
		err := fmt.Errorf("usage: %s", cmd.Name)
		return err
	}

	err := s.db.DeleteAllUsers(context.Background())
	if err != nil {
		err := fmt.Errorf("error deleting all users: %v", err)
		return err
	}

	fmt.Println("All users have been deleted")
	return nil
}

func handlerUsers(s *State, cmd Command) error {
	if len(cmd.Args) != 0 {
		err := fmt.Errorf("usage: %s", cmd.Name)
		return err
	}

	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		err := fmt.Errorf("error getting users: %v", err)
		return err
	}

	for _, user := range users {
		if user.Name == s.cfg.CurrentUserName {
			fmt.Printf("* %s (current)\n", user.Name)
			continue
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}

	return nil
}

func handlerAgg(s *State, cmd Command) error {
	if len(cmd.Args) != 0 {
		err := fmt.Errorf("usage: %s", cmd.Name)
		return err
	}

	feedURL := "https://www.wagslane.dev/index.xml"

	feed, err := rss.FetchFeed(context.Background(), feedURL)
	if err != nil {
		err := fmt.Errorf("error fetching feed: %v", err)
		return err
	}

	fmt.Printf("Feed: %+v\n", feed)

	return nil
}

func handlerAddFeed(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) != 2 {
		err := fmt.Errorf("usage: %s <name> <url>", cmd.Name)
		return err
	}

	feed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		UserID:    user.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.Args[0],
		Url:       cmd.Args[1],
	})
	if err != nil {
		err := fmt.Errorf("error creating feed: %v", err)
		return err
	}

	_, err = s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		err := fmt.Errorf("error following feed: %v", err)
		return err
	}

	fmt.Printf("Feed has been created:\n %+v \n", feed)

	return nil
}

func handlerFeeds(s *State, cmd Command) error {
	if len(cmd.Args) != 0 {
		err := fmt.Errorf("usage: %s", cmd.Name)
		return err
	}

	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		err := fmt.Errorf("error getting feeds: %v", err)
		return err
	}

	fmt.Printf("Feeds:\n %+v \n", feeds)

	return nil
}

func handlerFollow(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) != 1 {
		err := fmt.Errorf("usage: %s <feed name>", cmd.Name)
		return err
	}

	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		err := fmt.Errorf("error getting feeds: %v", err)
		return err
	}

	var feedID uuid.UUID
	for _, feed := range feeds {
		if feed.Url == cmd.Args[0] {
			feedID = feed.ID
			break
		}
	}

	feedFollow, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feedID,
	})
	if err != nil {
		err := fmt.Errorf("error following feed: %v", err)
		return err
	}

	fmt.Printf("User %s is now following feed %s\n", feedFollow.UserName, feedFollow.FeedName)

	return nil
}

func handlerFollowing(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) != 0 {
		err := fmt.Errorf("usage: %s", cmd.Name)
		return err
	}

	feeds, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		err := fmt.Errorf("error getting feeds for user: %v", err)
		return err
	}

	for _, feed := range feeds {
		fmt.Printf("*User %s is following: %s\n", feed.UserName, feed.FeedName)
	}

	return nil
}
