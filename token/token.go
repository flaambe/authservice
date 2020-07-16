package token

import (
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

const (
	AccessTokenDuration  time.Duration = 10
	RefreshTokenDuration time.Duration = 60
)

func CreateAccessToken(userGuid string) (string, error) {
	var err error

	atClaims := jwt.MapClaims{}
	atClaims["authorized"] = true
	atClaims["user_id"] = userGuid
	atClaims["exp"] = time.Now().Add(time.Minute * AccessTokenDuration).Unix()
	at := jwt.NewWithClaims(jwt.SigningMethodHS512, atClaims)

	token, err := at.SignedString([]byte(os.Getenv("ACCESS_SECRET")))

	if err != nil {
		return "", err
	}

	return token, nil
}

func CreateRefreshToken(userGuid string) (string, error) {
	var err error

	atClaims := jwt.MapClaims{}
	atClaims["authorized"] = true
	atClaims["user_id"] = userGuid
	atClaims["exp"] = time.Now().Add(time.Minute * RefreshTokenDuration).Unix()
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)

	token, err := at.SignedString([]byte((os.Getenv("REFRESH_SECRET"))))

	if err != nil {
		return "", err
	}

	return token, nil
}

func HashToken(token string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(token), 14)

	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func CheckTokenHash(token, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(token))
	return err == nil
}
