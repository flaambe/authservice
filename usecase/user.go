package usecase

import (
	"github.com/flaambe/authservice/models"
	"github.com/flaambe/authservice/repository"

	"github.com/flaambe/authservice/views"
)

type UserUsecase interface {
	GetToken(body views.AccessTokenRequest) (views.RefreshTokenResponse, error)
}

type userUsecase struct {
	userRepo repository.UserRepository
}

func NewUserUsecase(a repository.UserRepository) UserUsecase {
	return &userUsecase{a}
}

func (u *userUsecase) GetToken(body views.AccessTokenRequest) (views.RefreshTokenResponse, error) {
	var refreshTokenResponse views.RefreshTokenResponse

	u.userRepo.FindOneAndUpdate(&models.User{GUID: body.GUID})

	/*
		token := models.AuthToken{}
		token.Set(user.ID, user.ID.String())

		err := s.authTokenRepo.InsertOne(&token)
		if err != nil {
			return views.RefreshTokenResponse{}, err
		}

		refreshTokenResponse.AccessToken = token.Token
		refreshTokenResponse.TokenType = token.TokenType
		refreshTokenResponse.ExpiresIn = int(time.Until(token.ExpiresAt.Time()).Seconds())
		refreshTokenResponse.RefreshToken = token.Token
	*/
	return refreshTokenResponse, nil
}
