package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/flaambe/authservice/handlers"
	"github.com/flaambe/authservice/usecase"
)

func getPort() string {
	p := os.Getenv("PORT")
	if p != "" {
		return ":" + p
	}

	return ":8080"
}

func connect() (*mongo.Database, error) {
	mongoURI := os.Getenv("MONGODB_URI")
	dbName := os.Getenv("DBNAME")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))

	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}

	log.Println("Database connected")

	db := client.Database(dbName)

	return db, nil
}

func main() {
	db, err := connect()
	if err != nil {
		log.Fatal(err)
	}

	uniqueOpt := options.Index().SetUnique(true)
	uniqUserIndex := mongo.IndexModel{
		Keys:    bson.M{"guid": 1},
		Options: uniqueOpt,
	}
	db.Collection("users").Indexes().CreateOne(context.TODO(), uniqUserIndex)

	expireOpt := options.Index().SetExpireAfterSeconds(0)
	expireTokenIndex := mongo.IndexModel{
		Keys:    bson.M{"refresh_expires_at": 1},
		Options: expireOpt,
	}
	db.Collection("tokens").Indexes().CreateOne(context.TODO(), expireTokenIndex)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer db.Client().Disconnect(ctx)

	authUsecase := usecase.NewAuthUsecase(db)
	authHandler := handlers.NewAuthHandler(authUsecase)

	r := mux.NewRouter()
	r.HandleFunc("/auth", authHandler.Auth).Methods("POST")
	r.HandleFunc("/refreshToken", authHandler.RefreshToken).Methods("POST")
	r.HandleFunc("/deleteToken", authHandler.DeleteToken).Methods("POST")
	r.HandleFunc("/deleteAllTokens", authHandler.DeleteAllTokens).Methods("POST")

	srv := &http.Server{
		Addr:         getPort(),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Println("shutting down")
}
