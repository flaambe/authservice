package usecase

import (
	"context"
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

func (a *AuthUsecase) Auth(guid string) (views.AuthResponse, error) {
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

		opt := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
		userFilter := bson.M{"guid": guid}
		userUpdate := bson.M{"$set": bson.M{"guid": guid}}

		userValue := models.User{}

		err = users.FindOneAndUpdate(sctx, userFilter, userUpdate, opt).Decode(&userValue)
		if err != nil {
			sctx.AbortTransaction(sctx)
			return errs.New(500, "server internal error", err)
		}

		newAccessToken, err := token.CreateAccessToken(userValue.GUID)
		if err != nil {
			return errs.New(500, "server internal error", err)
		}

		newRefreshToken, err := token.CreateRefreshToken(userValue.GUID)
		if err != nil {
			return errs.New(500, "server internal error", err)
		}

		hashedRefreshToken, err := token.HashToken(newRefreshToken)
		if err != nil {
			return errs.New(500, "server internal error", err)
		}

		newTokenDocument := models.AuthToken{
			UserID:           userValue.ID,
			AccessToken:      newAccessToken,
			RefreshToken:     hashedRefreshToken,
			TokenType:        "Bearer",
			AccessExpiresAt:  primitive.NewDateTimeFromTime(time.Now().Add(time.Minute * token.AccessTokenDuration)),
			RefreshExpiresAt: primitive.NewDateTimeFromTime(time.Now().Add(time.Minute * token.RefreshTokenDuration)),
		}

		_, err = tokens.InsertOne(sctx, newTokenDocument)
		if err != nil {
			sctx.AbortTransaction(sctx)
			return errs.New(500, "server internal error", err)
		}

		err = sctx.CommitTransaction(sctx)
		if err != nil {
			return errs.New(500, "server internal error", err)
		}

		authResponse = views.AuthResponse{
			AccessToken:  newTokenDocument.AccessToken,
			TokenType:    newTokenDocument.TokenType,
			ExpiresIn:    int(time.Duration(token.AccessTokenDuration) * time.Second),
			RefreshToken: newTokenDocument.RefreshToken,
		}

		return nil
	})

	if err != nil {
		return views.AuthResponse{}, errs.New(500, err.Error(), err)
	}

	return authResponse, nil
}

func (a *AuthUsecase) Refresh(accessToken, refreshToken string) (views.RefreshResponse, error) {
	var refreshResponse views.RefreshResponse

	users := a.db.Collection("users")
	tokens := a.db.Collection("tokens")

	err := a.db.Client().UseSession(context.Background(), func(sctx mongo.SessionContext) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return errs.New(500, "server internal error", err)
		}

		tokenValue := models.AuthToken{}
		filterByAccessToken := bson.M{"access_token": accessToken}

		err = tokens.FindOne(sctx, filterByAccessToken).Decode(&tokenValue)
		if err != nil {
			return errs.New(403, "access token expired", err)
		}

		if tokenValue.AccessExpiresAt.Time().Before(time.Now()) {
			return errs.New(403, "access token expired", nil)
		}

		if !token.CheckTokenHash(refreshToken, tokenValue.RefreshToken) {
			return errs.New(403, "access forbidden", nil)
		}

		// Refresh token
		userValue := models.User{}
		filterByUserID := bson.M{"_id": tokenValue.UserID}
		err = users.FindOne(sctx, filterByUserID).Decode(&userValue)
		if err != nil {
			return errs.New(500, "server internal error", err)
		}

		newAccessToken, err := token.CreateAccessToken(userValue.GUID)
		if err != nil {
			return errs.New(500, "server internal error", err)
		}

		newRefreshToken, err := token.CreateRefreshToken(userValue.GUID)
		if err != nil {
			return errs.New(500, "server internal error", err)
		}

		replaceToken := models.AuthToken{
			AccessToken:      newAccessToken,
			RefreshToken:     newRefreshToken,
			TokenType:        "Bearer",
			UserID:           userValue.ID,
			AccessExpiresAt:  primitive.NewDateTimeFromTime(time.Now().Add(time.Minute * token.AccessTokenDuration)),
			RefreshExpiresAt: primitive.NewDateTimeFromTime(time.Now().Add(time.Minute * token.RefreshTokenDuration)),
		}

		opt := options.FindOneAndReplace().SetUpsert(true).SetReturnDocument(options.After)
		err = tokens.FindOneAndReplace(sctx, filterByAccessToken, replaceToken, opt).Decode(&tokenValue)
		if err != nil {
			sctx.AbortTransaction(sctx)
			return errs.New(500, "server internal error", err)
		}

		err = sctx.CommitTransaction(sctx)
		if err != nil {
			return errs.New(500, "server internal error", err)
		}

		refreshResponse = views.RefreshResponse{
			AccessToken:  tokenValue.AccessToken,
			TokenType:    tokenValue.TokenType,
			ExpiresIn:    int(time.Duration(token.AccessTokenDuration) * time.Second),
			RefreshToken: tokenValue.RefreshToken,
		}

		return nil
	})

	if err != nil {
		return views.RefreshResponse{}, errs.New(500, err.Error(), err)
	}

	return refreshResponse, nil
}

func (a *AuthUsecase) Delete(accessToken, refreshToken string) error {

	tokens := a.db.Collection("tokens")

	err := a.db.Client().UseSession(context.Background(), func(sctx mongo.SessionContext) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)

		if err != nil {
			return errs.New(500, "server internal error", err)
		}

		tokenValue := models.AuthToken{}
		filterByAccessToken := bson.M{"access_token": accessToken}

		err = tokens.FindOne(sctx, filterByAccessToken).Decode(&tokenValue)
		if err != nil {
			return errs.New(403, "access token expired", err)
		}

		if tokenValue.AccessExpiresAt.Time().Before(time.Now()) {
			return errs.New(403, "access token expired", nil)
		}

		// Delete refresh token
		if !token.CheckTokenHash(refreshToken, tokenValue.RefreshToken) {
			return errs.New(403, "Access forbidden", nil)
		}
		err = tokens.FindOneAndDelete(sctx, filterByAccessToken).Err()
		if err != nil {
			sctx.AbortTransaction(sctx)
			return errs.New(500, "server internal error", err)
		}

		if err != nil {
			sctx.AbortTransaction(sctx)
			return errs.New(500, "server internal error", err)
		}

		err = sctx.CommitTransaction(sctx)
		if err != nil {
			return errs.New(500, "server internal error", err)
		}

		return nil
	})

	return err
}

func (a *AuthUsecase) DeleteAll(accessToken string) error {

	tokens := a.db.Collection("tokens")

	err := a.db.Client().UseSession(context.Background(), func(sctx mongo.SessionContext) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)

		if err != nil {
			return errs.New(500, "server internal error", err)
		}

		tokenValue := models.AuthToken{}
		filterByAccessToken := bson.M{"access_token": accessToken}

		err = tokens.FindOne(sctx, filterByAccessToken).Decode(&tokenValue)
		if err != nil {
			return errs.New(403, "access token expired", err)
		}

		if tokenValue.AccessExpiresAt.Time().Before(time.Now()) {
			return errs.New(403, "access token expired", nil)
		}

		// Delete all tokens for user
		deleteFilter := bson.M{"user_id": tokenValue.UserID}
		_, err = tokens.DeleteMany(sctx, deleteFilter)
		if err != nil {
			sctx.AbortTransaction(sctx)
			return errs.New(500, "server internal error", err)
		}

		err = sctx.CommitTransaction(sctx)
		if err != nil {
			return errs.New(500, "server internal error", err)
		}

		return nil
	})

	return err
}
