package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/flaambe/authservice/views"
)

type AuthUsecase interface {
	Auth(body views.AccessTokenRequest) (views.TokensResponse, error)
}

type AuthHandler struct {
	authUsecase AuthUsecase
}

func NewAuthHandler(a AuthUsecase) *AuthHandler {
	return &AuthHandler{
		authUsecase: a,
	}
}

func (a *AuthHandler) Auth(w http.ResponseWriter, r *http.Request) {
	var body views.AccessTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	response, _ := a.authUsecase.Auth(body)

	respondWithJSON(w, http.StatusOK, response)
}

// helpers
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
