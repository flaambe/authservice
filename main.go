package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/flaambe/authservice/handlers"
	"github.com/flaambe/authservice/usecase"
)

const (
	MONGODB_URI = "mongodb://mongo1:27017,mongo2:27018,mongo3:27019/?replicaSet=rs0"
	dbName      = "auth"
)

func Connect() (*mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(MONGODB_URI))

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

	db, err := Connect()
	if err != nil {
		log.Fatal(err)
		return
	}

	defer db.Client().Disconnect(context.Background())

	authUsecase := usecase.NewAuthUsecase(db)
	handler := handlers.NewAuthHandler(authUsecase)

	http.HandleFunc("/auth", handler.Auth)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
