package storage

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	mongoURL = "mongodb://localhost:27017"
	dbName   = "auth"
)

type MongoStore struct {
	DB *mongo.Database
}

func NewMongoStore() *MongoStore {
	return &MongoStore{}
}

func (s *MongoStore) Open() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))

	if err != nil {
		return err
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return err
	}

	s.DB = client.Database(dbName)

	return nil
}

func (s *MongoStore) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s.DB.Client().Disconnect(ctx)
}
