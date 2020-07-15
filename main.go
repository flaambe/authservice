package main

import (
	"log"
	"net/http"

	"github.com/flaambe/authservice/storage"

	"github.com/flaambe/authservice/handlers"
	"github.com/flaambe/authservice/repository"
	"github.com/flaambe/authservice/usecase"
)

func main() {
	store := storage.NewMongoStore()
	userRepo := repository.NewUserRepository(store)
	userUsecase := usecase.NewUserUsecase(userRepo)
	handler := handlers.NewUserHandler(userUsecase)
	http.HandleFunc("/getToken", handler.GetToken)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
