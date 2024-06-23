package scenarios

import (
	"context"
	"fmt"
	"os"
	v1 "overseer/build/go"
	"overseer/common"
	"overseer/engine"
	"overseer/engine/handlers"
	"overseer/generative"
	"overseer/generative/ollama"
	"overseer/server"
	"overseer/storage"
	"path"
	"testing"
	"text/template"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type mapServerTestSuite struct {
	db     *gorm.DB
	dbFile string
	suite.Suite
}

func (s *mapServerTestSuite) SetupTest() {
	s.dbFile = path.Join(os.TempDir(), fmt.Sprintf("test-%s.db", uuid.NewString()))
	db, err := storage.NewSqliteDB(s.dbFile, true)
	s.Require().NoError(err)
	s.db = db
}

func (s *mapServerTestSuite) TearDownTest() {
	s.Require().NoError(os.Remove(s.dbFile), "failed to remove test database")
}

func TestMapServerSuite(t *testing.T) {
	suite.Run(t, new(mapServerTestSuite))
}

func (s *mapServerTestSuite) TestMapServer_CreateMap() {
	mockOllama := new(MockOllamaClient)
	mockTemplatingClient := new(MockTemplatingClient)
	mapSvc, err := generative.NewOllamaMapGenerationService(mockTemplatingClient, mockOllama)
	s.Require().NoError(err)
	mapStore := storage.NewSqlMapStore(s.db)
	mapServer := server.NewMapServer(mapStore, mapSvc)
	userStore := storage.NewSqlUserStore(s.db)
	gamesStore := storage.NewSqlGameStore(s.db, userStore)
	eventStore := storage.NewSqlEventStore(s.db)
	lockStore := storage.NewSqlLockStore(s.db)
	usersSrv := server.NewUserServer(userStore)
	gamesSrv := server.NewGameServer(usersSrv, lockStore, gamesStore)
	eventBus := engine.NewEventBus(
		[]engine.EventHandler{
			handlers.NewGameHandler(gamesStore, eventStore),
		},
		gamesSrv,
		usersSrv,
		eventStore,
	)
	eventSrv := server.NewEventServer(eventBus)
	user := &v1.User{
		Uid: "test",
	}
	ctx, _ := common.SetContextInformation(context.Background(), &common.OverseerContextInformation{
		User:  user,
		Actor: nil,
	})

	// mocks
	tmpl := template.Must(template.New("mock").Parse("mock"))
	mockTemplatingClient.On("InternalLoreTemplate").Return(tmpl)
	mockTemplatingClient.On("PublicLoreTemplate").Return(tmpl)
	mockTemplatingClient.On("CoordinateLoreTemplate").Return(tmpl)
	fakeChan := make(chan ollama.GenerateResponse)
	close(fakeChan)
	mockOllama.On("Generate", mock.Anything, mock.Anything).Return(fakeChan, nil)

	_, err = usersSrv.RegisterUser(ctx, user)
	s.NoError(err, "error should be nil on creating user")

	actor, err := usersSrv.RegisterActor(ctx, &v1.RegisterActorRequest{
		UserId: user.Uid,
		Source: v1.Actor_APP_DISCORD,
	})
	s.NoError(err, "error should be nil on creating actor")

	ctx, _ = common.SetContextInformation(ctx, &common.OverseerContextInformation{
		User:  user,
		Actor: actor,
	})

	game, err := gamesSrv.CreateGame(ctx, &v1.CreateGameRequest{
		Name:  "test game",
		Theme: v1.GameTheme_DEFAULT,
		Participants: []*v1.Actor{
			actor,
		},
	})
	s.NoError(err, "error should be nil")
	s.NotNil(game, "game should not be nil")

	game, err = gamesSrv.GetGame(ctx, &v1.GetGameRequest{
		GameUid: game.Uid,
	})
	s.NoError(err, "error should be nil")
	s.NotNil(game, "game should not be nil")
	s.False(game.Initialized, "game should not be initialized")
	s.False(game.Completed, "game should not be completed")

	receipts, err := eventSrv.Submit(
		ctx,
		&v1.Event{
			GameUid: game.Uid,
			Actor:   actor,
			Origin: &v1.Event_Discord{
				Discord: &v1.EventOriginDiscord{
					Guild:   "test",
					Channel: "test",
				},
			},
			Payload: &v1.Event_NewGame{
				NewGame: &v1.NewGameEvent{
					Name: "test game",
					Participants: []*v1.Actor{
						actor,
					},
				},
			},
		},
	)

	s.NoError(err, "error should be nil")
	s.Len(receipts.Receipts, 1, "new game receipt should be received")
	s.NotNil(receipts.Receipts[0].GetAck(), "receipt should contain new game event: ", receipts.Receipts[0])

	game, err = gamesSrv.GetGame(ctx, &v1.GetGameRequest{
		GameUid: game.Uid,
	})
	s.NoError(err, "error should be nil")
	s.NotNil(game, "game should not be nil")
	s.True(game.Initialized, "game should be initialized")
	s.False(game.Completed, "game should not be completed")

	_, err = mapServer.CreateMap(ctx, &v1.CreateMapRequest{
		GameUid:                game.Uid,
		MaxX:                   10,
		MaxY:                   10,
		Theme:                  game.Theme,
		DifficultTerrainChance: 0.3,
		SpriteDensity:          0.2,
		Actors:                 []*v1.Actor{actor},
	})
	s.NoError(err, "error should be nil")
}
