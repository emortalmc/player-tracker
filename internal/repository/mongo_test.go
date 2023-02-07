package repository

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"os"
	"player-tracker/internal/config"
	"player-tracker/internal/repository/model"
	"testing"
)

const (
	mongoUri = "mongodb://root:password@localhost:%s"
)

var (
	dbClient *mongo.Client
	database *mongo.Database
	repo     Repository
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("could not constuct pool: %s", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("could not connect to docker: %s", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "mongo",
		Tag:        "6.0.3",
		Env: []string{
			"MONGO_INITDB_ROOT_USERNAME=root",
			"MONGO_INITDB_ROOT_PASSWORD=password",
		},
	}, func(cfg *docker.HostConfig) {
		cfg.AutoRemove = true
		cfg.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		log.Fatalf("could not start resource: %s", err)
	}

	uri := fmt.Sprintf(mongoUri, resource.GetPort("27017/tcp"))

	err = pool.Retry(func() (err error) {
		dbClient, err = mongo.Connect(context.Background(), options.Client().ApplyURI(uri).SetRegistry(createCodecRegistry()))
		if err != nil {
			return
		}
		err = dbClient.Ping(context.Background(), nil)
		if err != nil {
			return
		}

		// Ping was successful, let's create the mongo repo
		repo, err = NewMongoRepository(context.Background(), &config.MongoDBConfig{URI: uri})
		database = dbClient.Database(databaseName)
		return
	})

	if err != nil {
		log.Fatalf("could not connect to docker: %s", err)
	}

	code := m.Run()

	if err := pool.Purge(resource); err != nil {
		log.Fatalf("could not purge resource: %s", err)
	}

	if err = dbClient.Disconnect(context.Background()); err != nil {
		log.Panicf("could not disconnect from mongo: %s", err)
	}

	os.Exit(code)
}

func TestMongoRepository_SetPlayerGameServer(t *testing.T) {
	playerId := uuid.New()
	serverId := "lobby-z24523-sdhbsd"
	proxyId := "proxy-sdgwsd-235eax"

	type args struct {
		playerId uuid.UUID
		serverId string
	}
	tests := []struct {
		name    string
		data    []model.Player
		args    args
		wantErr error
		wantDb  []model.Player
	}{
		{
			name: "doesnt_exist",
			args: args{
				playerId: playerId,
				serverId: serverId,
			},
			wantErr: nil,
			wantDb: []model.Player{
				{
					Id:           playerId,
					GameServerId: serverId,
				},
			},
		},
		{
			name: "already_exists",
			data: []model.Player{
				{
					Id:           playerId,
					GameServerId: "original-server-id",
					ProxyId:      proxyId,
				},
			},
			args: args{
				playerId: playerId,
				serverId: serverId,
			},
			wantErr: nil,
			wantDb: []model.Player{
				{
					Id:           playerId,
					GameServerId: serverId,
					ProxyId:      proxyId,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Cleanup(cleanup())
			// Insert test data
			if test.data != nil {
				_, err := database.Collection(playerCollectionName).InsertMany(context.Background(), convertToInterfaceSlice(test.data))
				assert.NoError(t, err)
			}

			err := repo.SetPlayerGameServer(context.Background(), test.args.playerId, test.args.serverId)
			assert.Equal(t, test.wantErr, err)

			// Check the database contents
			var players []model.Player
			cursor, err := database.Collection(playerCollectionName).Find(context.Background(), bson.D{})
			assert.NoError(t, err)

			err = cursor.All(context.Background(), &players)
			assert.NoError(t, err)

			assert.Equal(t, test.wantDb, players)
		})
	}
}

func TestMongoRepository_SetPlayerProxy(t *testing.T) {
	playerId := uuid.New()
	serverId := "lobby-z24523-sdhbsd"
	proxyId := "proxy-sdgwsd-235eax"

	type args struct {
		playerId uuid.UUID
		proxyId  string
	}

	tests := []struct {
		name    string
		data    []model.Player
		args    args
		wantErr error
		wantDb  []model.Player
	}{
		{
			name: "doesnt_exist",
			args: args{
				playerId: playerId,
				proxyId:  proxyId,
			},
		},
		{
			name: "already_exists",
			data: []model.Player{
				{
					Id:           playerId,
					GameServerId: serverId,
					ProxyId:      "original-proxy-id",
				},
			},
			args: args{
				playerId: playerId,
				proxyId:  proxyId,
			},
			wantErr: nil,
			wantDb: []model.Player{
				{
					Id:           playerId,
					GameServerId: serverId,
					ProxyId:      proxyId,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Cleanup(cleanup())

			// Insert test data
			if test.data != nil {
				_, err := database.Collection(playerCollectionName).InsertMany(context.Background(), convertToInterfaceSlice(test.data))
				assert.NoError(t, err)
			}

			err := repo.SetPlayerProxy(context.Background(), test.args.playerId, test.args.proxyId)
			assert.Equal(t, test.wantErr, err)

			// Check the database contents
			var players []model.Player
			cursor, err := database.Collection(playerCollectionName).Find(context.Background(), bson.D{})
			assert.NoError(t, err)

			err = cursor.All(context.Background(), &players)
			assert.NoError(t, err)
		})
	}
}

func TestMongoRepository_GetPlayer(t *testing.T) {
	playerId := uuid.New()
	serverId := "lobby-z24523-sdhbsd"
	proxyId := "proxy-sdgwsd-235eax"

	tests := []struct {
		name     string
		data     *model.Player
		playerId uuid.UUID
		want     *model.Player
		wantErr  error
	}{
		{
			name:     "doesnt_exist",
			data:     nil,
			playerId: playerId,
			want:     nil,
			wantErr:  mongo.ErrNoDocuments,
		},
		{
			name: "exists",
			data: &model.Player{
				Id:           playerId,
				GameServerId: serverId,
				ProxyId:      proxyId,
			},
			playerId: playerId,
			want: &model.Player{
				Id:           playerId,
				GameServerId: serverId,
				ProxyId:      proxyId,
			},
			wantErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Cleanup(cleanup())
			// Insert test data
			if test.data != nil {
				_, err := database.Collection(playerCollectionName).InsertOne(context.Background(), test.data)
				assert.NoError(t, err)
			}

			got, err := repo.GetPlayer(context.Background(), test.playerId)
			assert.Equal(t, test.wantErr, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestMongoRepository_DeletePlayer(t *testing.T) {
	playerId := uuid.New()
	serverId := "lobby-z24523-sdhbsd"
	proxyId := "proxy-sdgwsd-235eax"

	tests := []struct {
		name     string
		data     *model.Player
		playerId uuid.UUID
		wantErr  error
	}{
		{
			name:     "doesnt_exist",
			data:     nil,
			playerId: playerId,
			wantErr:  mongo.ErrNoDocuments,
		},
		{
			name: "exists",
			data: &model.Player{
				Id:           playerId,
				GameServerId: serverId,
				ProxyId:      proxyId,
			},
			playerId: playerId,
			wantErr:  nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Cleanup(cleanup())
			// Insert test data
			if test.data != nil {
				_, err := database.Collection(playerCollectionName).InsertOne(context.Background(), test.data)
				assert.NoError(t, err)
			}

			err := repo.DeletePlayer(context.Background(), test.playerId)
			assert.Equal(t, test.wantErr, err)
		})
	}
}

func TestMongoRepository_GetServerPlayerCount(t *testing.T) {
	playerIds := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	serverIds := []string{"lobby-1", "lobby-2", "lobby-3"}
	proxyIds := []string{"proxy-1", "proxy-2", "proxy-3"}

	tests := []struct {
		name            string
		data            []model.Player
		serverOrProxyId string
		proxy           bool
		want            int64
		wantErr         error
	}{
		{
			name:            "empty",
			data:            nil,
			serverOrProxyId: serverIds[0],
			proxy:           false,
			want:            0,
			wantErr:         nil,
		},
		{
			name: "valid_multiple_servers",
			data: []model.Player{
				{
					Id:           playerIds[0],
					GameServerId: serverIds[0],
					ProxyId:      proxyIds[0],
				},
				{
					Id:           playerIds[1],
					GameServerId: serverIds[0], // Same server
					ProxyId:      proxyIds[1],
				},
				{
					Id:           playerIds[2],
					GameServerId: serverIds[1], // Different server
					ProxyId:      proxyIds[2],
				},
			},
			serverOrProxyId: serverIds[0],
			proxy:           false,
			want:            2,
			wantErr:         nil,
		},
		{
			name: "proxy_count",
			data: []model.Player{
				{
					Id:           playerIds[0],
					GameServerId: serverIds[0],
					ProxyId:      proxyIds[0],
				},
				{
					Id:           playerIds[1],
					GameServerId: serverIds[1],
					ProxyId:      proxyIds[0], // Same proxy
				},
				{
					Id:           playerIds[2],
					GameServerId: serverIds[2],
					ProxyId:      proxyIds[1], // Different proxy
				},
			},
			serverOrProxyId: proxyIds[0],
			proxy:           true,
			want:            2,
			wantErr:         nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Cleanup(cleanup())
			// Insert test data
			if test.data != nil {
				_, err := database.Collection(playerCollectionName).InsertMany(context.Background(), convertToInterfaceSlice(test.data))
				assert.NoError(t, err)
			}

			got, err := repo.GetServerPlayerCount(context.Background(), test.serverOrProxyId, test.proxy)
			assert.Equal(t, test.wantErr, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestMongoRepository_GetServerTypePlayerCount(t *testing.T) {
	playerIds := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	fleetIds := []string{"lobby", "block-sumo"}
	serverIds := []string{"lobby-1", "block-sumo-2"}
	proxyIds := []string{"proxy-1", "proxy-2", "proxy-3"}

	tests := []struct {
		name    string
		data    []model.Player
		fleetId string
		want    int64
		wantErr error
	}{
		{
			name:    "empty",
			data:    nil,
			fleetId: fleetIds[0],
			want:    0,
			wantErr: nil,
		},
		{
			name: "valid_multiple_servers",
			data: []model.Player{
				{
					Id:           playerIds[0],
					GameServerId: serverIds[0],
					ProxyId:      proxyIds[0],
				},
				{
					Id:           playerIds[1],
					GameServerId: serverIds[0], // Same server
					ProxyId:      proxyIds[1],
				},
				{
					Id:           playerIds[2],
					GameServerId: serverIds[1], // Different server
					ProxyId:      proxyIds[2],
				},
			},
			fleetId: fleetIds[0],
			want:    2,
			wantErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Cleanup(cleanup())
			// Insert test data
			if test.data != nil {
				_, err := database.Collection(playerCollectionName).InsertMany(context.Background(), convertToInterfaceSlice(test.data))
				assert.NoError(t, err)
			}

			got, err := repo.GetServerTypePlayerCount(context.Background(), test.fleetId)
			assert.Equal(t, test.wantErr, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func convertToInterfaceSlice[T any](data []T) []interface{} {
	var result []interface{}
	for _, player := range data {
		result = append(result, player)
	}
	return result
}

func cleanup() func() {
	return func() {
		if err := database.Drop(context.TODO()); err != nil {
			log.Panicf("could not drop database: %s", err)
		}
	}
}
