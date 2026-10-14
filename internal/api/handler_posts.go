package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/mubeencodes/rss-aggregator/internal/database"
)

const (
	defaultPostsLimit = 10
	maxPostsLimit     = 100
)

// parsePagination reads the limit and offset query parameters of GET /v1/posts
func parsePagination(query url.Values) (limit, offset int32, err error) {
	limit = defaultPostsLimit
	if value := query.Get("limit"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil || parsed < 1 || parsed > maxPostsLimit {
			return 0, 0, fmt.Errorf("limit must be a whole number from 1 to %d", maxPostsLimit)
		}
		limit = int32(parsed)
	}

	if value := query.Get("offset"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil || parsed < 0 {
			return 0, 0, errors.New("offset must be a whole number, 0 or greater")
		}
		offset = int32(parsed)
	}

	return limit, offset, nil
}

func (apiCfg *Config) handlerGetPostsForUser(w http.ResponseWriter, r *http.Request, user database.User) {
	limit, offset, err := parsePagination(r.URL.Query())
	if err != nil {
		respondWithError(w, 400, err.Error())
		return
	}

	posts, err := apiCfg.DB.GetPostsForUser(r.Context(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		respondWithDBError(w, "get posts", err)
		return
	}

	respondWithJSON(w, 200, databasePostsToPosts(posts))
}
