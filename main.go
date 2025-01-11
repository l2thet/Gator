package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/l2thet/Gator/internal/config"
	"github.com/l2thet/Gator/internal/database"
	"github.com/l2thet/Gator/internal/rss"
	"github.com/lib/pq"
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
	cmds.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	cmds.register("browse", middlewareLoggedIn(handlerBrowse))

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

func scrapeFeeds(s *State) error {
	err := s.db.ResetFeedsToFetch(context.Background())
	if err != nil {
		err := fmt.Errorf("error resetting feeds to fetch: %v", err)
		return err
	}

	feeds, err := s.db.GetFeedstoFetch(context.Background())
	if err != nil {
		err := fmt.Errorf("error getting feeds to fetch: %v", err)
		return err
	}

	for _, feed := range feeds {
		articles, err := rss.FetchFeed(context.Background(), feed.Url)
		if err != nil {
			err := fmt.Errorf("error fetching feed: %v", err)
			return err
		}

		err = s.db.MarkFeedFetched(context.Background(), feed.ID)
		if err != nil {
			err := fmt.Errorf("error marking feed fetched: %v", err)
			return err
		}

		for _, article := range articles.Channel.Item {
			description := sql.NullString{
				String: article.Description,
				Valid:  article.Description != "",
			}

			pubDate, err := time.Parse(time.RFC1123Z, article.PubDate)
			if err != nil {
				fmt.Printf("Error parsing publication date: %v\n", err)
				continue
			}

			_, err = s.db.CreatePost(context.Background(), database.CreatePostParams{
				ID:          uuid.New(),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				FeedID:      feed.ID,
				Title:       article.Title,
				Url:         article.Link,
				Description: description,
				PublishedAt: pubDate,
			})
			if err != nil {
				if pqErr, ok := err.(*pq.Error); ok {
					if pqErr.Code == "23505" && pqErr.Constraint == "posts_url_key" {
						continue
					}
				}
				err := fmt.Errorf("error creating post: %v", err)
				return err
			}

			fmt.Printf("Article: %s\n", article.Title)
		}
	}

	return nil
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
	if len(cmd.Args) != 1 {
		err := fmt.Errorf("usage: %s <single|continuous>", cmd.Name)
		return err
	}

	if cmd.Args[0] == "single" {
		err := scrapeFeeds(s)
		if err != nil {
			err := fmt.Errorf("error scraping feeds: %v", err)
			return err
		}
	} else if cmd.Args[0] == "continuous" {
		fmt.Printf("Continuous mode not implemented yet\n")
	}

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

func handlerUnfollow(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) != 1 {
		err := fmt.Errorf("usage: %s <feed url>", cmd.Name)
		return err
	}

	feedID, err := s.db.GetFeedIdByUrl(context.Background(), cmd.Args[0])
	if err != nil {
		err := fmt.Errorf("error getting feed id by url: %v", err)
		return err
	}

	err = s.db.UnfollowFeedFollow(context.Background(), database.UnfollowFeedFollowParams{
		UserID: user.ID,
		FeedID: feedID,
	})
	if err != nil {
		err := fmt.Errorf("error unfollowing feed: %v", err)
		return err
	}

	fmt.Printf("User %s has unfollowed feed %s\n", user.Name, cmd.Args[0])

	return nil
}

func handlerBrowse(s *State, cmd Command, user database.User) error {
	if len(cmd.Args) > 1 {
		err := fmt.Errorf("usage: %s <limit#>(optional)", cmd.Name)
		return err
	}

	limit := 2

	if len(cmd.Args) == 1 {
		var err error
		limit, err = strconv.Atoi(cmd.Args[0])
		if err != nil {
			return fmt.Errorf("invalid limit value: %v", err)
		}
	}

	posts, err := s.db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	})
	if err != nil {
		err := fmt.Errorf("error getting posts for user: %v", user.Name)
		return err
	}

	for _, post := range posts {
		fmt.Printf("* Title: %s\n", post.Title)
		fmt.Printf("* Url: %s\n", post.Url)
		fmt.Printf("* Description: %s\n", post.Description.String)
		fmt.Printf("* Published At: %s\n", post.PublishedAt)
		fmt.Printf("\n")
	}

	return nil
}
