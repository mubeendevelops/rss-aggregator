package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/mubeencodes/rss-aggregator/internal/auth"
	"github.com/mubeencodes/rss-aggregator/internal/database"
)

type authedHandler func(http.ResponseWriter, *http.Request, database.User)

func (apiCfg *apiConfig) middlewareAuth(handler authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey, err := auth.GetAPIKey(r.Header)
		if err != nil {
			respondWithError(w, 401, fmt.Sprintf("Auth error: %v", err))
			return
		}

		user, err := apiCfg.DB.GetUserByAPIKey(r.Context(), apiKey)
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, 401, "Auth error: invalid API key")
			return
		}
		if err != nil {
			log.Printf("Couldn't get user: %v", err)
			respondWithError(w, 500, "Internal server error")
			return
		}

		handler(w, r, user)
	}
}
