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

type sqlMapStore struct {
	db  *gorm.DB
	log *charm.Logger
}

func NewSqlMapStore(db *gorm.DB) MapStore {
	return &sqlMapStore{
		db:  db,
		log: common.GetLogger("store.psql.games"),
	}
}

func (s *sqlMapStore) CreateMap(ctx context.Context, req *v1.CreateMapRequest) (*v1.Map, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		return nil, err
	}

	s.log.Info("persisting new map", info.LoggingContext(
		"game", req.GameUid,
		"maxX", req.MaxX,
		"maxY", req.MaxY,
		"theme", req.Theme,
	)...)

	newMap := &v1.Map{
		Uid:     common.GenerateUniqueId(),
		GameUid: req.GameUid,
		Name:    req.Name,
		MaxX:    req.MaxX,
		MaxY:    req.MaxY,
	}
	record, err := MapRecordFromProto(newMap)
	if err != nil {
		s.log.Error("failed to make new map record", info.LoggingContext(
			"error", err,
			"game", req.GameUid,
		)...)
		return nil, err
	}

	err = s.db.Create(&record).Error
	if err != nil {
		s.log.Error("failed to create map",
			"error", err,
			"game", req.GameUid,
			"actor", info.Actor.Uid,
			"source", info.Actor.Source,
			"sourceIdentity", info.Actor.SourceIdentity,
			"maxX", req.MaxX,
			"maxY", req.MaxY,
			"theme", req.Theme,
		)
		return nil, err
	}

	return newMap, nil
}

func (s *sqlMapStore) GetMap(ctx context.Context, uid string) (*v1.Map, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		return nil, err
	}

	s.log.Debug("fetching map", info.LoggingContext(
		"map", uid,
	)...)
	var record gameMap
	err = s.db.Where("id = ?", uid).First(&record).Error
	if err != nil {
		s.log.Error("failed to fetch map", info.LoggingContext(
			"error", err,
			"map", uid,
		)...)
		return nil, err
	}

	pb, err := record.ToProto()
	if err != nil {
		s.log.Error("failed to convert map to proto", info.LoggingContext(
			"error", err,
			"map", uid,
		)...)
		return nil, err
	}

	return pb, nil
}

func (s *sqlMapStore) CreateCoordinate(ctx context.Context, coordinate *v1.MapCoordinateDetail) error {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		return err
	}

	s.log.Info("creating new map coordinate", info.LoggingContext(
		"game", coordinate.GameUid,
		"map", coordinate.MapUid,
		"uid", coordinate.Uid,
		"x", coordinate.Position.X,
		"y", coordinate.Position.Y,
	)...)
	record, err := MapCoordinateRecordFromProto(coordinate)
	if err != nil {
		s.log.Error("failed to make new map coordinate record", info.LoggingContext(
			"error", err,
			"game", coordinate.GameUid,
			"map", coordinate.MapUid,
			"coordinate", coordinate.Uid,
		)...)
		return status.Error(codes.Internal, "failed to make new map coordinate record")
	}

	err = s.db.Create(&record).Error
	if err != nil {
		s.log.Error("failed to create map coordinate", info.LoggingContext(
			"error", err,
			"game", coordinate.GameUid,
			"map", coordinate.MapUid,
			"coordinate", coordinate.Uid,
			"x", coordinate.Position.X,
			"y", coordinate.Position.Y,
		)...)
		return err
	}

	return nil
}

func (s *sqlMapStore) GetCoordinates(ctx context.Context, mapId string) ([]*v1.MapCoordinateDetail, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		return nil, err
	}

	s.log.Debug("fetching map coordinates", info.LoggingContext(
		"map", mapId,
	)...)
	var records []mapCoordinate
	err = s.db.Where(mapCoordinate{GameMapID: mapId}, "game_map_id").Find(&records).Error
	if err != nil {
		s.log.Error("failed to fetch map coordinates", info.LoggingContext(
			"error", err,
			"map", mapId,
		)...)
		return nil, err
	}

	s.log.Debug("fetched map coordinates", info.LoggingContext(
		"map", mapId,
		"count", len(records),
	)...)

	var pb []*v1.MapCoordinateDetail
	for _, record := range records {
		pbRecord, err := record.ToProto()
		if err != nil {
			s.log.Error("failed to convert map coordinate to proto", info.LoggingContext(
				"error", err,
				"map", mapId,
				"coordinate", record.ID,
			)...)
			return nil, err
		}

		pb = append(pb, pbRecord)
	}

	return pb, nil
}

func (s *sqlMapStore) UpdateCoordinate(ctx context.Context, coordinate *v1.MapCoordinateDetail) error {
	return status.Error(codes.Unimplemented, "method UpdateSprite not implemented")
}

func (s *sqlMapStore) GetCoordinate(ctx context.Context, gameId string, mapId string, x int64, y int64) (*v1.MapCoordinateDetail, error) {
	return nil, status.Error(codes.Unimplemented, "method GetCoordinate not implemented")
}
