package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flaambe/authservice/handlers"
	"github.com/flaambe/authservice/mongoconf"
	"github.com/flaambe/authservice/usecase"

	"github.com/gorilla/mux"
)

func main() {
	dbConfig := mongoconf.NewConfig()
	if err := dbConfig.Open(os.Getenv("MONGODB_URI"), os.Getenv("DBNAME")); err != nil {
		log.Fatal(err)
	}

	if err := dbConfig.EnsureIndexes(); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer dbConfig.DB.Client().Disconnect(ctx)

	authUsecase := usecase.NewAuthUsecase(dbConfig.DB)
	authHandler := handlers.NewAuthHandler(authUsecase)

	router := mux.NewRouter()
	router.HandleFunc("/auth", authHandler.Auth).Methods("POST")
	router.HandleFunc("/refreshToken", authHandler.RefreshToken).Methods("POST")
	router.HandleFunc("/deleteToken", authHandler.DeleteToken).Methods("POST")
	router.HandleFunc("/deleteAllTokens", authHandler.DeleteAllTokens).Methods("POST")

	srv := &http.Server{
		Addr:         getPort(),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      router,
	}

	go func() {
		panic(srv.ListenAndServe())
	}()

	// Create channel for shutdown signals.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	signal.Notify(stop, syscall.SIGTERM)

	//Recieve shutdown signals.
	<-stop

	ctx, cancel = context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down server %s", err)
	} else {
		log.Println("Server gracefully stopped")
	}
}

func getPort() string {
	p := os.Getenv("PORT")
	if p != "" {
		return ":" + p
	}

	return ":8080"
}
