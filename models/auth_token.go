package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AuthToken struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	UserID    primitive.ObjectID `bson:"user_id"`
	TokenType string             `bson:"token_type"`
	Token     string             `bson:"token"`
	ExpiresAt primitive.DateTime `bson:"expires_at"`
	GroupID   primitive.ObjectID `bson:"group_id"`
}

func (c *AuthToken) Set(userId primitive.ObjectID, token string) {
	c.UserID = userId
	c.Token = token
	c.TokenType = "Bearer"
	c.ExpiresAt = primitive.NewDateTimeFromTime(time.Now().Add(time.Minute * 10))
	c.GroupID = primitive.NewObjectID()
}
