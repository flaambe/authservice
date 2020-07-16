package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AuthToken struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"`
	UserID           primitive.ObjectID `bson:"user_id"`
	TokenType        string             `bson:"token_type"`
	AccessToken      string             `bson:"aсcess_token"`
	RefreshToken     string             `bson:"refresh_token"`
	AccessExpiresAt  primitive.DateTime `bson:"access_expires_at"`
	RefreshExpiresAt primitive.DateTime `bson:"refresh_expires_at"`
}
