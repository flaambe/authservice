package config

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Config struct {
	DB *mongo.Database
}

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) Open(mongoURI, dbName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}

	c.DB = client.Database(dbName)

	if err != nil {
		log.Fatal(err)
	}
}

func (c *Config) EnsureIndexes() {
	uniqUserIndex := mongo.IndexModel{
		Keys:    bson.M{"guid": 1},
		Options: options.Index().SetUnique(true),
	}

	_, err := c.DB.Collection("users").Indexes().CreateOne(context.TODO(), uniqUserIndex)

	if err != nil {
		log.Fatal(err)
	}

	expireTokenIndex := mongo.IndexModel{
		Keys:    bson.M{"refresh_expires_at": 1},
		Options: options.Index().SetExpireAfterSeconds(0),
	}

	_, err = c.DB.Collection("tokens").Indexes().CreateOne(context.TODO(), expireTokenIndex)

	if err != nil {
		log.Fatal(err)
	}
}
