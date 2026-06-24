package main

import (
	"blogaggregator/internal/config"
	"blogaggregator/internal/database"
	"blogaggregator/models"
	"context"
	"database/sql"
	"fmt"
	"os"

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
	registry.Register("addfeed", handlerAddFeed)
	registry.Register("feeds", handlerFeeds)
	registry.Register("follow", handlerFollowFeed)
	registry.Register("following", handlerFollowingFeeds)

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
	url := "https://www.wagslane.dev/index.xml"
	rssFeed, err := fetchFeed(context.Background(), url)
	if err != nil {
		return fmt.Errorf("failed to fetch feed: %v", err)
	}

	fmt.Printf("Feed:\n%#v\n", rssFeed)
	return nil
}

func handlerAddFeed(state *models.State, cmd models.Command) error {
	if len(cmd.Args) < 2 {
		return fmt.Errorf("feed name and URL are required")
	}

	feedName := cmd.Args[0]
	feedURL := cmd.Args[1]

	user, err := state.DBQueries.GetUserByName(context.Background(), state.Config.CurrentUserName)
	if err != nil {
		return fmt.Errorf("failed to get current user: %v", err)
	}

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

func handlerFollowFeed(state *models.State, cmd models.Command) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("feed URL is required")
	}
	feedURL := cmd.Args[0]
	feed, err := state.DBQueries.GetFeedByURL(context.Background(), feedURL)
	if err != nil {
		return fmt.Errorf("failed to get feed by URL: %v", err)
	}

	user, err := state.DBQueries.GetUserByName(context.Background(), state.Config.CurrentUserName)
	if err != nil {
		return fmt.Errorf("failed to get current user: %v", err)
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

func handlerFollowingFeeds(state *models.State, cmd models.Command) error {
	followedFeeds, err := state.DBQueries.GetFeedFollowsForUser(
		context.Background(),
		state.Config.CurrentUserName,
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
