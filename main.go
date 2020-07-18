package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

func ensureindexes(db *mongo.Database) {
	uniqUserIndex := mongo.IndexModel{
		Keys:    bson.M{"guid": 1},
		Options: options.Index().SetUnique(true),
	}

	_, err := db.Collection("users").Indexes().CreateOne(context.TODO(), uniqUserIndex)

	if err != nil {
		log.Fatal(err)
	}

	expireTokenIndex := mongo.IndexModel{
		Keys:    bson.M{"refresh_expires_at": 1},
		Options: options.Index().SetExpireAfterSeconds(0),
	}

	_, err = db.Collection("tokens").Indexes().CreateOne(context.TODO(), expireTokenIndex)

	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	db, err := connect()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer db.Client().Disconnect(ctx)

	ensureindexes(db)

	authUsecase := usecase.NewAuthUsecase(db)
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
		log.Printf("error shutting down server %s", err)
	} else {
		log.Println("Server gracefully stopped")
	}
}
