package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/flaambe/authservice/errs"
	"github.com/flaambe/authservice/views"
)

const (
	bearerSchema string = "Bearer "
)

type AuthUsecase interface {
	Auth(guid string) (views.AuthResponse, error)
	RefreshToken(accessToken, refreshToken string) (views.RefreshResponse, error)
	DeleteToken(accessToken, refreshToken string) error
	DeleteAllTokens(accessToken string) error
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
		if err == io.EOF {
			respondWithError(w, http.StatusBadRequest, "request body is empty")
			return
		}

		respondWithError(w, http.StatusBadRequest, err.Error())

		return
	}

	if body.GUID == "" {
		respondWithError(w, http.StatusBadRequest, "GUID not found")
		return
	}

	response, err := h.authUsecase.Auth(body.GUID)
	if err != nil {
		var requestError *errs.RequestError
		if errors.As(err, &requestError) {
			if requestError.Err != nil {
				log.Println(requestError.Err.Error())
			}

			respondWithError(w, requestError.Status, requestError.Message)

			return
		}

		respondWithError(w, http.StatusInternalServerError, err.Error())

		return
	}

	respondWithJSON(w, http.StatusOK, response)
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var body views.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		if err == io.EOF {
			respondWithError(w, http.StatusBadRequest, "request body is empty")
			return
		}

		respondWithError(w, http.StatusBadRequest, err.Error())

		return
	}

	accessToken, err := getBearer(r)
	if err != nil {
		respondWithError(w, http.StatusForbidden, err.Error())
		return
	}

	if body.RefreshToken == "" {
		respondWithError(w, http.StatusBadRequest, "refresh token is missing")
		return
	}

	response, err := h.authUsecase.RefreshToken(accessToken, body.RefreshToken)
	if err != nil {
		var requestError *errs.RequestError
		if errors.As(err, &requestError) {
			if requestError.Err != nil {
				log.Println(requestError.Err.Error())
			}

			respondWithError(w, requestError.Status, requestError.Message)

			return
		}

		respondWithError(w, http.StatusInternalServerError, err.Error())

		return
	}

	respondWithJSON(w, http.StatusOK, response)
}

func (h *AuthHandler) DeleteToken(w http.ResponseWriter, r *http.Request) {
	var body views.DeleteTokenRequest

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		if err == io.EOF {
			respondWithError(w, http.StatusBadRequest, "request body is empty")
			return
		}

		respondWithError(w, http.StatusBadRequest, err.Error())

		return
	}

	accessToken, err := getBearer(r)
	if err != nil {
		respondWithError(w, http.StatusForbidden, err.Error())
		return
	}

	if body.RefreshToken == "" {
		respondWithError(w, http.StatusBadRequest, "refresh token is missing")
		return
	}

	err = h.authUsecase.DeleteToken(accessToken, body.RefreshToken)
	if err != nil {
		var requestError *errs.RequestError
		if errors.As(err, &requestError) {
			if requestError.Err != nil {
				log.Println(requestError.Err.Error())
			}

			respondWithError(w, requestError.Status, requestError.Message)

			return
		}

		respondWithError(w, http.StatusInternalServerError, err.Error())

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) DeleteAllTokens(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getBearer(r)
	if err != nil {
		respondWithError(w, http.StatusForbidden, err.Error())
		return
	}

	err = h.authUsecase.DeleteAllTokens(accessToken)
	if err != nil {
		var requestErr *errs.RequestError
		if errors.As(err, &requestErr) {
			if requestErr.Err != nil {
				log.Println(requestErr.Err.Error())
			}

			respondWithError(w, requestErr.Status, requestErr.Message)

			return
		}

		respondWithError(w, http.StatusInternalServerError, err.Error())

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// helpers
func getBearer(req *http.Request) (string, error) {
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return "", errs.New(http.StatusForbidden, "authorization header required", nil)
	}

	if !strings.HasPrefix(authHeader, bearerSchema) {
		return "", errs.New(http.StatusForbidden, "authorization requires Bearer scheme", nil)
	}

	return strings.ReplaceAll(authHeader, bearerSchema, ""), nil
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, views.ErrorResponse{ErrorMessage: message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
