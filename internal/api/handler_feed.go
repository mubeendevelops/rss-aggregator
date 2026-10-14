package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mubeencodes/rss-aggregator/internal/database"
)

func (apiCfg *Config) handlerCreateFeed(w http.ResponseWriter, r *http.Request, user database.User) {
	type parameters struct {
		Name string `json:"name"`
		URL  string `json:"url"`
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

	if err := validateFeedURL(params.URL); err != nil {
		respondWithError(w, 400, err.Error())
		return
	}

	feed, err := apiCfg.DB.CreateFeed(r.Context(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      params.Name,
		Url:       params.URL,
		UserID:    user.ID,
	})
	if err != nil {
		respondWithDBError(w, "create feed", err)
		return
	}

	respondWithJSON(w, 201, databaseFeedToFeed(feed))
}

func (apiCfg *Config) handlerGetFeeds(w http.ResponseWriter, r *http.Request) {

	feeds, err := apiCfg.DB.GetFeeds(r.Context())
	if err != nil {
		respondWithDBError(w, "get feeds", err)
		return
	}

	respondWithJSON(w, 200, databaseFeedsToFeeds(feeds))
}
