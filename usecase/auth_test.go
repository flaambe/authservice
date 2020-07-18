package usecase_test

import (
	"context"
	"encoding/base64"
	"log"
	"os"
	"testing"
	"time"

	"github.com/flaambe/authservice/usecase"
	"github.com/stretchr/testify/require"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	db          *mongo.Database
	authUseCase *usecase.AuthUsecase
)

func ensureindexes(db *mongo.Database) {
	uniqUserIndex := mongo.IndexModel{
		Keys:    bson.M{"guid": 1},
		Options: options.Index().SetUnique(true),
	}

	_, err := db.Collection("users").Indexes().CreateOne(context.TODO(), uniqUserIndex)

	if err != nil {
		log.Println(err)
	}

	expireTokenIndex := mongo.IndexModel{
		Keys:    bson.M{"refresh_expires_at": 1},
		Options: options.Index().SetExpireAfterSeconds(0),
	}

	_, err = db.Collection("tokens").Indexes().CreateOne(context.TODO(), expireTokenIndex)

	if err != nil {
		log.Println(err)
	}
}

func setup() {
	mongoURI := os.Getenv("MONGODB_TEST_URI")
	dbName := os.Getenv("DBNAME_TEST")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}

	db = client.Database(dbName)

	ensureindexes(db)

	if err != nil {
		log.Fatal(err)
	}
}

func TestMain(m *testing.M) {
	setup()

	authUseCase = usecase.NewAuthUsecase(db)

	exitVal := m.Run()

	_ = db.Client().Disconnect(context.TODO())

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
}

func TestDeleteAllTokens(t *testing.T) {
	authResponse, err := authUseCase.Auth("4aa32cc5-d0e6-49e7-897d-d2b26748b7d3")
	require.NoError(t, err)

	err = authUseCase.DeleteAllTokens(authResponse.AccessToken)
	require.NoError(t, err)
}
