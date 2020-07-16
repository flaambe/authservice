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
	bearerSchema string = "Bearer "
)

type AuthUsecase interface {
	Auth(body views.AuthRequest) (views.AuthResponse, error)
	Refresh(body views.RefreshTokenRequest) (views.AuthResponse, error)
	Delete(body views.DeleteTokenRequest) error
	DeleteAll(body views.DeleteAllTokensRequest) error
}

type AuthHandler struct {
	authUsecase AuthUsecase
}

func NewAuthHandler(au AuthUsecase) *AuthHandler {
	return &AuthHandler{
		authUsecase: au,
	}
}

func (h *AuthHandler) Auth(w http.ResponseWriter, r *http.Request) {
	var body views.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if len(body.GUID) == 0 {
		respondWithError(w, http.StatusBadRequest, "GUID not found")
		return
	}

	response, err := h.authUsecase.Auth(body)
	if err != nil {
		var requestError *errs.RequestError
		if errors.As(err, &requestError) {
			respondWithError(w, requestError.Status, requestError.Message)
			return
		}

		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, response)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body views.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	bearer, err := getBearer(r)
	if err != nil {
		respondWithError(w, http.StatusForbidden, err.Error())
		return
	}
	body.AccessToken = bearer

	if len(body.RefreshToken) == 0 {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	response, err := h.authUsecase.Refresh(body)
	if err != nil {
		var requestError *errs.RequestError
		if errors.As(err, &requestError) {
			respondWithError(w, requestError.Status, requestError.Message)
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, response)
}

func (h *AuthHandler) Delete(w http.ResponseWriter, r *http.Request) {
	var body views.DeleteTokenRequest

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	bearer, err := getBearer(r)
	if err != nil {
		respondWithError(w, http.StatusForbidden, err.Error())
		return
	}
	body.AccessToken = bearer

	if len(body.RefreshToken) == 0 {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	err = h.authUsecase.Delete(body)
	if err != nil {
		var requestError *errs.RequestError
		if errors.As(err, &requestError) {
			respondWithError(w, requestError.Status, requestError.Message)
			return
		}

		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusNoContent, views.EmptyResponse{})
}

func (h *AuthHandler) DeleteAll(w http.ResponseWriter, r *http.Request) {
	var body views.DeleteAllTokensRequest

	bearer, err := getBearer(r)
	if err != nil {
		respondWithError(w, http.StatusForbidden, err.Error())
		return
	}
	body.AccessToken = bearer

	err = h.authUsecase.DeleteAll(body)
	if err != nil {
		var requestErr *errs.RequestError
		if errors.As(err, &requestErr) {
			respondWithError(w, requestErr.Status, requestErr.Message)
			return
		}

		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusNoContent, views.EmptyResponse{})
}

// helpers
func getBearer(req *http.Request) (string, error) {
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("Authorization header required")
	}

	if !strings.HasPrefix(authHeader, bearerSchema) {
		return "", errors.New("Authorization requires Bearer scheme")
	}

	return authHeader[len(bearerSchema):], nil
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
