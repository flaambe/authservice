package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/flaambe/authservice/handlers"
	"github.com/flaambe/authservice/usecase"
)

func Connect() (*mongo.Database, error) {
	MONGOURI := os.Getenv("MONGODB_URI")
	DBNAME := os.Getenv("DBNAME")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(MONGOURI))

	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}

	log.Println("Database connected")

	db := client.Database(DBNAME)

	return db, nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
		return
	}

	db, err := Connect()
	if err != nil {
		log.Fatal(err)
		return
	}

	defer db.Client().Disconnect(context.Background())

	authUsecase := usecase.NewAuthUsecase(db)
	handler := handlers.NewAuthHandler(authUsecase)

	http.HandleFunc("/auth", handler.Auth)
	http.HandleFunc("/deleteAll", handler.DeleteAll)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
