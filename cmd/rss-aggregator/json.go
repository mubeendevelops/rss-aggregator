package main

import (
	"encoding/json"
	"log"
	"net/http"
)

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
