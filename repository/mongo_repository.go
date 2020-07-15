package repository

import (
	"context"
	"log"

	"github.com/flaambe/authservice/models"
	"github.com/flaambe/authservice/storage"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type userRepository struct {
	store *storage.MongoStore
}

func NewUserRepository(mongoStorage *storage.MongoStore) UserRepository {
	return &userRepository{
		store: mongoStorage,
	}
}

func (r *userRepository) FindOneAndUpdate(f *models.User) *models.User {
	if err := r.store.Open(); err != nil {
		log.Fatal(err)
	}

	defer r.store.Close()

	upsert := true
	after := options.After
	opt := options.FindOneAndUpdateOptions{
		ReturnDocument: &after,
		Upsert:         &upsert,
	}

	update := bson.D{
		{"$set", bson.D{{"guid", f.GUID}}},
	}

	result := r.store.DB.Collection("Users").FindOneAndUpdate(context.Background(), f, update, &opt)

	user := models.User{}

	result.Decode(&user)

	return &user
}
