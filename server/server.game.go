package server

import (
	"context"
	"fmt"
	"overseer/auth"
	v1 "overseer/build/go"
	"overseer/common"
	"overseer/storage"
	"slices"
	"time"

	charm "github.com/charmbracelet/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type defaultGameServer struct {
	locks storage.LockStore
	games storage.GameStore
	// todo: replace this with a client to avoid loopback dependencies
	users v1.UsersServer
	log   *charm.Logger
	v1.UnimplementedGamesServer
}

func NewGameServer(users v1.UsersServer, locks storage.LockStore, games storage.GameStore) v1.GamesServer {
	return &defaultGameServer{
		users: users,
		locks: locks,
		games: games,
		log:   common.GetLogger("server.game"),
	}
}

func (s *defaultGameServer) CreateGame(ctx context.Context, req *v1.CreateGameRequest) (*v1.Game, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("error getting context information", err)
		return nil, err
	}

	s.log.Debug(
		"storing new game",
		info.LoggingContext()...,
	)

	err = s.validateActors(ctx, req.Participants)
	if err != nil {
		s.log.Error("failed to validate actors", info.LoggingContext("error", err)...)
		return nil, err
	}

	game := &v1.Game{
		Uid: common.GenerateRandomStringFromSeed(
			"game",
			common.GenerateUniqueId(),
			fmt.Sprintf("%d", time.Now().UTC().UnixMilli()),
		),
		Name:         req.Name,
		Theme:        req.Theme,
		ActiveActor:  info.Actor,
		Participants: req.Participants,
	}
	err = s.games.CreateGame(ctx, game)
	if err != nil {
		s.log.Error("failed to create game", info.LoggingContext("error", err)...)
		return nil, err
	}

	s.log.Info("game created", info.LoggingContext("game", game.Uid)...)
	return game, nil
}

func (s *defaultGameServer) validateActors(ctx context.Context, actors []*v1.Actor) error {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("error getting context information", err)
		return err
	}

	foundCurrent := info.User.Uid == auth.SystemUserId && info.Actor.Uid == auth.SystemActorId
	if foundCurrent {
		s.log.Info("system actor detected bypassing actor validation for new game")
		return nil
	}

	for _, actor := range actors {
		s.log.Debug("validating actor", "actor", actor.Uid)
		_, err := s.users.GetActor(ctx, &v1.GetActorRequest{
			ActorId: actor.Uid,
		})
		if err != nil {
			return err
		}

		if actor.Uid == info.Actor.Uid {
			foundCurrent = true
		}
	}

	if !foundCurrent {
		return status.Error(codes.InvalidArgument, "current actor is not part of the participants")
	}

	return nil
}

func (s *defaultGameServer) GetGame(ctx context.Context, req *v1.GetGameRequest) (*v1.Game, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("error getting context information", err)
		return nil, err
	}

	game, err := s.games.GetGame(ctx, req.GameUid)
	if err != nil {
		return nil, err
	}

	if game == nil {
		s.log.Error("game not found", info.LoggingContext("game", req.GameUid)...)
		return nil, status.Error(codes.NotFound, "game not found")
	}

	err = s.validateActors(ctx, game.Participants)
	if err == nil {
		return game, nil
	} else {
		s.log.Error("actor is not part of the game", info.LoggingContext("game", req.GameUid)...)
		return nil, status.Error(codes.PermissionDenied, "actor is not part of the game")
	}
}

func (s *defaultGameServer) LockGame(ctx context.Context, req *v1.LockGameRequest) (*v1.LockGameResponse, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("error getting context information", err)
		return nil, err
	}

	game, err := s.games.GetGame(ctx, req.GameUid)
	if err != nil {
		s.log.Error("failed to get game", info.LoggingContext("error", err)...)
		return &v1.LockGameResponse{
			Success: false,
			GameUid: req.GameUid,
		}, err
	}

	if game == nil {
		s.log.Error("game not found", info.LoggingContext("game", req.GameUid)...)
		return &v1.LockGameResponse{
			Success: false,
			GameUid: req.GameUid,
		}, status.Error(codes.NotFound, "game not found")
	}

	err = s.validateActors(ctx, game.Participants)
	if err != nil {
		s.log.Error("actor is not part of the game", info.LoggingContext("game", req.GameUid)...)
		return &v1.LockGameResponse{
			Success: false,
			GameUid: req.GameUid,
		}, status.Error(codes.PermissionDenied, "actor is not part of the game")
	}

	s.log.Debug("attempting to lock game", info.LoggingContext("game", req.GameUid, "wait", req.Wait)...)
	result, err := s.locks.LockGame(ctx, req)
	if err != nil {
		s.log.Error("failed to lock game", info.LoggingContext("error", err)...)
		return &v1.LockGameResponse{
			Success: false,
			GameUid: req.GameUid,
		}, err
	}

	s.log.Info("game lock resulted", info.LoggingContext("game", req.GameUid, "result", result)...)
	return &v1.LockGameResponse{
		Success: result,
		GameUid: req.GameUid,
	}, nil
}

func (s *defaultGameServer) UnlockGame(ctx context.Context, req *v1.UnlockGameRequest) (*v1.UnlockGameResponse, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("error getting context information", err)
		return nil, err
	}

	game, err := s.games.GetGame(ctx, req.GameUid)
	if err != nil {
		s.log.Error("failed to get game", info.LoggingContext("error", err)...)
		return &v1.UnlockGameResponse{
			Success: false,
			GameUid: req.GameUid,
		}, err
	}
	if game == nil {
		s.log.Error("game not found", info.LoggingContext("game", req.GameUid)...)
		return &v1.UnlockGameResponse{
			Success: false,
			GameUid: req.GameUid,
		}, status.Error(codes.NotFound, "game not found")
	}

	err = s.validateActors(ctx, game.Participants)
	if err != nil {
		s.log.Error("actor is not part of the game", info.LoggingContext("game", req.GameUid)...)
		return &v1.UnlockGameResponse{
			Success: false,
			GameUid: req.GameUid,
		}, status.Error(codes.PermissionDenied, "actor is not part of the game")
	}

	s.log.Debug("attempting to unlock game", info.LoggingContext("game", req.GameUid)...)
	result, err := s.locks.UnlockGame(ctx, req)
	if err != nil {
		s.log.Error("failed to unlock game", info.LoggingContext("error", err)...)
		return &v1.UnlockGameResponse{
			Success: false,
			GameUid: req.GameUid,
		}, err
	}

	s.log.Info("game unlock resulted", info.LoggingContext("game", req.GameUid, "result", result)...)
	return &v1.UnlockGameResponse{
		Success: result,
		GameUid: req.GameUid,
	}, nil
}

func (s *defaultGameServer) EndGame(ctx context.Context, req *v1.EndGameRequest) (*v1.EndGameResponse, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("error getting context information", err)
		return nil, err
	}

	game, err := s.games.GetGame(ctx, req.GameUid)
	if err != nil {
		s.log.Error("failed to get game", info.LoggingContext("error", err)...)
		return &v1.EndGameResponse{
			GameUid: req.GameUid,
		}, err
	}

	if game == nil {
		s.log.Error("game not found", info.LoggingContext("game", req.GameUid)...)
		return &v1.EndGameResponse{
			GameUid: req.GameUid,
		}, status.Error(codes.NotFound, "game not found")
	}

	if !slices.Contains(game.Participants, info.Actor) {
		s.log.Error("actor is not part of the game", info.LoggingContext("game", req.GameUid)...)
		return &v1.EndGameResponse{
			GameUid: req.GameUid,
		}, status.Error(codes.PermissionDenied, "actor is not part of the game")
	}

	s.log.Info("ending game", info.LoggingContext("game", req.GameUid)...)

	game.Completed = true
	err = s.games.SaveGame(ctx, game)
	if err != nil {
		s.log.Error("failed to save game", info.LoggingContext("error", err)...)
		return &v1.EndGameResponse{
			GameUid: req.GameUid,
		}, err
	}

	return &v1.EndGameResponse{
		GameUid: req.GameUid,
	}, nil
}
