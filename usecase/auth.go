package usecase

import (
	"context"
	"log"

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

func (a *AuthUsecase) Auth(body views.AccessTokenRequest) (views.TokensResponse, error) {
	var tokensResponse views.TokensResponse

	users := a.db.Collection("users")

	err := a.db.Client().UseSession(context.Background(), func(sctx mongo.SessionContext) error {
		err := sctx.StartTransaction(options.Transaction().
			SetReadConcern(readconcern.Snapshot()).
			SetWriteConcern(writeconcern.New(writeconcern.WMajority())),
		)
		if err != nil {
			return err
		}

		_, err = users.InsertOne(sctx, bson.M{"guid": body.GUID})
		if err != nil {
			sctx.AbortTransaction(sctx)
			log.Println("caught exception during transaction, aborting.")
			return err
		}

		for {
			err = sctx.CommitTransaction(sctx)
			switch e := err.(type) {
			case nil:
				return nil
			case mongo.CommandError:
				if e.HasErrorLabel("UnknownTransactionCommitResult") {
					log.Println("UnknownTransactionCommitResult, retrying commit operation...")
					continue
				}
				log.Println("Error during commit...")
				return e
			default:
				log.Println("Error during commit...")
				return e
			}
		}
	})

	if err != nil {
		log.Fatal(err)
	}

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
	return tokensResponse, nil
}
