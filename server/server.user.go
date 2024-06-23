package server

import (
	"context"
	"overseer/auth"
	v1 "overseer/build/go"
	"overseer/common"
	"overseer/storage"

	charm "github.com/charmbracelet/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type defaultUserServer struct {
	users storage.UserStore
	log   *charm.Logger
	v1.UnimplementedUsersServer
}

func NewUserServer(users storage.UserStore) v1.UsersServer {
	return &defaultUserServer{
		users: users,
		log:   common.GetLogger("server.user"),
	}
}

func (s *defaultUserServer) RegisterUser(ctx context.Context, user *v1.User) (*v1.User, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("failed to get context information", "error", err)
		return nil, status.Error(codes.Unauthenticated, "failed to get context information")
	}

	if user.Uid == auth.SystemUserId {
		s.log.Error("user id is reserved", info.LoggingContext("uid", user.Uid)...)
		return nil, status.Error(codes.InvalidArgument, "user id is reserved")
	}

	err = s.users.UpsertUser(ctx, user)
	if err != nil {
		s.log.Error("failed to create user", info.LoggingContext("error", err)...)
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	return user, nil
}

func (s *defaultUserServer) RegisterActor(ctx context.Context, req *v1.RegisterActorRequest) (*v1.Actor, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("failed to get context information", "error", err)
		return nil, status.Error(codes.Unauthenticated, "failed to get context information")
	}

	if req.GetUserId() == auth.SystemUserId {
		s.log.Error("user id is reserved", info.LoggingContext("uid", req.GetUserId())...)
		return nil, status.Error(codes.InvalidArgument, "user id is reserved and cannot register new actors")
	}

	actor := &v1.Actor{
		Uid: common.GenerateRandomStringFromSeed(
			"actor",
			req.GetUserId(),
			req.GetSourceIdentity(),
			req.GetSource().String(),
			common.GenerateUniqueId(),
		),
		SourceIdentity: req.GetSourceIdentity(),
		Metadata:       req.GetMetadata(),
	}

	err = s.users.UpsertActor(ctx, req.GetUserId(), actor)
	if err != nil {
		s.log.Error("failed to create actor", info.LoggingContext("error", err)...)
		return nil, status.Error(codes.Internal, "failed to create actor")
	}

	return actor, nil
}

func (s *defaultUserServer) GetUser(ctx context.Context, actor *v1.Actor) (*v1.User, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("failed to get context information", "error", err)
		return nil, status.Error(codes.Unauthenticated, "failed to get context information")
	}

	user, err := s.users.GetUserForActor(ctx, actor.GetUid())
	if err != nil {
		s.log.Error("failed to get user", info.LoggingContext("error", err)...)
		return nil, status.Error(codes.Internal, "failed to get user")
	}
	if user == nil {
		s.log.Error("user not found", info.LoggingContext()...)
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return user, nil
}

func (s *defaultUserServer) GetActor(ctx context.Context, req *v1.GetActorRequest) (*v1.Actor, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("failed to get context information", "error", err)
		return nil, status.Error(codes.Unauthenticated, "failed to get context information")
	}

	actor, err := s.users.GetActor(ctx, req.GetActorId())
	if err != nil {
		s.log.Error("failed to get actor", info.LoggingContext("error", err)...)
		return nil, status.Error(codes.Internal, "failed to get actor")
	}
	if actor == nil {
		s.log.Error("actor not found", info.LoggingContext()...)
		return nil, status.Error(codes.NotFound, "actor not found")
	}

	return actor, nil
}

func (s *defaultUserServer) GetActors(ctx context.Context, user *v1.User) (*v1.Actors, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("failed to get context information", "error", err)
		return nil, status.Error(codes.Unauthenticated, "failed to get context information")
	}

	actors, err := s.users.GetActors(ctx, user.GetUid())
	if err != nil {
		s.log.Error("failed to get actors", info.LoggingContext("error", err)...)
		return nil, status.Error(codes.Internal, "failed to get actors")
	}

	return &v1.Actors{Actors: actors}, nil
}
