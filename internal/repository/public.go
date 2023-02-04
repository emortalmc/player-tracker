package statestore

import (
	"context"
	"github.com/google/uuid"
)

type StateStore interface {
	HealthPing(ctx context.Context) error

	SetPlayerGameServer(ctx context.Context, playerId uuid.UUID, serverId string) error
	GetPlayerGameServer(ctx context.Context, playerId uuid.UUID) (string, error)

	SetPlayerProxy(ctx context.Context, playerId uuid.UUID, proxyId string) error
	GetPlayerProxy(ctx context.Context, playerId uuid.UUID) (string, error)

	GetServerPlayerIds(ctx context.Context, serverId string) ([]string, error)
	GetServerPlayerCount(ctx context.Context, serverId string) (int64, error)

	GetServerTypePlayerCount(ctx context.Context, serverId string) (int64, error)
}
