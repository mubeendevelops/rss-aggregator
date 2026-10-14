package main

import (
	"encoding/json"
	"log"
	"net/http"
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

func (apiCfg *apiConfig) handlerGetPostsForUser(w http.ResponseWriter, r *http.Request, user database.User) {
	posts, err := apiCfg.DB.GetPostsForUser(r.Context(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  10,
	})
	if err != nil {
		respondWithDBError(w, "get posts", err)
		return
	}

	respondWithJSON(w, 200, databasePostsToPosts(posts))
}
