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
