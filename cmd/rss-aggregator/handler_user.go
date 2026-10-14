package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mubeencodes/rss-aggregator/internal/auth"
	"github.com/mubeencodes/rss-aggregator/internal/database"
)

func (apiCfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Name string `json:"name"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithDecodeError(w, err)
		return
	}

	if err := validateName("name", params.Name); err != nil {
		respondWithError(w, 400, err.Error())
		return
	}

	apiKey, err := auth.GenerateAPIKey()
	if err != nil {
		log.Printf("Couldnt generate API key: %v", err)
		respondWithError(w, 500, "Internal server error")
		return
	}

	user, err := apiCfg.DB.CreateUser(r.Context(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      params.Name,
		ApiKey:    apiKey,
	})
	if err != nil {
		respondWithDBError(w, "create user", err)
		return
	}

	respondWithJSON(w, 201, databaseUserToUser(user))
}

func (apiCfg *apiConfig) handlerGetUser(w http.ResponseWriter, r *http.Request, user database.User) {
	respondWithJSON(w, 200, databaseUserToUser(user))
}

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

func (apiCfg *apiConfig) handlerGetPostsForUser(w http.ResponseWriter, r *http.Request, user database.User) {
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
