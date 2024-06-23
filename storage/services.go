package storage

import (
	"context"
	v1 "overseer/build/go"
)

type LockStore interface {
	LockGame(ctx context.Context, request *v1.LockGameRequest) (bool, error)
	UnlockGame(ctx context.Context, request *v1.UnlockGameRequest) (bool, error)
}

type GameStore interface {
	CreateGame(ctx context.Context, game *v1.Game) error
	GetGame(ctx context.Context, id string) (*v1.Game, error)
	SaveGame(ctx context.Context, game *v1.Game) error
}

type UserStore interface {
	UpsertUser(ctx context.Context, user *v1.User) error
	UpsertActor(ctx context.Context, user string, actor *v1.Actor) error
	GetUser(ctx context.Context, id string) (*v1.User, error)
	GetUserForActor(ctx context.Context, actorID string) (*v1.User, error)
	GetActor(ctx context.Context, id string) (*v1.Actor, error)
	GetActors(ctx context.Context, id string) ([]*v1.Actor, error)
	DeleteUser(ctx context.Context, id string) error
	DeleteActor(ctx context.Context, id string) error
}

type EventStore interface {
	RecordEvent(ctx context.Context, event *v1.Event) (*v1.EventRecord, error)
	RecordReceipt(ctx context.Context, receipt *v1.EventReceipt) error
	GetEvent(ctx context.Context, id string) (*v1.EventRecord, error)
}

type MapStore interface {
	CreateMap(ctx context.Context, request *v1.CreateMapRequest) (*v1.Map, error)
	GetMap(ctx context.Context, uid string) (*v1.Map, error)
	CreateCoordinate(ctx context.Context, coordinate *v1.MapCoordinateDetail) error
	GetCoordinates(ctx context.Context, mapId string) ([]*v1.MapCoordinateDetail, error)
	UpdateCoordinate(ctx context.Context, coordinate *v1.MapCoordinateDetail) error
	GetCoordinate(ctx context.Context, gameId string, mapId string, x int64, y int64) (*v1.MapCoordinateDetail, error)
}
