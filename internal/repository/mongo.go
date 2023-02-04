package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"player-tracker/internal/config"
	"player-tracker/internal/repository/model"
	"player-tracker/internal/repository/registrytypes"
	"time"
)

const (
	databaseName         = "player-tracker"
	playerCollectionName = "player"
)

type mongoRepository struct {
	Repository
	db *mongo.Database

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
		playerCollection: database.Collection(playerCollectionName),
	}, nil
}

func (r *mongoRepository) SetPlayerGameServer(ctx context.Context, playerId uuid.UUID, serverId string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := r.playerCollection.UpdateByID(ctx, playerId, bson.M{"$set": bson.M{"gameServerId": serverId}})
	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		_, err := r.playerCollection.InsertOne(ctx, bson.M{"_id": playerId, "gameServerId": serverId})
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *mongoRepository) GetPlayerGameServer(ctx context.Context, playerId uuid.UUID) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := r.playerCollection.Distinct(ctx, "gameServerId", bson.M{"_id": playerId})
	if err != nil {
		return "", err
	}
	if len(res) == 0 {
		return "", mongo.ErrNoDocuments
	}

	return res[0].(string), nil
}

func (r *mongoRepository) SetPlayerProxy(ctx context.Context, playerId uuid.UUID, proxyId string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := r.playerCollection.UpdateByID(ctx, playerId, bson.M{"$set": bson.M{"proxyId": proxyId}})
	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		_, err := r.playerCollection.InsertOne(ctx, bson.M{"_id": playerId, "proxyId": proxyId})
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *mongoRepository) GetPlayerProxy(ctx context.Context, playerId uuid.UUID) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := r.playerCollection.Distinct(ctx, "proxyId", bson.M{"_id": playerId})
	if err != nil {
		return "", err
	}
	if len(res) == 0 {
		return "", mongo.ErrNoDocuments
	}

	return res[0].(string), nil
}

func (r *mongoRepository) GetPlayer(ctx context.Context, playerId uuid.UUID) (*model.Player, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var player model.Player
	err := r.playerCollection.FindOne(ctx, bson.M{"_id": playerId}).Decode(&player)
	if err != nil {
		return nil, err
	}

	return &player, nil
}

func (r *mongoRepository) GetPlayers(ctx context.Context, playerIds []uuid.UUID) ([]*model.Player, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var players []*model.Player
	cursor, err := r.playerCollection.Find(ctx, bson.M{"_id": bson.M{"$in": playerIds}})
	if err != nil {
		return nil, err
	}

	for cursor.Next(ctx) {
		var player model.Player
		err := cursor.Decode(&player)
		if err != nil {
			return nil, err
		}

		players = append(players, &player)
	}

	return players, nil
}

func (r *mongoRepository) DeletePlayer(ctx context.Context, playerId uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := r.playerCollection.DeleteOne(ctx, bson.M{"_id": playerId})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (r *mongoRepository) GetServerPlayers(ctx context.Context, serverId string) ([]*model.Player, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var players []*model.Player
	cursor, err := r.playerCollection.Find(ctx, bson.M{"gameServerId": serverId})
	if err != nil {
		return nil, err
	}

	for cursor.Next(ctx) {
		var player model.Player
		err := cursor.Decode(&player)
		if err != nil {
			return nil, err
		}

		players = append(players, &player)
	}

	return players, nil
}

func (r *mongoRepository) GetServerPlayerCount(ctx context.Context, targetId string, proxy bool) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if proxy {
		return r.playerCollection.CountDocuments(ctx, bson.M{"proxyId": targetId})
	}
	return r.playerCollection.CountDocuments(ctx, bson.M{"gameServerId": targetId})
}

func (r *mongoRepository) GetServerTypePlayerCount(ctx context.Context, fleetName string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return r.playerCollection.CountDocuments(ctx, bson.M{"gameServerId": bson.M{"$regex": fmt.Sprintf("^%s", fleetName+"-")}})
}

func (r *mongoRepository) PlayerCount(ctx context.Context) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return r.playerCollection.CountDocuments(ctx, bson.M{})
}

func createCodecRegistry() *bsoncodec.Registry {
	return bson.NewRegistryBuilder().
		RegisterTypeEncoder(registrytypes.UUIDType, bsoncodec.ValueEncoderFunc(registrytypes.UuidEncodeValue)).
		RegisterTypeDecoder(registrytypes.UUIDType, bsoncodec.ValueDecoderFunc(registrytypes.UuidDecodeValue)).
		Build()
}
