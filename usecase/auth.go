package usecase

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/flaambe/authservice/errs"

	"github.com/dgrijalva/jwt-go"
	"github.com/flaambe/authservice/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	"github.com/flaambe/authservice/views"
)

const (
	BEARER_SCHEMA string = "Bearer "
)

type EmptyResponse struct {
}

type AuthUsecase struct {
	db *mongo.Database
}

func NewAuthUsecase(db *mongo.Database) *AuthUsecase {
	return &AuthUsecase{db}
}

func CreateAccessToken(userId string) (string, error) {
	var err error
	//Creating Access Token
	atClaims := jwt.MapClaims{}
	atClaims["authorized"] = true
	atClaims["user_id"] = userId
	atClaims["exp"] = time.Now().Add(time.Minute * 10).Unix()
	at := jwt.NewWithClaims(jwt.SigningMethodHS512, atClaims)
	token, err := at.SignedString([]byte(os.Getenv("ACCESS_SECRET")))
	if err != nil {
		return "", err
	}
	return token, nil
}

func CreateRefreshToken(userId string) (string, error) {
	var err error
	//Creating Refresh Token
	atClaims := jwt.MapClaims{}
	atClaims["authorized"] = true
	atClaims["user_id"] = userId
	atClaims["exp"] = time.Now().Add(time.Minute * 60).Unix()
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	token, err := at.SignedString([]byte((os.Getenv("REFRESH_SECRET"))))
	if err != nil {
		return "", err
	}
	return token, nil
}

func HashToken(token string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(token), 14)
	return string(bytes), err
}

func CheckTokenHash(token, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(token))
	return err == nil
}

func (a *AuthUsecase) Auth(body views.AccessTokenRequest) (views.TokensResponse, error) {
	var tokensResponse views.TokensResponse

	users := a.db.Collection("users")
	tokens := a.db.Collection("tokens")

	err := a.db.Client().UseSession(context.Background(), func(sctx mongo.SessionContext) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return errs.New(500, "Server internal error")
		}

		upsert := true
		after := options.After
		opt := options.FindOneAndUpdateOptions{
			ReturnDocument: &after,
			Upsert:         &upsert,
		}

		userFilter := bson.M{"guid": body.GUID}
		userUpdate := bson.D{
			{"$set", bson.D{{"guid", body.GUID}}},
		}

		userResult := users.FindOneAndUpdate(sctx, userFilter, userUpdate, &opt)
		if userResult.Err() != nil {
			sctx.AbortTransaction(sctx)
			log.Println("caught exception during transaction, aborting.")
			return errs.New(404, "User not founded")
		}

		user := models.User{}
		userResult.Decode(&user)

		accessToken, err := CreateAccessToken(user.GUID)
		if err != nil {
			return errs.New(500, "Server internal error")
		}

		refreshToken, err := CreateRefreshToken(user.GUID)
		if err != nil {
			return errs.New(500, "Server internal error")
		}
		hashedRefreshToken, err := HashToken(refreshToken)
		if err != nil {
			return errs.New(500, "Server internal error")
		}

		tokenDocument := models.AuthToken{
			UserID:           user.ID,
			AccessToken:      accessToken,
			RefreshToken:     hashedRefreshToken,
			TokenType:        "Bearer",
			AccessExpiresAt:  primitive.NewDateTimeFromTime(time.Now().Add(time.Minute * 10)),
			RefreshExpiresAt: primitive.NewDateTimeFromTime(time.Now().Add(time.Minute * 60)),
		}

		_, err = tokens.InsertOne(sctx, tokenDocument)
		if err != nil {
			sctx.AbortTransaction(sctx)
			log.Println("caught exception during transaction, aborting.")
			return errs.New(500, "Server internal error")
		}

		tokensResponse.AccessToken = accessToken
		tokensResponse.TokenType = tokenDocument.TokenType
		tokensResponse.ExpiresIn = int(time.Duration(10) * time.Minute / time.Second)
		tokensResponse.RefreshToken = refreshToken

		err = sctx.CommitTransaction(sctx)
		if err != nil {
			return errs.New(500, "Server internal error")
		}

		return nil
	})

	if err != nil {
		log.Fatal(err)
		return views.TokensResponse{}, errs.New(500, "Internal Server Error")
	}

	return tokensResponse, nil
}

func (a *AuthUsecase) Delete(bearer string) (views.TokensResponse, error) {

	tokens := a.db.Collection("tokens")

	err := a.db.Client().UseSession(context.Background(), func(sctx mongo.SessionContext) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return errs.New(500, "Server internal error")
		}

		token := models.AuthToken{}
		accessTokenResult := tokens.FindOne(sctx, bson.D{{"aсcess_token", bearer}})
		err = accessTokenResult.Decode(&token)
		if err != nil {
			return errs.New(403, "Access forbidden")
		}

		if token.AccessExpiresAt.Time().Before(time.Now()) {
			return errs.New(403, "Access forbidden")
		}

		// Delete token
		_, err = tokens.DeleteMany(sctx, bson.D{{"access_token", token.AccessToken}})
		if err != nil {
			sctx.AbortTransaction(sctx)
			log.Println("caught exception during transaction, aborting.")
			return errs.New(500, "Server internal error")
		}

		err = sctx.CommitTransaction(sctx)
		if err != nil {
			return errs.New(500, "Server internal error")
		}
		return nil
	})

	if err != nil {
		return views.TokensResponse{}, err
	}

	return views.TokensResponse{}, nil
}

func (a *AuthUsecase) DeleteAll(bearer string) (views.TokensResponse, error) {

	tokens := a.db.Collection("tokens")

	err := a.db.Client().UseSession(context.Background(), func(sctx mongo.SessionContext) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return errs.New(500, "Server internal error")
		}

		token := models.AuthToken{}
		accessTokenResult := tokens.FindOne(sctx, bson.D{{"aсcess_token", bearer}})
		err = accessTokenResult.Decode(&token)
		if err != nil {
			return errs.New(403, "Access forbidden")
		}

		if token.AccessExpiresAt.Time().Before(time.Now()) {
			return errs.New(403, "Access forbidden")
		}

		// Delete all tokens for user
		_, err = tokens.DeleteMany(sctx, bson.D{{"user_id", token.UserID}})
		if err != nil {
			sctx.AbortTransaction(sctx)
			log.Println("caught exception during transaction, aborting.")
			return errs.New(500, "Server internal error")
		}

		err = sctx.CommitTransaction(sctx)
		if err != nil {
			return errs.New(500, "Server internal error")
		}
		return nil
	})

	if err != nil {
		return views.TokensResponse{}, err
	}

	return views.TokensResponse{}, nil
}
