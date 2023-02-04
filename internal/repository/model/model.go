package model

import "github.com/google/uuid"

type Player struct {
	Id       uuid.UUID `bson:"_id"`
	Username string    `bson:"username"`

	GameServerId string `bson:"gameServerId"`
	ProxyId      string `bson:"proxyId"`
}
