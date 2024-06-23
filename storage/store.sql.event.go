package storage

import (
	"context"
	"fmt"
	v1 "overseer/build/go"
	"overseer/common"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	charm "github.com/charmbracelet/log"
	"gorm.io/gorm"
)

type sqlEventStore struct {
	db  *gorm.DB
	log *charm.Logger
}

func NewSqlEventStore(db *gorm.DB) EventStore {
	return &sqlEventStore{
		db:  db,
		log: common.GetLogger("store.psql.events"),
	}
}

func (s *sqlEventStore) RecordEvent(ctx context.Context, event *v1.Event) (*v1.EventRecord, error) {
	recordId := common.GenerateRandomStringFromSeed(
		event.Actor.Uid,
		fmt.Sprintf("%d", time.Now().UTC().Unix()),
	)

	origin, err := getEventOrigin(event)
	if err != nil {
		return nil, err
	}

	pType, err := getEventType(event)
	if err != nil {
		return nil, err
	}

	raw, err := proto.Marshal(event)
	if err != nil {
		s.log.Error("failed to marshal event", "error", err)
		return nil, err
	}

	record := &v1.EventRecord{
		Uid:      recordId,
		GameUid:  event.GameUid,
		Payload:  event,
		Receipts: make([]*v1.EventReceipt, 0),
	}
	evt := &eventRow{
		ID:          recordId,
		GameID:      event.GameUid,
		ActorID:     event.Actor.Uid,
		Origin:      origin,
		PayloadType: pType,
		Raw:         raw,
	}

	db := s.db.WithContext(ctx)

	err = db.Transaction(func(tx *gorm.DB) error {
		if txErr := tx.Create(evt).Error; txErr != nil {
			return txErr
		}
		return nil
	})

	if err != nil {
		s.log.Error("failed to record eventRow", "error", err)
		return nil, err
	}

	return record, nil
}

func (s *sqlEventStore) RecordReceipt(ctx context.Context, receipt *v1.EventReceipt) error {
	if exists, err := s.receiptExists(ctx, receipt.Uid); err != nil {
		s.log.Error("failed to check receipt existence",
			"error", err,
			"receipt_id", receipt.Uid,
			"event_id", receipt.EventUid,
		)
		return status.Error(codes.Internal, fmt.Sprintf("failed to check receipt existence: %s", err))
	} else if exists {
		s.log.Warn("receipt already exists",
			"receipt_id", receipt.Uid,
			"event_id", receipt.EventUid,
		)
		return status.Error(codes.AlreadyExists, fmt.Sprintf("receipt already exists: %s", receipt.Uid))
	}

	eType, err := getEffectType(receipt)
	if err != nil {
		s.log.Error("failed to get effect type",
			"error", err,
			"receipt_id", receipt.Uid,
			"event_id", receipt.EventUid,
		)
		return err
	}

	raw, err := proto.Marshal(receipt)
	if err != nil {
		s.log.Error("failed to marshal receipt",
			"error", err,
			"receipt_id", receipt.Uid,
			"event_id", receipt.EventUid,
		)
		return status.Error(codes.Internal, fmt.Sprintf("failed to marshal receipt: %s", err))
	}

	record := &eventReceipt{
		ID:         receipt.Uid,
		GameID:     receipt.GameUid,
		EventID:    receipt.EventUid,
		EffectType: eType,
		Raw:        raw,
	}

	s.log.Debug("recording receipt",
		"receipt_id", receipt.Uid,
		"event_id", receipt.EventUid,
		"game_id", receipt.GameUid,
	)

	db := s.db.WithContext(ctx)

	err = db.Transaction(func(tx *gorm.DB) error {
		if txErr := tx.Create(record).Error; txErr != nil {
			return txErr
		}
		return nil
	})
	if err != nil {
		s.log.Error("failed to record receipt",
			"error", err,
			"receipt_id", receipt.Uid,
			"event_id", receipt.EventUid,
		)
		return status.Error(codes.Internal, fmt.Sprintf("failed to record receipt: %s", err))
	}

	return nil
}

func (s *sqlEventStore) receiptExists(ctx context.Context, id string) (bool, error) {
	var count int64

	db := s.db.WithContext(ctx)

	err := db.Model(&eventReceipt{}).Where("id = ?", id).Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *sqlEventStore) GetEvent(ctx context.Context, id string) (*v1.EventRecord, error) {
	return nil, status.Error(codes.Unimplemented, "method GetEvent not implemented")
}
