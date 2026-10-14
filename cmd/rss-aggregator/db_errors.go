package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/lib/pq"
)

const (
	pqUniqueViolation     = "23505"
	pqForeignKeyViolation = "23503"
)

// dbErrorToResponse maps a database error to a status code and a message
// that is safe to show to clients. The raw driver text is never returned.
func dbErrorToResponse(action string, err error) (int, string) {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case pqUniqueViolation:
			return 409, fmt.Sprintf("Couldnt %s: already exists", action)
		case pqForeignKeyViolation:
			return 404, fmt.Sprintf("Couldnt %s: referenced record not found", action)
		}
	}
	return 500, "Internal server error"
}

func respondWithDBError(w http.ResponseWriter, action string, err error) {
	code, msg := dbErrorToResponse(action, err)
	if code > 499 {
		log.Printf("Couldnt %s: %v", action, err)
	}
	respondWithError(w, code, msg)
}
