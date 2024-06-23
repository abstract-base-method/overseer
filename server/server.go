package server

import (
	"context"
	overseerAuth "overseer/auth"
	v1 "overseer/build/go"
	"overseer/common"
	"overseer/engine"
	"overseer/engine/handlers"
	"overseer/generative"
	"overseer/storage"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const sqliteDBPath = "overseer.db"

func NewServer() (*grpc.Server, error) {
	server := grpc.NewServer(
		grpc.ChainStreamInterceptor(
			recovery.StreamServerInterceptor(
				recovery.WithRecoveryHandlerContext(func(ctx context.Context, p interface{}) error {
					log := common.GetLogger("server")
					info, err := common.GetContextInformation(ctx)
					if err != nil {
						log.Error("panic recovered - failed to get context information", "error", err, "panic", p)
					} else {
						log.Error("panic recovered", info.LoggingContext("panic", p)...)
					}
					return status.Error(codes.Internal, "unrecoverable error")
				}),
			),
			overseerAuth.StreamServerAuthFunc,
		),
		grpc.ChainUnaryInterceptor(
			recovery.UnaryServerInterceptor(
				recovery.WithRecoveryHandlerContext(func(ctx context.Context, p interface{}) error {
					log := common.GetLogger("server")
					info, err := common.GetContextInformation(ctx)
					if err != nil {
						log.Error("panic recovered - failed to get context information", "error", err, "panic", p)
					} else {
						log.Error("panic recovered", info.LoggingContext("panic", p)...)
					}
					return status.Error(codes.Internal, "unrecoverable error")
				}),
			),
			overseerAuth.UnaryServerAuthFunc,
		),
	)

	// dependencies
	db, err := storage.NewSqliteDB(sqliteDBPath, true)
	if err != nil {
		common.GetLogger("server").Error("failed to create db", "error", err)
		return nil, err
	}
	eventStore := storage.NewSqlEventStore(db)
	userStore := storage.NewSqlUserStore(db)
	gameStore := storage.NewSqlGameStore(db, userStore)
	mapStore := storage.NewSqlMapStore(db)
	lockStore := storage.NewSqlLockStore(db)

	mapGeneration, err := generative.NewMapGenerationService()
	if err != nil {
		common.GetLogger("server").Error("failed to create map generation service", "error", err)
		return nil, err
	}

	userServer := NewUserServer(userStore)
	gameServer := NewGameServer(userServer, lockStore, gameStore)
	bus := engine.NewEventBus([]engine.EventHandler{
		handlers.NewGameHandler(gameStore, eventStore),
	}, gameServer, userServer, eventStore)
	eventServer := NewEventServer(bus)
	mapServer := NewMapServer(mapStore, mapGeneration)

	v1.RegisterEventsServer(server, eventServer)
	v1.RegisterUsersServer(server, userServer)
	v1.RegisterGamesServer(server, gameServer)
	v1.RegisterMapsServer(server, mapServer)

	return server, nil
}
