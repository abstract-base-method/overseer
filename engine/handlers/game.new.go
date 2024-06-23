package handlers

import (
	"context"
	"fmt"
	v1 "overseer/build/go"
	"overseer/common"
	"overseer/engine"
	"overseer/storage"
	"time"

	charm "github.com/charmbracelet/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type newGameHandler struct {
	events storage.EventStore
	games  storage.GameStore
	log    *charm.Logger
}

func NewGameHandler(games storage.GameStore, events storage.EventStore) engine.EventHandler {
	return newGameHandler{
		games:  games,
		events: events,
		log:    common.GetLogger("engine.handler.newgame"),
	}
}

func (h newGameHandler) Name() string {
	return "game.new"
}

func (h newGameHandler) Predicate() engine.EventPredicate {
	return func(ctx context.Context, event *v1.EventRecord) (bool, error) {
		if event == nil {
			return false, status.Error(codes.InvalidArgument, "event is nil")
		}
		if event.GetPayload().GetNewGame() == nil {
			return false, nil
		}
		return true, nil
	}
}

func (h newGameHandler) Handle(ctx context.Context, payload *v1.EventRecord) (<-chan *v1.EventReceipt, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		return nil, err
	}
	h.log.Info("handling new game event", info.LoggingContext()...)

	results := make(chan *v1.EventReceipt, common.GetConfiguration().ChannelBuffer)
	defer close(results)

	game, err := h.games.GetGame(ctx, payload.GetPayload().GetGameUid())
	if err != nil {
		h.log.Error("failed to get game", info.LoggingContext("error", err)...)
		return nil, err
	}
	if game == nil {
		err = status.Error(codes.NotFound, "game not found")
		h.log.Error("failed to get game", info.LoggingContext("error", err)...)
		return nil, err
	}

	if game.GetInitialized() {
		err = status.Error(codes.FailedPrecondition, "game already initialized")
		h.log.Error("failed to initialize game", info.LoggingContext("error", err)...)
		return nil, err
	}

	// step 1: create a map
	// step 2: create an initial codition
	// step 3: mark game as initialized

	game.Initialized = true

	err = h.games.SaveGame(ctx, game)
	if err != nil {
		h.log.Error("failed to save game", info.LoggingContext("error", err)...)
		return nil, err
	}

	message := "game created successfully"

	receipt := v1.EventReceipt{
		Uid: common.GenerateRandomStringFromSeed(
			common.GenerateUniqueId(),
			fmt.Sprintf("%d", time.Now().UTC().Unix()),
			payload.GameUid,
		),
		GameUid:  payload.GetGameUid(),
		EventUid: payload.GetUid(),
		Effect: &v1.EventReceipt_Ack{Ack: &v1.Acknowledgement{
			Message: &message,
		}},
	}

	err = h.events.RecordReceipt(ctx, &receipt)
	if err != nil {
		h.log.Error("failed to record receipt", info.LoggingContext("error", err)...)
		return nil, err
	}

	results <- &receipt

	return results, nil
}
