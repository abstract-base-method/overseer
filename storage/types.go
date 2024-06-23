package storage

import (
	"database/sql/driver"
	"errors"
	"fmt"
	v1 "overseer/build/go"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

type actor struct {
	gorm.Model
	ID             string
	SourceIdentity string
	Source         v1.Actor_Source
	Raw            []byte
}

type user struct {
	gorm.Model
	ID string
}

type game struct {
	gorm.Model
	ID          string
	Name        string
	ActorID     string
	Initialized bool
	Completed   bool
	Raw         []byte
}

type gameParticipant struct {
	gorm.Model
	GameID  string
	ActorID string
}

type eventRow struct {
	gorm.Model
	ID          string
	GameID      string
	ActorID     string
	Origin      eventOrigin `gorm:"type:text"`
	PayloadType payloadType `gorm:"type:text"`
	Raw         []byte
}

type eventOrigin string

const (
	eventOriginSystem  eventOrigin = "system"
	eventOriginDiscord eventOrigin = "discord"
)

func getEventOrigin(event *v1.Event) (eventOrigin, error) {
	switch event.Origin.(type) {
	case *v1.Event_Discord:
		return eventOriginDiscord, nil
	case *v1.Event_System:
		return eventOriginSystem, nil
	default:
		return "", status.Error(codes.NotFound, fmt.Sprintf("unknown event origin: %T", event.Origin))
	}
}

func (o eventOrigin) String() string {
	return string(o)
}

func (o *eventOrigin) Scan(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return errors.New("failed to scan eventRow origin")
	}
	*o = eventOrigin(str)
	return nil
}

func (o eventOrigin) Value() (driver.Value, error) {
	return string(o), nil
}

type payloadType string

const (
	payloadTypeNewGame     payloadType = "new_game"
	payloadTypeInteraction payloadType = "interaction"
)

func getEventType(event *v1.Event) (payloadType, error) {
	switch event.Payload.(type) {
	case *v1.Event_NewGame:
		return payloadTypeNewGame, nil
	case *v1.Event_Interaction:
		return payloadTypeInteraction, nil
	default:
		return "", status.Error(codes.NotFound, fmt.Sprintf("unknown event type: %T", event.Payload))
	}
}

func (t payloadType) String() string {
	return string(t)
}

func (t *payloadType) Scan(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return errors.New("failed to scan payload type")
	}
	*t = payloadType(str)
	return nil
}

func (t payloadType) Value() (driver.Value, error) {
	return string(t), nil
}

type recieptEffectType string

const (
	receptError       recieptEffectType = "error"
	receptAcknowledge recieptEffectType = "acknowledge"
	recieptGameState  recieptEffectType = "game_state"
	receptUtterance   recieptEffectType = "utterance"
)

type eventReceipt struct {
	gorm.Model
	ID         string
	EventID    string
	GameID     string
	EffectType recieptEffectType `gorm:"type:text"`
	Raw        []byte
}

func getEffectType(receipt *v1.EventReceipt) (recieptEffectType, error) {
	switch receipt.Effect.(type) {
	case *v1.EventReceipt_Error:
		return receptError, nil
	case *v1.EventReceipt_Ack:
		return receptAcknowledge, nil
	case *v1.EventReceipt_GameState:
		return recieptGameState, nil
	case *v1.EventReceipt_Utterance:
		return receptUtterance, nil
	default:
		return "", status.Error(codes.NotFound, fmt.Sprintf("unknown receipt effect type: %T", receipt.Effect))
	}
}

func (t recieptEffectType) String() string {
	return string(t)
}

func (t *recieptEffectType) Scan(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return errors.New("failed to scan receipt effect type")
	}
	*t = recieptEffectType(str)
	return nil
}

func (t recieptEffectType) Value() (driver.Value, error) {
	return string(t), nil
}

type lock struct {
	gorm.Model
	ID        string
	GameID    string
	Locked    bool
	Completed bool
}

type gameMap struct {
	gorm.Model
	ID     string
	GameID string
	Name   string
	XMax   int64
	YMax   int64
	Raw    []byte
}

func (m *gameMap) ToProto() (*v1.Map, error) {
	var pb v1.Map
	err := proto.Unmarshal(m.Raw, &pb)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to unmarshal map: %v", err))
	}
	return &pb, nil
}

func MapRecordFromProto(src *v1.Map) (*gameMap, error) {
	raw, err := proto.Marshal(src)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to marshal map: %v", err))
	}
	return &gameMap{
		ID:     src.Uid,
		GameID: src.GameUid,
		Name:   src.Name,
		XMax:   src.MaxX,
		YMax:   src.MaxY,
		Raw:    raw,
	}, nil
}

type mapCoordinate struct {
	gorm.Model
	ID               string
	GameID           string
	GameMapID        string
	X                int64
	Y                int64
	Type             string
	DifficultTerrain bool
	Lore             string
	Raw              []byte
}

func MapCoordinateRecordFromProto(src *v1.MapCoordinateDetail) (*mapCoordinate, error) {
	pb, err := proto.Marshal(src)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to marshal map coordinate: %v", err))
	}

	return &mapCoordinate{
		ID:               src.Uid,
		GameID:           src.GameUid,
		GameMapID:        src.MapUid,
		X:                src.Position.X,
		Y:                src.Position.Y,
		Type:             v1.MapCoordinateDetail_CoordinateType_name[int32(src.Type)],
		DifficultTerrain: src.DifficultTerrain,
		Lore:             src.Lore,
		Raw:              pb,
	}, nil
}

func (c *mapCoordinate) ToProto() (*v1.MapCoordinateDetail, error) {
	var pb v1.MapCoordinateDetail
	err := proto.Unmarshal(c.Raw, &pb)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to unmarshal map coordinate: %v", err))
	}

	return &pb, nil
}
