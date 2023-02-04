package repository

import (
	"context"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"player-tracker/internal/config"
	"player-tracker/internal/repository/registrytypes"
)

const (
	databaseName         = "player-tracker"
	serverCollectionName = "server"
	playerCollectionName = "player"
)

type mongoRepository struct {
	Repository
	db *mongo.Database

	serverCollection *mongo.Collection
	playerCollection *mongo.Collection
}

func NewMongoRepository(ctx context.Context, cfg *config.MongoDBConfig) (Repository, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI).SetRegistry(createCodecRegistry()))
	if err != nil {
		return nil, err
	}

	database := client.Database(databaseName)
	return &mongoRepository{
		db:               database,
		serverCollection: database.Collection(serverCollectionName),
		playerCollection: database.Collection(playerCollectionName),
	}, nil
}

func (r *mongoRepository) HealthPing(ctx context.Context) error {
	return r.Redis.Ping(ctx).Err()
}

func (r *mongoRepository) SetPlayerGameServer(ctx context.Context, playerId uuid.UUID, serverId string) error {

}

func createCodecRegistry() *bsoncodec.Registry {
	return bson.NewRegistryBuilder().
		RegisterTypeEncoder(registrytypes.UUIDType, bsoncodec.ValueEncoderFunc(registrytypes.UuidEncodeValue)).
		RegisterTypeDecoder(registrytypes.UUIDType, bsoncodec.ValueDecoderFunc(registrytypes.UuidDecodeValue)).
		Build()
}
