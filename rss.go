package main

import (
	"blogaggregator/internal/database"
	"blogaggregator/models"
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func fetchFeed(ctx context.Context, feedURL string) (*models.RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "gator/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 299 {
		return nil, fmt.Errorf("failed to fetch feed '%s': %s", feedURL, string(body))
	}

	var rssFeed models.RSSFeed
	if err := xml.Unmarshal(body, &rssFeed); err != nil {
		return nil, fmt.Errorf("failed to parse feed '%s': %v", feedURL, err)
	}

	return &rssFeed, nil
}

func scrapeFeeds(ctx context.Context, state *models.State) error {
	feed, err := state.DBQueries.GetNextFeedToFetch(ctx)
	if err != nil {
		return fmt.Errorf("failed to get next feed to fetch: %v", err)
	}

	// Yes, we could mark as fetched after a successful fetch,
	// but this way we avoid fetching the same feed multiple times in case of errors.
	if err := state.DBQueries.MarkFeedFetched(ctx, feed.ID); err != nil {
		return fmt.Errorf("failed to update feed last fetched: %v", err)
	}

	dt := time.Now()
	fmt.Printf("(%s) Fetching feed: %s (%s)\n", dt.Format("2006-01-02 15:04:05"), feed.Name, feed.Url)

	rssFeed, err := fetchFeed(ctx, feed.Url)
	if err != nil {
		return fmt.Errorf("failed to fetch feed: %v", err)
	}

	for _, item := range rssFeed.Channel.Item {
		// We ignore errors here, as they'll almost certainly be due
		// to dupes, and we don't want to stop processing the rest of the feed.
		state.DBQueries.AddPost(ctx, database.AddPostParams{
			ID:          uuid.New(),
			Title:       item.Title,
			Url:         item.Link,
			Description: sql.NullString{String: item.Description, Valid: item.Description != ""},
			FeedID:      feed.ID,
		})
	}

	fmt.Println()

	return nil
}
