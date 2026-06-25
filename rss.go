package main

import (
	"blogaggregator/models"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
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

	fmt.Printf("Fetching feed: %s (%s)\n", feed.Name, feed.Url)

	rssFeed, err := fetchFeed(ctx, feed.Url)
	if err != nil {
		return fmt.Errorf("failed to fetch feed: %v", err)
	}

	for _, item := range rssFeed.Channel.Item {
		fmt.Printf("%s\n", item.Title)
		fmt.Printf("\t %s\n", item.Link)
	}

	fmt.Println()

	return nil
}
