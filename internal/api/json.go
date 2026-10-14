package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

// maxBodyBytes caps the size of a request body
const maxBodyBytes = 1 << 20

// middlewareBodyLimit makes reads past maxBodyBytes fail instead of buffering them
func middlewareBodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		next.ServeHTTP(w, r)
	})
}

// respondWithDecodeError answers a failed JSON body decode
func respondWithDecodeError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		respondWithError(w, 413, fmt.Sprintf("Request body is larger than %d bytes", maxBodyBytes))
		return
	}
	respondWithError(w, 400, fmt.Sprintf("Error parsing JSON: %v", err))
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	if code > 499 {
		log.Println("Responding with 5XX error:", msg)
	}

	type errResponse struct {
		Error string `json:"error"`
	}

	respondWithJSON(w, code, errResponse{
		Error: msg,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal JSON response: %v", err)
		code = 500
		data = []byte(`{"error":"Internal server error"}`)
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(code)
	if _, err := w.Write(data); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}
