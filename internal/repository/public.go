package repository

import (
	"context"
	"github.com/google/uuid"
	"player-tracker/internal/repository/model"
)

// Repository contains methods for all repository implementations.
// All Set methods should insert if the Player is not already present
type Repository interface {
	SetPlayerGameServer(ctx context.Context, playerId uuid.UUID, serverId string) error

	SetPlayerProxy(ctx context.Context, playerId uuid.UUID, proxyId string) error

	GetPlayer(ctx context.Context, playerId uuid.UUID) (*model.Player, error)
	GetPlayers(ctx context.Context, playerIds []uuid.UUID) ([]*model.Player, error)
	DeletePlayer(ctx context.Context, playerId uuid.UUID) error

	GetServerPlayers(ctx context.Context, serverId string) ([]*model.Player, error)
	GetServerPlayerCount(ctx context.Context, serverId string, proxy bool) (int64, error)

	// GetServerTypePlayerCount returns the number of players on a server type
	// where fleetName is the prefix of the server type (e.g. {fleetName}-3xja3t-qlx35)
	GetServerTypePlayerCount(ctx context.Context, fleetName string) (int64, error)
	PlayerCount(ctx context.Context) (int64, error)
}
