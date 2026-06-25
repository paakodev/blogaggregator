package main

import (
	"blogaggregator/internal/config"
	"blogaggregator/internal/database"
	"blogaggregator/models"
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

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

	state := &models.State{Config: cfg, DBQueries: database.New(db), DB: db}

	registry := models.NewCommandRegistry()
	registry.Register("login", handlerLogin)
	registry.Register("register", handlerRegister)
	registry.Register("reset", handlerReset)
	registry.Register("users", handlerUsers)
	registry.Register("agg", handlerAgg)
	registry.Register("addfeed", middlewareLoggedIn(handlerAddFeed))
	registry.Register("feeds", handlerFeeds)
	registry.Register("follow", middlewareLoggedIn(handlerFollowFeed))
	registry.Register("following", middlewareLoggedIn(handlerFollowingFeeds))
	registry.Register("unfollow", middlewareLoggedIn(handlerUnfollowFeed))
	registry.Register("browse", middlewareLoggedIn(handlerBrowse))

	registry.Register("help", func(state *models.State, cmd models.Command) error {
		fmt.Println("Available commands:")
		fmt.Println("  login <username>          - Log in as a user")
		fmt.Println("  register <username>       - Register a new user")
		fmt.Println("  reset                     - Reset the database")
		fmt.Println("  users                     - List all users")
		fmt.Println("  agg <interval>            - Start feed aggregation with the specified interval (e.g., '10s', '1m')")
		fmt.Println("  addfeed <name> <url>      - Add a new feed and follow it (requires login)")
		fmt.Println("  feeds                     - List all feeds")
		fmt.Println("  follow <url>              - Follow a feed by URL (requires login)")
		fmt.Println("  following                 - List followed feeds (requires login)")
		fmt.Println("  unfollow <url>            - Unfollow a feed by URL (requires login)")
		fmt.Println("  browse                    - Browse posts from followed feeds (requires login)")
		return nil
	})

	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("No command provided")
		os.Exit(1)
	}

	cmd := models.Command{Name: args[0], Args: args[1:]}
	if err := registry.Run(state, cmd); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func middlewareLoggedIn(handler func(s *models.State, cmd models.Command, user database.User) error) func(*models.State, models.Command) error {
	return func(state *models.State, cmd models.Command) error {
		user, err := state.DBQueries.GetUserByName(context.Background(), state.Config.CurrentUserName)
		if err != nil {
			return fmt.Errorf("failed to get user: %v", err)
		}
		return handler(state, cmd, user)
	}
}

func handlerLogin(state *models.State, cmd models.Command) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("username is required")
	}

	username := cmd.Args[0]
	user, err := state.DBQueries.GetUserByName(context.Background(), username)
	if err != nil {
		return fmt.Errorf("failed to get user: %v", err)
	}
	if err := state.Config.SetUser(user.Name); err != nil {
		return fmt.Errorf("failed to set user: %v", err)
	}

	fmt.Printf("Logged in as %s\n", user.Name)
	return nil
}

func handlerRegister(state *models.State, cmd models.Command) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("username is required")
	}

	username := cmd.Args[0]
	userID := uuid.New()

	user, err := state.DBQueries.CreateUser(
		context.Background(),
		database.CreateUserParams{
			ID:   userID,
			Name: username,
		})
	if err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}

	if err := state.Config.SetUser(user.Name); err != nil {
		return fmt.Errorf("failed to set user: %v", err)
	}

	fmt.Println("User created")
	fmt.Printf("Name:       %s\n", user.Name)
	fmt.Printf("ID:         %s\n", user.ID)
	fmt.Printf("Created:    %s\n", user.CreatedAt)
	fmt.Printf("Updated:    %s\n", user.UpdatedAt)
	return nil
}

func handlerReset(state *models.State, cmd models.Command) error {
	if err := state.DBQueries.Reset(context.Background()); err != nil {
		return fmt.Errorf("failed to reset database: %v", err)
	}
	fmt.Println("Database reset successfully")
	return nil
}

func handlerUsers(state *models.State, cmd models.Command) error {
	users, err := state.DBQueries.GetAllUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get users: %v", err)
	}

	fmt.Println("Users:")
	for _, user := range users {
		if user.Name == state.Config.CurrentUserName {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}
	return nil
}

func handlerAgg(state *models.State, cmd models.Command) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("time interval is required")
	}

	interval, err := time.ParseDuration(cmd.Args[0])
	if err != nil {
		return fmt.Errorf("invalid time interval: %v", err)
	}

	fmt.Printf("Collecting feeds every %s\n", interval)
	ticker := time.NewTicker(interval)
	for ; ; <-ticker.C {
		if err := scrapeFeeds(context.Background(), state); err != nil {
			fmt.Printf("Error scraping feeds: %v\n", err)
		}
	}
	return nil
}

func handlerAddFeed(state *models.State, cmd models.Command, user database.User) error {
	if len(cmd.Args) < 2 {
		return fmt.Errorf("feed name and URL are required")
	}

	feedName := cmd.Args[0]
	feedURL := cmd.Args[1]

	tx, err := state.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Commit()
	qtx := state.DBQueries.WithTx(tx)

	feedID := uuid.New()
	feed, err := qtx.CreateFeed(
		context.Background(),
		database.CreateFeedParams{
			ID:     feedID,
			Name:   feedName,
			Url:    feedURL,
			UserID: user.ID,
		})
	if err != nil {
		return fmt.Errorf("failed to create feed: %v", err)
	}

	newFollowID := uuid.New()
	newFollow, err := qtx.CreateFeedFollow(
		context.Background(),
		database.CreateFeedFollowParams{
			ID:     newFollowID,
			FeedID: feed.ID,
			UserID: user.ID,
		})
	if err != nil {
		return fmt.Errorf("failed to follow feed: %v", err)
	}
	tx.Commit()

	fmt.Println("Feed added and followed successfully")
	fmt.Printf("Name:       %s\n", feed.Name)
	fmt.Printf("URL:        %s\n", feed.Url)
	fmt.Printf("ID:         %s\n", feed.ID)
	fmt.Printf("Follow ID:  %s\n", newFollow.ID)
	fmt.Printf("Created:    %s\n", feed.CreatedAt)
	fmt.Printf("Updated:    %s\n", feed.UpdatedAt)
	return nil
}

func handlerUnfollowFeed(state *models.State, cmd models.Command, user database.User) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("feed URL is required")
	}
	feedURL := cmd.Args[0]
	feed, err := state.DBQueries.GetFeedByURL(context.Background(), feedURL)
	if err != nil {
		return fmt.Errorf("failed to get feed by URL: %v", err)
	}

	if err := state.DBQueries.FeedUnfollow(context.Background(), database.FeedUnfollowParams{
		FeedID: feed.ID,
		UserID: user.ID,
	}); err != nil {
		return fmt.Errorf("failed to unfollow feed: %v", err)
	}

	fmt.Println("Feed unfollowed successfully")
	fmt.Printf("Feed Name:  %s (ID: %s)\n", feed.Name, feed.ID)

	return nil
}

func handlerFeeds(state *models.State, cmd models.Command) error {
	feeds, err := state.DBQueries.GetAllFeedsWithUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list feeds: %s", err)
	}

	fmt.Println("Feeds:")
	for _, feed := range feeds {
		fmt.Printf("* %s (%s) - %s\n", feed.Name, feed.Url, feed.UserName)
	}
	return nil
}

func handlerFollowFeed(state *models.State, cmd models.Command, user database.User) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("feed URL is required")
	}
	feedURL := cmd.Args[0]
	feed, err := state.DBQueries.GetFeedByURL(context.Background(), feedURL)
	if err != nil {
		return fmt.Errorf("failed to get feed by URL: %v", err)
	}

	newFollowID := uuid.New()
	newFollow, err := state.DBQueries.CreateFeedFollow(
		context.Background(),
		database.CreateFeedFollowParams{
			ID:     newFollowID,
			FeedID: feed.ID,
			UserID: user.ID,
		})
	if err != nil {
		return fmt.Errorf("failed to follow feed: %v", err)
	}

	fmt.Println("Feed followed successfully")
	fmt.Printf("Feed Name:  %s (ID: %s)\n", newFollow.FeedName, newFollow.FeedID)

	return nil
}

func handlerFollowingFeeds(state *models.State, cmd models.Command, user database.User) error {
	followedFeeds, err := state.DBQueries.GetFeedFollowsForUser(
		context.Background(),
		user.Name,
	)
	if err != nil {
		return fmt.Errorf("failed to get followed feeds: %v", err)
	}

	fmt.Println("Followed Feeds:")
	for _, feed := range followedFeeds {
		fmt.Printf("* %s (%s)\n", feed.FeedName, feed.FeedID)
	}
	return nil
}

func handlerBrowse(state *models.State, cmd models.Command, user database.User) error {
	var limit = 2
	if len(cmd.Args) > 0 {
		limitArg, err := strconv.Atoi(cmd.Args[0])
		if err != nil {
			return fmt.Errorf("invalid limit: %v", err)
		}
		limit = limitArg
	}

	posts, err := state.DBQueries.GetPostsForUser(
		context.Background(),
		database.GetPostsForUserParams{
			Name:  user.Name,
			Limit: int32(limit),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to get posts for user: %v", err)
	}

	fmt.Printf("Latest %d posts from followed feeds:\n", limit)
	for _, post := range posts {
		fmt.Printf("* %s (%s) - %s\n", post.Title, post.FeedName, post.Url)
	}
	return nil
}
