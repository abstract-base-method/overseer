package storage

import (
	"context"
	v1 "overseer/build/go"
	"overseer/common"
	"time"

	charm "github.com/charmbracelet/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// this sucks plz replace with something robust
type sqlLockStore struct {
	db  *gorm.DB
	log *charm.Logger
}

func NewSqlLockStore(db *gorm.DB) LockStore {
	return &sqlLockStore{
		db:  db,
		log: common.GetLogger("store.stub.lock"),
	}
}

func (s *sqlLockStore) LockGame(ctx context.Context, request *v1.LockGameRequest) (bool, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("failed to get context information", "error", err)
		return false, err
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&lock{
			ID:        request.ClaimUid,
			GameID:    request.GameUid,
			Completed: false,
		}).Error; err != nil {
			s.log.Error("failed to create lock", info.LoggingContext("error", err)...)
			return err
		}
		return nil
	})

	if err != nil {
		s.log.Error("failed to create lock", info.LoggingContext("error", err)...)
		return false, err
	}

	var pending int64
	var myLock lock

	err = s.db.Where(&lock{GameID: request.GameUid, ID: request.ClaimUid}).First(&myLock).Error
	if err != nil {
		s.log.Error("failed to get lock", info.LoggingContext("error", err)...)
		return false, err
	}

	pending, err = s.getPendingLocks(ctx, request.GameUid, myLock.CreatedAt)
	if err != nil {
		s.log.Error("failed to get pending locks", info.LoggingContext("error", err)...)
		return false, err
	}

	waiting := pending > 0

	if waiting && !request.Wait {
		s.log.Warn("game is locked", info.LoggingContext("game", request.GameUid)...)
		return false, nil
	}

	for waiting {
		select {
		case <-ctx.Done():
			s.log.Warn("context cancelled", info.LoggingContext("game", request.GameUid)...)
			return false, ctx.Err()
		default:
			pending, err = s.getPendingLocks(ctx, request.GameUid, myLock.CreatedAt)
			if err != nil {
				s.log.Error("failed to get pending locks", info.LoggingContext("error", err)...)
				return false, err
			}

			if pending == 0 {
				waiting = false
			}
		}

		if waiting {
			s.log.Info("waiting for lock", info.LoggingContext("game", request.GameUid, "claim", request.ClaimUid, "pending", pending)...)
			time.Sleep(100 * time.Second)
		} else {
			break
		}
	}

	s.log.Info("locking game", info.LoggingContext("game", request.GameUid, "claim", request.ClaimUid, "period", myLock.CreatedAt)...)

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&lock{}).Where("game_id = ? AND id = ?", request.GameUid, request.ClaimUid).Update("locked", true).Error; err != nil {
			s.log.Error("failed to complete lock", info.LoggingContext("error", err)...)
			return err
		}
		return nil
	})
	if err != nil {
		s.log.Error("failed to complete lock", info.LoggingContext("error", err, "game", request.GameUid, "claim", request.ClaimUid)...)
		return false, err
	}

	return true, nil
}

func (s *sqlLockStore) getPendingLocks(ctx context.Context, gameId string, period time.Time) (int64, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("failed to get context information", "error", err)
		return 0, err
	}

	s.log.Debug("getting pending locks", info.LoggingContext("game", gameId, "period", period)...)
	var count int64
	err = s.db.Model(&lock{}).Where("game_id = ? AND created_at < ? AND completed = false", gameId, period).Count(&count).Error

	if err != nil {
		s.log.Error("failed to get pending locks", info.LoggingContext("error", err, "game", gameId)...)
		return 0, status.Error(codes.Internal, "failed to get pending locks")
	}

	return count, nil
}

func (s *sqlLockStore) UnlockGame(ctx context.Context, request *v1.UnlockGameRequest) (bool, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("failed to get context information", "error", err)
		return false, err
	}

	s.log.Debug("unlocking game", info.LoggingContext("game", request.GameUid, "claim", request.ClaimUid)...)

	db := s.db.WithContext(ctx)

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&lock{
			ID:        request.ClaimUid,
			GameID:    request.GameUid,
			Completed: true,
			Locked:    false,
		}).Error; err != nil {
			s.log.Error("failed to unlock game", info.LoggingContext("error", err, "game", request.GameUid, "claim", request.ClaimUid)...)
			return err
		}
		return nil
	})
	if err != nil {
		s.log.Error("failed to unlock game", info.LoggingContext("error", err, "game", request.GameUid, "claim", request.ClaimUid)...)
		return false, err
	}

	s.log.Info("unlocked game", info.LoggingContext("game", request.GameUid, "claim", request.ClaimUid)...)

	return true, nil
}
