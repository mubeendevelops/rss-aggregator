package main

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mubeencodes/rss-aggregator/internal/database"
)

// scrapeCycleTimeout bounds the database calls and HTTP requests of one cycle
const scrapeCycleTimeout = time.Minute

// startScraping runs scrape cycles until ctx is cancelled
func startScraping(
	ctx context.Context,
	db database.Querier,
	cfg scraperConfig,
) {
	log.Printf("Scraping on %v goroutines every %s duration, each feed at most once per %s", cfg.Concurrency, cfg.Interval, cfg.MinRefetchInterval)
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()
	for {
		scrapeCycle(ctx, db, cfg.Concurrency, cfg.MinRefetchInterval)

		select {
		case <-ctx.Done():
			log.Println("Scraper stopped")
			return
		case <-ticker.C:
		}
	}
}

// scrapeCycle fetches up to concurrency feeds that were not fetched within minRefetchInterval
func scrapeCycle(ctx context.Context, db database.Querier, concurrency int, minRefetchInterval time.Duration) {
	ctx, cancel := context.WithTimeout(ctx, scrapeCycleTimeout)
	defer cancel()

	feeds, err := db.GetNextFeedsToFetch(
		ctx,
		database.GetNextFeedsToFetchParams{
			FetchedBefore: time.Now().UTC().Add(-minRefetchInterval),
			MaxFeeds:      int32(concurrency),
		},
	)

	if err != nil {
		log.Println("error fetching feeds", err)
		return
	}

	wg := &sync.WaitGroup{}
	for _, feed := range feeds {
		wg.Add(1)

		go scrapeFeed(ctx, db, wg, feed)
	}

	wg.Wait()
}

func scrapeFeed(ctx context.Context, db database.Querier, wg *sync.WaitGroup, feed database.Feed) {
	defer wg.Done()

	_, err := db.MarkFeedAsFetched(ctx, feed.ID)
	if err != nil {
		log.Println("Error marking feed as fetched", err)
		return
	}

	rssFeed, err := urlToFeed(ctx, feed.Url)
	if err != nil {
		log.Println("Error fetching feed", err)
		return
	}

	for _, item := range rssFeed.Channel.Item {
		description := sql.NullString{}
		if item.Description != "" {
			description.String = item.Description
			description.Valid = true
		}

		pubAt, err := parsePubDate(item.PubDate)
		if err != nil {
			log.Printf("Couldnt parse date %v with err %v", item.PubDate, err)
			continue
		}

		// duplicates are skipped by the database (ON CONFLICT DO NOTHING)
		err = db.CreatePost(ctx, database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
			Title:       item.Title,
			Description: description,
			PublishedAt: pubAt,
			Url:         item.Link,
			FeedID:      feed.ID,
		})
		if err != nil {
			log.Println("failed to create post:", err)
		}
	}
	log.Printf("Feed %s collected, %v posts found", feed.Name, len(rssFeed.Channel.Item))
}
