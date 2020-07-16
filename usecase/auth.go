package usecase

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/flaambe/authservice/errs"
	"github.com/flaambe/authservice/models"
	"github.com/flaambe/authservice/token"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	"github.com/flaambe/authservice/views"
)

type AuthUsecase struct {
	db *mongo.Database
}

func NewAuthUsecase(db *mongo.Database) *AuthUsecase {
	return &AuthUsecase{db}
}

func (a *AuthUsecase) Auth(body views.AuthRequest) (views.AuthResponse, error) {
	var authResponse views.AuthResponse

	users := a.db.Collection("users")
	tokens := a.db.Collection("tokens")

	err := a.db.Client().UseSession(context.Background(), func(sctx mongo.SessionContext) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)

		if err != nil {
			return err
		}

		upsert := true
		after := options.After
		opt := options.FindOneAndUpdateOptions{
			ReturnDocument: &after,
			Upsert:         &upsert,
		}

		userFilter := bson.M{"guid": body.GUID}
		userUpdate := bson.M{"$set": bson.M{"guid": body.GUID}}

		user := models.User{}

		err = users.FindOneAndUpdate(sctx, userFilter, userUpdate, &opt).Decode(&user)
		if err != nil {
			sctx.AbortTransaction(sctx)
			log.Println("caught exception during transaction, aborting.")
			return errs.New(500, "Server internal error")
		}

		accessToken, err := token.CreateAccessToken(user.GUID)
		if err != nil {
			return err
		}

		refreshToken, err := token.CreateRefreshToken(user.GUID)
		if err != nil {
			return err
		}

		hashedRefreshToken, err := token.HashToken(refreshToken)
		if err != nil {
			return err
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
			return err
		}

		err = sctx.CommitTransaction(sctx)
		if err != nil {
			return err
		}

		authResponse = views.AuthResponse{
			AccessToken:  accessToken,
			TokenType:    tokenDocument.TokenType,
			ExpiresIn:    int(time.Duration(token.AccessTokenDuration) * time.Minute / time.Second),
			RefreshToken: refreshToken,
		}

		return nil
	})

	if err != nil {
		return views.AuthResponse{}, errs.New(500, "Internal Server Error")
	}

	return authResponse, nil
}

func (a *AuthUsecase) Refresh(body views.RefreshTokenRequest) (views.AuthResponse, error) {
	var authResponse views.AuthResponse

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

		tokenModel := models.AuthToken{}
		filterToken := bson.M{"access_token": body.AccessToken}
		err = tokens.FindOne(sctx, filterToken).Decode(&tokenModel)
		if err != nil {
			return errs.New(403, "Access forbidden")
		}

		if tokenModel.AccessExpiresAt.Time().Before(time.Now()) {
			return errs.New(403, "Access forbidden")
		}

		if !token.CheckTokenHash(body.RefreshToken, tokenModel.RefreshToken) {
			return errs.New(403, "Access forbidden")
		}

		// Refresh token
		user := models.User{}
		err = users.FindOne(sctx, bson.M{"_id": tokenModel.UserID}).Decode(&user)
		if err != nil {
			return errs.New(500, "Server internal error")
		}

		newAccessToken, err := token.CreateAccessToken(user.GUID)
		if err != nil {
			return errs.New(500, "Server internal error")
		}
		updateToken := bson.M{"$set": bson.M{"access_token": newAccessToken}}
		upsert := true
		after := options.After
		opt := options.FindOneAndUpdateOptions{
			ReturnDocument: &after,
			Upsert:         &upsert,
		}
		err = tokens.FindOneAndUpdate(sctx, filterToken, updateToken, &opt).Decode(&tokenModel)
		if err != nil {
			return errs.New(500, "Server internal error")
		}

		if err != nil {
			sctx.AbortTransaction(sctx)
			log.Println("caught exception during transaction, aborting.")
			return errs.New(500, "Server internal error")
		}

		err = sctx.CommitTransaction(sctx)
		if err != nil {
			return errs.New(500, "Server internal error")
		}

		authResponse = views.AuthResponse{
			AccessToken:  tokenModel.AccessToken,
			TokenType:    tokenModel.TokenType,
			ExpiresIn:    int(time.Duration(token.AccessTokenDuration) * time.Minute / time.Second),
			RefreshToken: body.RefreshToken,
		}

		return nil
	})

	if err != nil {
		var requestError *errs.RequestError
		if errors.As(err, &requestError) {
			return views.AuthResponse{}, errs.New(requestError.Status, requestError.Error())
		}
		return views.AuthResponse{}, errs.New(500, err.Error())
	}

	return authResponse, nil
}

func (a *AuthUsecase) Delete(body views.DeleteTokenRequest) error {

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
		filterToken := bson.M{"access_token": body.AccessToken}
		err = tokens.FindOne(sctx, filterToken).Decode(&token)
		if err != nil {
			return errs.New(403, "Access forbidden")
		}

		if token.AccessExpiresAt.Time().Before(time.Now()) {
			return errs.New(403, "Access forbidden")
		}

		// TODO:
		// Delete refresh token

		if err != nil {
			sctx.AbortTransaction(sctx)
			log.Println("caught exception during transaction, aborting.")
			return errs.New(500, "Server internal error")
		}

		err = sctx.CommitTransaction(sctx)
		if err != nil {
			var requestError *errs.RequestError
			if errors.As(err, &requestError) {
				return errs.New(requestError.Status, requestError.Error())
			}
			return errs.New(500, err.Error())
		}
		return nil
	})

	return err
}

func (a *AuthUsecase) DeleteAll(body views.DeleteAllTokensRequest) error {

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
		filterToken := bson.M{"access_token": body.AccessToken}

		err = tokens.FindOne(sctx, filterToken).Decode(&token)
		if err != nil {
			return errs.New(403, "Access forbidden")
		}

		if token.AccessExpiresAt.Time().Before(time.Now()) {
			return errs.New(403, "Access forbidden")
		}

		// Delete all tokens for user
		_, err = tokens.DeleteMany(sctx, bson.M{"user_id": token.UserID})
		if err != nil {
			sctx.AbortTransaction(sctx)
			log.Println("caught exception during transaction, aborting.")
			return errs.New(500, "Server internal error")
		}

		err = sctx.CommitTransaction(sctx)
		if err != nil {
			var requestError *errs.RequestError
			if errors.As(err, &requestError) {
				return errs.New(requestError.Status, requestError.Error())
			}
			return errs.New(500, err.Error())
		}
		return nil
	})

	return err
}
