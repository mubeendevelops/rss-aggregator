package scraper

import (
	"fmt"
	"strconv"
	"time"
)

// Config holds the scraper settings read from the environment
type Config struct {
	// Interval is the time between scrape cycles (SCRAPE_INTERVAL)
	Interval time.Duration
	// Concurrency is the most feeds fetched in one cycle (SCRAPE_CONCURRENCY)
	Concurrency int
	// MinRefetchInterval is how long a feed is left alone after a fetch (SCRAPE_MIN_REFETCH_INTERVAL)
	MinRefetchInterval time.Duration
}

var defaultConfig = Config{
	Interval:           30 * time.Second,
	Concurrency:        10,
	MinRefetchInterval: 5 * time.Minute,
}

const maxScrapeConcurrency = 100

// LoadConfig reads the scraper settings through getenv, which is os.Getenv outside tests.
// Unset variables keep their defaults, invalid ones are an error.
func LoadConfig(getenv func(string) string) (Config, error) {
	cfg := defaultConfig

	if value := getenv("SCRAPE_INTERVAL"); value != "" {
		interval, err := time.ParseDuration(value)
		if err != nil || interval <= 0 {
			return Config{}, fmt.Errorf("SCRAPE_INTERVAL must be a positive duration such as 30s or 2m, got %q", value)
		}
		cfg.Interval = interval
	}

	if value := getenv("SCRAPE_CONCURRENCY"); value != "" {
		concurrency, err := strconv.Atoi(value)
		if err != nil || concurrency < 1 || concurrency > maxScrapeConcurrency {
			return Config{}, fmt.Errorf("SCRAPE_CONCURRENCY must be a whole number from 1 to %d, got %q", maxScrapeConcurrency, value)
		}
		cfg.Concurrency = concurrency
	}

	if value := getenv("SCRAPE_MIN_REFETCH_INTERVAL"); value != "" {
		minRefetch, err := time.ParseDuration(value)
		if err != nil || minRefetch < 0 {
			return Config{}, fmt.Errorf("SCRAPE_MIN_REFETCH_INTERVAL must be a duration such as 5m, or 0 to disable it, got %q", value)
		}
		cfg.MinRefetchInterval = minRefetch
	}

	return cfg, nil
}
