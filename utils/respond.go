package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func respondWithError(w http.ResponseWriter, err interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusBadRequest)

	response := struct {
		Error string `json:"error"`
	}{}

	response.Error = fmt.Sprintf("%s", err)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		Respond(w, nil, err)
		return
	}
}

func respondWith(w http.ResponseWriter, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		respondWithError(w, err)
		return
	}
}

// Respond with JSON payload including formatted error response
func Respond(w http.ResponseWriter, response interface{}, err interface{}) {
	if err != nil {
		respondWithError(w, err)
		return
	}
	respondWith(w, response)
}
