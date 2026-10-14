package main

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// validateName rejects empty and whitespace-only values
func validateName(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

// validateFeedURL accepts absolute http and https URLs only
func validateFeedURL(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("url is required")
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return errors.New("url is not a valid URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("url must start with http:// or https://")
	}
	if parsed.Hostname() == "" {
		return errors.New("url must include a host")
	}
	return nil
}
