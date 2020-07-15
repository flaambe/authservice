package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AuthToken struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	UserID       primitive.ObjectID `bson:"user_id"`
	TokenType    string             `bson:"token_type"`
	AccessToken  string             `bson:"acess_token"`
	RefreshToken string             `bson:"refresh_token"`
	ExpiresAt    primitive.DateTime `bson:"expires_at"`
}
