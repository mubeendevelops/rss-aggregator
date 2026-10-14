package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/mubeencodes/rss-aggregator/internal/database"
)

type apiConfig struct {
	DB *database.Queries
}

func main() {

	// .env is optional so the server can run from real environment variables
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(".env"); err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	portString := os.Getenv("PORT")
	if portString == "" {
		log.Fatal("PORT is not found in the environment or the .env file")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL is not found in the environment or the .env file")
	}

	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Cant open database: ", redactDBURL(err, dbURL))
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(10)
	conn.SetConnMaxLifetime(30 * time.Minute)

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = conn.PingContext(pingCtx)
	cancel()
	if err != nil {
		log.Fatal("Cant connect to database, check DB_URL: ", redactDBURL(err, dbURL))
	}

	db := database.New(conn)

	apiCfg := apiConfig{
		DB: db,
	}

	seconds := 30
	go startScraping(db, 10, time.Duration(seconds)*time.Second)

	fmt.Println("Port:", portString)

	router := chi.NewRouter()

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	v1Router := chi.NewRouter()
	v1Router.Get("/healthz", handlerReadiness)
	v1Router.Get("/err", handlerErr)

	v1Router.Post("/users", apiCfg.handlerCreateUser)
	v1Router.Get("/users", apiCfg.middlewareAuth(apiCfg.handlerGetUser))

	v1Router.Post("/feeds", apiCfg.middlewareAuth(apiCfg.handlerCreateFeed))
	v1Router.Get("/feeds", apiCfg.handlerGetFeeds)

	v1Router.Get("/posts", apiCfg.middlewareAuth(apiCfg.handlerGetPostsForUser))

	v1Router.Post("/feed_follows", apiCfg.middlewareAuth(apiCfg.handlerCreateFeedFollow))
	v1Router.Get("/feed_follows", apiCfg.middlewareAuth(apiCfg.handlerGetFeedFollows))
	v1Router.Delete("/feed_follows/{feedFollowID}", apiCfg.middlewareAuth(apiCfg.handlerDeleteFeedFollow))

	router.Mount("/v1", v1Router)

	srv := &http.Server{
		Handler: router,
		Addr:    ":" + portString,
	}

	log.Printf("Server starting on port %v", portString)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}

}

// redactDBURL keeps the connection string, which holds the password, out of error logs
func redactDBURL(err error, dbURL string) string {
	return strings.ReplaceAll(err.Error(), dbURL, "[DB_URL]")
}
