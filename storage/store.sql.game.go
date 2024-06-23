package storage

import (
	"context"
	v1 "overseer/build/go"
	"overseer/common"

	charm "github.com/charmbracelet/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type sqlGameStore struct {
	db    *gorm.DB
	users UserStore
	log   *charm.Logger
}

func NewSqlGameStore(db *gorm.DB, users UserStore) GameStore {
	return &sqlGameStore{
		db:    db,
		users: users,
		log:   common.GetLogger("store.psql.game"),
	}
}

func (s sqlGameStore) CreateGame(ctx context.Context, gameObj *v1.Game) error {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("failed to get context information", "error", err)
		return status.Error(codes.Unauthenticated, "failed to get context information")
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&game{
			ID:        gameObj.Uid,
			Name:      gameObj.Name,
			ActorID:   gameObj.ActiveActor.Uid,
			Completed: gameObj.Completed,
		}).Error; err != nil {
			s.log.Error("failed to create game", "error", err)
			return err
		}

		return nil
	})

	if err != nil {
		s.log.Error("failed to create game", info.LoggingContext("error", err)...)
		return status.Error(codes.Internal, "failed to create game")
	}

	for _, participant := range gameObj.Participants {
		err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Save(&gameParticipant{
				GameID:  gameObj.Uid,
				ActorID: participant.Uid,
			}).Error; err != nil {
				s.log.Error("failed to create game participant", "error", err)
				return err
			}

			return nil
		})
		if err != nil {
			s.log.Error("failed to create game participant", info.LoggingContext("error", err)...)
			return status.Error(codes.Internal, "failed to create game participant")
		}
	}

	return nil
}

func (s sqlGameStore) GetGame(ctx context.Context, uid string) (*v1.Game, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("failed to get context information", "error", err)
		return nil, status.Error(codes.Unauthenticated, "failed to get context information")
	}

	var gameObj game
	err = s.db.WithContext(ctx).Where("id = ?", uid).First(&gameObj).Error
	if err != nil {
		s.log.Error("failed to get game", info.LoggingContext("error", err)...)
		return nil, status.Error(codes.NotFound, "failed to get game")
	}

	var participants []gameParticipant
	err = s.db.WithContext(ctx).Where("game_id = ?", uid).Find(&participants).Error
	if err != nil {
		s.log.Error("failed to get game participants", info.LoggingContext("error", err)...)
		return nil, status.Error(codes.NotFound, "failed to get game participants")
	}

	active, err := s.users.GetActor(ctx, gameObj.ActorID)
	if err != nil {
		s.log.Error("failed to get game active actor", info.LoggingContext("error", err)...)
		return nil, status.Error(codes.NotFound, "failed to get game active actor")
	}
	if active == nil {
		s.log.Error("actor not found for game", info.LoggingContext("game", uid, "actor", gameObj.ActorID)...)
		return nil, status.Error(codes.NotFound, "failed to get game active actor")
	}

	participantsRet := make([]*v1.Actor, 0)
	for _, participant := range participants {
		actor, err := s.users.GetActor(ctx, participant.ActorID)
		if err != nil {
			s.log.Error("failed to get game participant", info.LoggingContext("error", err)...)
			return nil, status.Error(codes.NotFound, "failed to get game participant")
		}
		if actor == nil {
			s.log.Error("actor not found for game", info.LoggingContext("game", uid, "actor", participant.ActorID)...)
			return nil, status.Error(codes.NotFound, "failed to get game participant")
		}
		participantsRet = append(participantsRet, actor)
	}

	gameRet := &v1.Game{
		Uid:          gameObj.ID,
		Name:         gameObj.Name,
		Initialized:  gameObj.Initialized,
		Completed:    gameObj.Completed,
		ActiveActor:  active,
		Participants: participantsRet,
	}

	return gameRet, nil
}

func (s sqlGameStore) SaveGame(ctx context.Context, gameObj *v1.Game) error {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("failed to get context information", "error", err)
		return status.Error(codes.Unauthenticated, "failed to get context information")
	}

	gameRecord := &game{
		ID:          gameObj.Uid,
		Name:        gameObj.Name,
		ActorID:     gameObj.ActiveActor.Uid,
		Initialized: gameObj.Initialized,
		Completed:   gameObj.Completed,
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(gameRecord).Error; err != nil {
			s.log.Error("failed to save game", "error", err)
			return err
		}

		return nil
	})

	if err != nil {
		s.log.Error("failed to save game", info.LoggingContext("error", err)...)
		return status.Error(codes.Internal, "failed to save game")
	}

	for _, p := range gameObj.Participants {
		err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Save(&gameParticipant{
				GameID:  gameObj.Uid,
				ActorID: p.Uid,
			}).Error; err != nil {
				s.log.Error("failed to save game participant", "error", err)
				return err
			}

			return nil
		})
		if err != nil {
			s.log.Error("failed to save game participant", info.LoggingContext("error", err)...)
			return status.Error(codes.Internal, "failed to save game participant")
		}
	}

	return nil
}
