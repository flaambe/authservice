package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/flaambe/authservice/usecase"
	"github.com/flaambe/authservice/views"
)

type UserHandler struct {
	userUsecase usecase.UserUsecase
}

func NewUserHandler(u usecase.UserUsecase) *UserHandler {
	return &UserHandler{
		userUsecase: u,
	}
}

func (h *UserHandler) GetToken(w http.ResponseWriter, r *http.Request) {
	var body views.AccessTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	response, _ := h.userUsecase.GetToken(body)

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
