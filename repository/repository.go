package repository

import (
	"github.com/flaambe/authservice/models"
)

type UserRepository interface {
	FindOneAndUpdate(u *models.User) *models.User
}

type AuthTokenRepository interface {
	InsertOne(a *models.AuthToken) error
}
