package engine

import (
	"context"
	"fmt"
	v1 "overseer/build/go"
	"overseer/common"
	"overseer/storage"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	charm "github.com/charmbracelet/log"
)

type defaultEventBus struct {
	handlers map[EventHandler]EventPredicate
	// TODO make this a games client instead of server to avoid loopback dependence
	games v1.GamesServer
	// TODO make this a games client instead of server to avoid loopback dependence
	users  v1.UsersServer
	events storage.EventStore
	log    *charm.Logger
}

func NewEventBus(handlers []EventHandler, games v1.GamesServer, user v1.UsersServer, events storage.EventStore) EventBus {
	hMap := make(map[EventHandler]EventPredicate)
	for _, h := range handlers {
		hMap[h] = h.Predicate()
	}
	return &defaultEventBus{
		handlers: hMap,
		games:    games,
		users:    user,
		events:   events,
		log:      common.GetLogger("engine.eventbus"),
	}
}

func (b *defaultEventBus) gameExists(ctx context.Context, gameUid string) (bool, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		b.log.Error("failed to get actor from context",
			"error", err,
		)
		return false, err
	}
	if info == nil {
		b.log.Error("actor in context was nil")
		return false, status.Error(codes.Unauthenticated, "actor in context was nil")
	}
	game, err := b.games.GetGame(ctx, &v1.GetGameRequest{GameUid: gameUid})
	if err != nil {
		b.log.Error("failed to get game",
			info.LoggingContext(
				"error", err,
				"game_id", gameUid,
			)...,
		)
		return false, err
	}
	if game == nil {
		b.log.Error("game not found",
			info.LoggingContext(
				"game_id", gameUid,
			)...,
		)
		return false, nil
	}

	err = b.validateActors(ctx, info.Actor, game.Participants)
	if err == nil {
		return true, nil
	}
	b.log.Error("actor not in game",
		info.LoggingContext(
			"game_id", gameUid,
			"actor", info.Actor,
		)...,
	)
	return true, status.Error(codes.Unauthenticated, "actor not in game")
}

func (b *defaultEventBus) validateActors(ctx context.Context, current *v1.Actor, actors []*v1.Actor) error {
	foundCurrent := false
	for _, actor := range actors {
		b.log.Debug("validating actor", "actor", actor.Uid)
		_, err := b.users.GetActor(ctx, &v1.GetActorRequest{
			ActorId: actor.Uid,
		})
		if err != nil {
			return err
		}

		if actor.Uid == current.Uid {
			foundCurrent = true
		}
	}

	if !foundCurrent {
		return status.Error(codes.InvalidArgument, "current actor is not part of the participants")
	}

	return nil
}

func (b *defaultEventBus) Submit(ctx context.Context, event *v1.Event) (<-chan *v1.EventReceipt, error) {
	results := make(chan *v1.EventReceipt, common.GetConfiguration().ChannelBuffer)
	exists, err := b.gameExists(ctx, event.GameUid)
	if err != nil {
		b.log.Error("failed to check if game exists",
			"error", err,
		)
		close(results)
		return results, err
	}
	if !exists {
		b.log.Error("game does not exist",
			"game_id", event.GameUid,
		)
		close(results)
		return results, status.Error(codes.NotFound, "game does not exist")
	}

	if len(b.handlers) == 0 {
		b.log.Error("no handlers registered")
		close(results)
		return results, status.Error(codes.Internal, "no handlers registered")
	}

	info, err := common.GetContextInformation(ctx)
	if err != nil {
		b.log.Error("failed to get actor from context",
			"error", err,
		)
		close(results)
		return results, err
	}
	if info == nil {
		b.log.Error("actor in context was nil")
		close(results)
		return results, status.Error(codes.Unauthenticated, "actor in context was nil")
	}

	asyncCtx, err := common.SetContextInformation(context.Background(), info)
	if err != nil {
		b.log.Error("failed to set actor to context",
			"error", err,
		)
		close(results)
		return results, err
	}

	record, err := b.events.RecordEvent(asyncCtx, event)
	if err != nil {
		b.log.Error("failed to record event",
			"error", err,
		)
		close(results)
		return nil, err
	}
	b.log.Debug("event recorded",
		"game_id", record.GameUid,
		"event_id", record.Uid,
	)
	go b.executeSubmission(asyncCtx, record, results)
	return results, nil
}

func (b *defaultEventBus) executeSubmission(ctx context.Context, event *v1.EventRecord, results chan<- *v1.EventReceipt) {
	info, _ := common.GetContextInformation(ctx)
	for handler, predicate := range b.handlers {
		if eval, err := predicate(ctx, event); err != nil {
			b.log.Error("failed to evaluate predicate",
				"error", err,
			)
			continue
		} else {
			b.log.Debug(
				"predicate evaluated",
				info.LoggingContext(
					"game_id", event.GameUid,
					"handler", handler.Name(),
					"event_id", event.Uid,
					"result", eval,
				)...,
			)
			if eval {
				claimId := common.GenerateRandomStringFromSeed(
					"eventbus",
					event.GameUid,
					handler.Name(),
					fmt.Sprintf("%d", time.Now().UTC().Unix()),
				)
				b.log.Debug("locking game",
					info.LoggingContext(
						"game_id", event.GameUid,
						"handler", handler.Name(),
						"event_id", event.Uid,
						"claim_id", claimId,
					)...)
				lock, err := b.games.LockGame(ctx, &v1.LockGameRequest{
					GameUid:  event.GameUid,
					ClaimUid: claimId,
					Wait:     true,
				})
				if err != nil {
					b.log.Error("failed to lock game",
						info.LoggingContext(
							"error", err,
							"game_id", event.GameUid,
							"handler", handler.Name(),
							"event_id", event.Uid,
						)...,
					)
					err = b.sendErrorReceipt(ctx, &v1.EventReceipt{
						Uid: common.GenerateRandomStringFromSeed(
							"eventbus",
							event.GameUid,
							handler.Name(),
							"error",
							fmt.Sprintf("%d", time.Now().UTC().Unix()),
						),
						GameUid:  event.GameUid,
						EventUid: event.Uid,
						Effect: &v1.EventReceipt_Error{
							Error: &v1.ErrorEffect{
								Message: "failed to lock game",
								Type:    v1.ErrorEffect_INTERNAL,
							},
						},
					}, results)
					if err != nil {
						b.log.Error("failed to send error receipt after failed request to acquire lock",
							info.LoggingContext(
								"error", err,
								"game_id", event.GameUid,
								"handler", handler.Name(),
								"event_id", event.Uid,
							)...,
						)
					}
					close(results)
					return
				}
				if !lock.Success {
					b.log.Error("game is locked",
						info.LoggingContext(
							"game_id", event.GameUid,
							"handler", handler.Name(),
							"event_id", event.Uid,
						)...,
					)
					err = b.sendErrorReceipt(ctx, &v1.EventReceipt{
						Uid: common.GenerateRandomStringFromSeed(
							"eventbus",
							event.GameUid,
							handler.Name(),
							"error",
							fmt.Sprintf("%d", time.Now().UTC().Unix()),
						),
						GameUid:  event.GameUid,
						EventUid: event.Uid,
						Effect: &v1.EventReceipt_Error{
							Error: &v1.ErrorEffect{
								Message: "game is locked cannot process event",
								Type:    v1.ErrorEffect_INTERNAL,
							},
						},
					}, results)
					if err != nil {
						b.log.Error("failed to send error receipt after failed to aquire lock",
							info.LoggingContext(
								"error", err,
								"game_id", event.GameUid,
								"handler", handler.Name(),
								"event_id", event.Uid,
							)...,
						)
					}
					close(results)
					return
				}

				b.log.Debug("handling event",
					info.LoggingContext(
						"game_id", event.GameUid,
						"handler", handler.Name(),
						"event_id", event.Uid,
					)...,
				)
				receipt, err := handler.Handle(ctx, event)
				if err != nil {
					b.log.Error("failed to handle event",
						"error", err,
					)
					err = b.sendErrorReceipt(ctx, &v1.EventReceipt{
						Uid: common.GenerateRandomStringFromSeed(
							"eventbus",
							event.GameUid,
							handler.Name(),
							"error",
							fmt.Sprintf("%d", time.Now().UTC().Unix()),
						),
						GameUid:  event.GameUid,
						EventUid: event.Uid,
						Effect: &v1.EventReceipt_Error{
							Error: &v1.ErrorEffect{
								Message: fmt.Sprintf("handler %s failed: %v", handler.Name(), err),
								Type:    v1.ErrorEffect_INTERNAL,
							},
						},
					}, results)
					if err != nil {
						b.log.Error("failed to send error receipt after handler failed",
							info.LoggingContext(
								"error", err,
								"game_id", event.GameUid,
								"handler", handler.Name(),
								"event_id", event.Uid,
							)...,
						)
					}
					continue
				}

				for r := range receipt {
					b.log.Debug("event handled",
						info.LoggingContext(
							"game_id", event.GameUid,
							"handler", handler.Name(),
							"event_id", event.Uid,
							"receipt_id", r.Uid,
						)...,
					)
					results <- r
				}

				unlock, err := b.games.UnlockGame(ctx, &v1.UnlockGameRequest{
					GameUid:  event.GameUid,
					ClaimUid: claimId,
				})
				if err != nil {
					b.log.Error("failed to unlock game",
						info.LoggingContext(
							"error", err,
							"game_id", event.GameUid,
							"handler", handler.Name(),
							"event_id", event.Uid,
						)...,
					)
					err = b.sendErrorReceipt(ctx, &v1.EventReceipt{
						Uid: common.GenerateRandomStringFromSeed(
							"eventbus",
							event.GameUid,
							handler.Name(),
							"error",
							fmt.Sprintf("%d", time.Now().UTC().Unix()),
						),
						GameUid:  event.GameUid,
						EventUid: event.Uid,
						Effect: &v1.EventReceipt_Error{
							Error: &v1.ErrorEffect{
								Message: "failed to unlock game",
								Type:    v1.ErrorEffect_INTERNAL,
							},
						},
					}, results)
					if err != nil {
						b.log.Error("failed to send error receipt after unlock request failed",
							info.LoggingContext(
								"error", err,
								"game_id", event.GameUid,
								"handler", handler.Name(),
								"event_id", event.Uid,
							)...,
						)
					}
					close(results)
					return
				}
				if !unlock.Success {
					b.log.Error("game is not unlocked",
						info.LoggingContext(
							"game_id", event.GameUid,
							"handler", handler.Name(),
							"event_id", event.Uid,
						)...,
					)
					err = b.sendErrorReceipt(ctx, &v1.EventReceipt{
						Uid: common.GenerateRandomStringFromSeed(
							"eventbus",
							event.GameUid,
							handler.Name(),
							"error",
							fmt.Sprintf("%d", time.Now().UTC().Unix()),
						),
						GameUid:  event.GameUid,
						EventUid: event.Uid,
						Effect: &v1.EventReceipt_Error{
							Error: &v1.ErrorEffect{
								Message: "failed to unlock game",
								Type:    v1.ErrorEffect_INTERNAL,
							},
						},
					}, results)
					if err != nil {
						b.log.Error("failed to send error receipt after failed unlock",
							info.LoggingContext(
								"error", err,
								"game_id", event.GameUid,
								"handler", handler.Name(),
								"event_id", event.Uid,
							)...,
						)
					}
					close(results)
					return
				}
			}
		}
	}
	close(results)
}

func (b *defaultEventBus) sendErrorReceipt(ctx context.Context, receipt *v1.EventReceipt, results chan<- *v1.EventReceipt) error {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		b.log.Error("failed to get actor from context",
			"error", err,
		)
		return err
	}
	err = b.events.RecordReceipt(ctx, receipt)
	if err != nil {
		b.log.Error("failed to record error receipt",
			info.LoggingContext(
				"error", err,
				"game_id", receipt.GameUid,
				"event_id", receipt.EventUid,
				"recept_id", receipt.Uid,
			)...,
		)
		return err
	}

	results <- receipt
	return nil
}
