package tests

import (
	"context"
	"fmt"
	"os"
	v1 "overseer/build/go"
	"overseer/common"
	"overseer/engine"
	"overseer/server"
	"overseer/storage"
	"path"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type EventBusSuite struct {
	db     *gorm.DB
	dbFile string
	suite.Suite
}

func (s *EventBusSuite) SetupTest() {
	s.dbFile = path.Join(os.TempDir(), fmt.Sprintf("test-%s.db", uuid.NewString()))
	db, err := storage.NewSqliteDB(s.dbFile, true)
	s.Require().NoError(err)
	s.db = db
}

func (s *EventBusSuite) TearDownTest() {
	s.Require().NoError(os.Remove(s.dbFile), "failed to remove test database")
}

func TestEventBus(t *testing.T) {
	suite.Run(t, new(EventBusSuite))
}

func (s *EventBusSuite) TestEventBus_EmptyHandlers() {
	userStore := storage.NewSqlUserStore(s.db)
	gamesStore := storage.NewSqlGameStore(s.db, userStore)
	lockStore := storage.NewSqlLockStore(s.db)
	usersSrv := server.NewUserServer(userStore)
	gamesSrv := server.NewGameServer(usersSrv, lockStore, gamesStore)
	eventBus := engine.NewEventBus([]engine.EventHandler{}, gamesSrv, usersSrv, storage.NewSqlEventStore(s.db))
	ctx, err := common.SetContextInformation(context.Background(), &common.OverseerContextInformation{
		User: &v1.User{Uid: "test"},
	})
	s.NoError(err)

	_, err = eventBus.Submit(ctx, &v1.Event{})
	s.Error(err, "without handlers the event bus should always error")
}
