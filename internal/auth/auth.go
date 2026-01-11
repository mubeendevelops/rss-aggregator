package auth

import (
	"errors"
	"net/http"
	"strings"
)

// this function gets API key from headers of the http request
// Example:
// Authorization: ApiKey {actual apikey here}
func GetAPIKey(headers http.Header) (string, error) {
	value := headers.Get("Authorization")
	if value == "" {
		return "", errors.New("No authentication info found")

	}

	values := strings.Split(value, " ")
	if len(values) != 2 {
		return "", errors.New("Malformed auth header")

	}

	if values[0] != "ApiKey" {
		return "", errors.New("Malformed first part of auth header")
	}

	return values[1], nil
}
