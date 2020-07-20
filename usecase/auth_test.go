package usecase_test

import (
	"context"
	"encoding/base64"
	"log"
	"os"
	"testing"

	"github.com/flaambe/authservice/models"
	"github.com/flaambe/authservice/mongoconf"
	"github.com/flaambe/authservice/usecase"
	"github.com/stretchr/testify/require"

	"go.mongodb.org/mongo-driver/bson"
)

var (
	dbConfig    *mongoconf.Config
	authUseCase *usecase.AuthUsecase
)

func TestMain(m *testing.M) {
	dbConfig = mongoconf.NewConfig()
	if err := dbConfig.Open(os.Getenv("MONGODB_TEST_URI"), os.Getenv("DBNAME_TEST")); err != nil {
		log.Fatal(err)
	}

	if err := dbConfig.EnsureIndexes(); err != nil {
		log.Fatal(err)
	}

	authUseCase = usecase.NewAuthUsecase(dbConfig.DB)

	exitVal := m.Run()

	_ = dbConfig.DB.Client().Disconnect(context.TODO())

	os.Exit(exitVal)
}

func TestAuth(t *testing.T) {
	// Validate GUID and Token type
	authResponse, err := authUseCase.Auth("4aa32cc5-d0e6-49e7-897d-d2b26748b7d3")
	require.NoError(t, err)
	require.Equal(t, "Bearer", authResponse.TokenType)

	// Refresh token should be base64 encoded
	_, err = base64.StdEncoding.DecodeString(authResponse.RefreshToken)
	require.NoError(t, err)
}

func TestRefreshToken(t *testing.T) {
	authResponse, err := authUseCase.Auth("4aa32cc5-d0e6-49e7-897d-d2b26748b7d3")
	require.NoError(t, err)

	_, err = authUseCase.RefreshToken(authResponse.AccessToken, authResponse.RefreshToken)
	require.NoError(t, err)
}

func TestDeleteToken(t *testing.T) {
	authResponse, err := authUseCase.Auth("4aa32cc5-d0e6-49e7-897d-d2b26748b7d3")
	require.NoError(t, err)

	err = authUseCase.DeleteToken(authResponse.AccessToken, authResponse.RefreshToken)
	require.NoError(t, err)

	filter := bson.M{"refresh_token": authResponse.RefreshToken}
	result := dbConfig.DB.Collection("tokens").FindOne(context.TODO(), filter)
	require.Error(t, result.Err())
}

func TestDeleteAllTokens(t *testing.T) {
	authResponse, err := authUseCase.Auth("4aa32cc5-d0e6-49e7-897d-d2b26748b7d3")
	require.NoError(t, err)

	err = authUseCase.DeleteAllTokens(authResponse.AccessToken)
	require.NoError(t, err)

	user := models.User{}
	filter := bson.M{"guid": "4aa32cc5-d0e6-49e7-897d-d2b26748b7d3"}
	err = dbConfig.DB.Collection("users").FindOne(context.TODO(), filter).Decode(&user)
	require.NoError(t, err)

	filter = bson.M{"guid": user.GUID}
	cursor, err := dbConfig.DB.Collection("tokens").Find(context.TODO(), filter)
	require.NoError(t, err)

	require.Nil(t, cursor.Current)
}
