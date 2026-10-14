package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	// maxFeedBytes caps how much of a feed response is read into memory
	maxFeedBytes = 10 << 20
	// feedUserAgent identifies the scraper, some hosts reject Go's default
	feedUserAgent = "rss-aggregator/1.0 (+https://github.com/mubeendevelops/rss-aggregator)"
)

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

// feedHTTPClient fetches feeds. By default it refuses loopback, private and
// link-local destinations, main replaces it when FEED_ALLOW_PRIVATE_ADDRESSES is set.
var feedHTTPClient = newFeedHTTPClient(false)

func urlToFeed(ctx context.Context, url string) (RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return RSSFeed{}, err
	}
	req.Header.Set("User-Agent", feedUserAgent)

	res, err := feedHTTPClient.Do(req)
	if err != nil {
		return RSSFeed{}, err
	}

	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Println("Error closing feed response body", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return RSSFeed{}, fmt.Errorf("unexpected status %d", res.StatusCode)
	}

	// read one byte past the cap so an oversized body can be told apart from one that fits
	data, err := io.ReadAll(io.LimitReader(res.Body, maxFeedBytes+1))
	if err != nil {
		return RSSFeed{}, err
	}
	if len(data) > maxFeedBytes {
		return RSSFeed{}, fmt.Errorf("feed is larger than %d bytes", maxFeedBytes)
	}

	rssFeed := RSSFeed{}

	err = xml.Unmarshal(data, &rssFeed)

	if err != nil {
		return RSSFeed{}, err
	}

	return rssFeed, nil
}

// pubDateLayouts are tried in order. Zone abbreviations other than GMT, UT and
// UTC carry no offset in Go, so such dates are read as if they were UTC.
var pubDateLayouts = []string{
	time.RFC1123Z,
	time.RFC1123,
	time.RFC822Z,
	time.RFC822,
	time.RFC3339,
	"Mon, 2 Jan 2006 15:04:05 -0700",
	"Mon, 2 Jan 2006 15:04:05 MST",
	"Mon, 2 Jan 2006 15:04:05 UT",
	"Mon, 2 Jan 2006 15:04 -0700",
	"Mon, 2 Jan 2006 15:04 MST",
	"2 Jan 2006 15:04:05 -0700",
	"2 Jan 2006 15:04:05 MST",
	"2 Jan 06 15:04 -0700",
	"2 Jan 06 15:04 MST",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// parsePubDate parses the date formats commonly found in feeds and returns the time in UTC
func parsePubDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	for _, layout := range pubDateLayouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format %q", value)
}
