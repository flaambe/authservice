package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/flaambe/authservice/errs"

	"github.com/flaambe/authservice/views"
)

const (
	BEARER_SCHEMA string = "Bearer "
)

type AuthUsecase interface {
	Auth(body views.AccessTokenRequest) (views.TokensResponse, error)
	Delete(bearer string) (views.TokensResponse, error)
	DeleteAll(bearer string) (views.TokensResponse, error)
}

type AuthHandler struct {
	authUsecase AuthUsecase
}

func NewAuthHandler(a AuthUsecase) *AuthHandler {
	return &AuthHandler{
		authUsecase: a,
	}
}

func GetBearer(req *http.Request) (string, error) {
	// Grab the raw Authoirzation header
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("Authorization header required")
	}

	if !strings.HasPrefix(authHeader, BEARER_SCHEMA) {
		return "", errors.New("Authorization requires Basic/Bearer scheme")
	}

	return authHeader[len(BEARER_SCHEMA):], nil
}

func (a *AuthHandler) Auth(w http.ResponseWriter, r *http.Request) {
	var body views.AccessTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if len(body.GUID) == 0 {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	response, err := a.authUsecase.Auth(body)
	if err != nil {
		var requestErr errs.RequestError
		if errors.As(err, &requestErr) {
			respondWithError(w, requestErr.Status, requestErr.Message)
			return
		}

		respondWithError(w, http.StatusInternalServerError, err.Error())
	}

	respondWithJSON(w, http.StatusOK, response)
}

func (a *AuthHandler) Delete(w http.ResponseWriter, r *http.Request) {
	var body views.AccessTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondWithError(w, http.StatusForbidden, "Authorization header required")
		return
	}

	if !strings.HasPrefix(authHeader, BEARER_SCHEMA) {
		respondWithError(w, http.StatusForbidden, "Authorization requires Bearer scheme")
		return
	}

	bearer, err := GetBearer(r)
	if err != nil {
		respondWithError(w, http.StatusForbidden, err.Error())
		return
	}

	response, err := a.authUsecase.Delete(bearer)
	if err != nil {
		var requestErr *errs.RequestError
		if errors.As(err, &requestErr) {
			respondWithError(w, requestErr.Status, requestErr.Message)
			return
		}

		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, response)
}

func (a *AuthHandler) DeleteAll(w http.ResponseWriter, r *http.Request) {
	var body views.AccessTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondWithError(w, http.StatusForbidden, "Authorization header required")
		return
	}

	if !strings.HasPrefix(authHeader, BEARER_SCHEMA) {
		respondWithError(w, http.StatusForbidden, "Authorization requires Bearer scheme")
		return
	}

	bearer, err := GetBearer(r)
	if err != nil {
		respondWithError(w, http.StatusForbidden, err.Error())
		return
	}

	response, err := a.authUsecase.DeleteAll(bearer)
	if err != nil {
		var requestErr *errs.RequestError
		if errors.As(err, &requestErr) {
			respondWithError(w, requestErr.Status, requestErr.Message)
			return
		}

		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

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
