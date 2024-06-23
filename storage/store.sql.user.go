package storage

import (
	"context"
	v1 "overseer/build/go"
	"overseer/common"

	charm "github.com/charmbracelet/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

type sqlUserStore struct {
	db  *gorm.DB
	log *charm.Logger
}

func NewSqlUserStore(db *gorm.DB) UserStore {
	return &sqlUserStore{
		db:  db,
		log: common.GetLogger("store.psql.user"),
	}
}

func (s *sqlUserStore) UpsertUser(ctx context.Context, userMsg *v1.User) error {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("failed to get context information", "error", err)
		return status.Error(codes.Unauthenticated, "failed to get context information")
	}

	s.log.Info("upserting user", info.LoggingContext("user", userMsg)...)

	userRow := &user{
		ID: userMsg.GetUid(),
	}

	err = s.db.Save(&userRow).Error
	if err != nil {
		s.log.Error("failed to create user", info.LoggingContext("error", err)...)
		return status.Error(codes.Internal, "failed to create user")
	}

	return nil
}

func (s *sqlUserStore) UpsertActor(ctx context.Context, userId string, actorMsg *v1.Actor) error {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("failed to get context information", "error", err)
		return status.Error(codes.Unauthenticated, "failed to get context information")
	}

	s.log.Info("upserting actor", info.LoggingContext("actor", actorMsg)...)
	user, err := s.GetUser(ctx, userId)
	if err != nil {
		s.log.Error("failed to get user", info.LoggingContext("error", err)...)
		return status.Error(codes.Internal, "failed to get user")
	}
	if user == nil {
		return status.Error(codes.NotFound, "user not found")
	}

	raw, err := proto.Marshal(actorMsg)
	if err != nil {
		s.log.Error("failed to marshal actor", info.LoggingContext("error", err)...)
		return status.Error(codes.Internal, "failed to marshal actor")
	}

	row := &actor{
		ID:             actorMsg.GetUid(),
		SourceIdentity: actorMsg.GetSourceIdentity(),
		Source:         actorMsg.GetSource(),
		Raw:            raw,
	}

	err = s.db.Save(&row).Error
	if err != nil {
		s.log.Error("failed to create actor", info.LoggingContext("error", err)...)
		return status.Error(codes.Internal, "failed to create actor")
	}

	return nil
}

func (s *sqlUserStore) GetUser(ctx context.Context, id string) (*v1.User, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("failed to get context information", "error", err)
		return nil, status.Error(codes.Unauthenticated, "failed to get context information")
	}

	s.log.Info("getting user", info.LoggingContext("id", id)...)

	row := &user{}
	err = s.db.Where("id = ?", id).First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}

		s.log.Error("failed to get user", info.LoggingContext("error", err)...)
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	return &v1.User{
		Uid: row.ID,
	}, nil
}

func (s *sqlUserStore) GetUserForActor(ctx context.Context, actorID string) (*v1.User, error) {
	return nil, status.Error(codes.Unimplemented, "GetUserForActor not implemented")
}

func (s *sqlUserStore) GetActor(ctx context.Context, id string) (*v1.Actor, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("failed to get context information", "error", err)
		return nil, status.Error(codes.Unauthenticated, "failed to get context information")
	}

	s.log.Info("getting actor", info.LoggingContext("id", id)...)

	row := &actor{}
	err = s.db.Where("id = ?", id).First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "actor not found")
		}

		s.log.Error("failed to get actor", info.LoggingContext("error", err)...)
		return nil, status.Error(codes.Internal, "failed to get actor")
	}

	actorMsg := &v1.Actor{}
	err = proto.Unmarshal(row.Raw, actorMsg)
	if err != nil {
		s.log.Error("failed to unmarshal actor", info.LoggingContext("error", err)...)
		return nil, status.Error(codes.Internal, "failed to unmarshal actor")
	}

	return actorMsg, nil
}

func (s *sqlUserStore) GetActors(ctx context.Context, id string) ([]*v1.Actor, error) {
	return nil, status.Error(codes.Unimplemented, "GetActors not implemented")
}

func (s *sqlUserStore) DeleteUser(ctx context.Context, id string) error {
	return status.Error(codes.Unimplemented, "DeleteUser not implemented")
}

func (s *sqlUserStore) DeleteActor(ctx context.Context, id string) error {
	return status.Error(codes.Unimplemented, "DeleteActor not implemented")
}
