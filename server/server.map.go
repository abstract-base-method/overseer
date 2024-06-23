package server

import (
	"context"
	"fmt"
	"math/rand"
	v1 "overseer/build/go"
	"overseer/common"
	"overseer/generative"
	"overseer/storage"
	"time"

	charm "github.com/charmbracelet/log"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type defaultMapServer struct {
	mapGenerator generative.MapGenerationService
	mapsDb       storage.MapStore
	log          *charm.Logger
	v1.UnimplementedMapsServer
}

type cardinalDirection string

const (
	north cardinalDirection = "north"
	south cardinalDirection = "south"
	east  cardinalDirection = "east"
)

func NewMapServer(mapsDb storage.MapStore, mapGenerator generative.MapGenerationService) v1.MapsServer {
	return &defaultMapServer{
		mapsDb:       mapsDb,
		mapGenerator: mapGenerator,
		log:          common.GetLogger("server.map"),
	}
}

func (s *defaultMapServer) CreateMap(ctx context.Context, req *v1.CreateMapRequest) (*v1.Map, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		return nil, err
	}

	err = s.validateCreateMapRequest(req)
	if err != nil {
		s.log.Warn("map creation request invalid", info.LoggingContext("error", err)...)
		return nil, err
	}

	generationStartTime := time.Now()
	newMap, err := s.mapsDb.CreateMap(ctx, req)
	if err != nil {
		s.log.Error("failed to persist map", info.LoggingContext("error", err)...)
		return nil, err
	}
	s.log.Info("creating new map", info.LoggingContext(
		"game", req.GameUid,
		"map", newMap.Uid,
	)...)

	max_positive_y := req.MaxY
	max_negative_y := req.MaxY * -1
	max_positive_x := req.MaxX
	max_negative_x := req.MaxX * -1
	// total coordinates are the positive and negative values of the max x and y multiplied by 2 (for negative and positive) PLUS the zero axis on the x an Y demension plus 1 (for map center)
	total_coordinates := ((req.MaxX*2)*(req.MaxY*2) + (req.MaxX * 2) + (req.MaxY * 2)) + 1
	s.log.Debug("map geometry determined", info.LoggingContext(
		"max_y_positive", max_positive_y,
		"max_y_negative", max_negative_y,
		"max_x_positive", max_positive_x,
		"max_x_negative", max_negative_x,
		"total_coordinates", total_coordinates,
		"game", req.GameUid)...,
	)

	// map[X:Y] = &v1.MapCoordinateDetail{X: X, Y: Y}
	grid := make(map[string]*v1.MapCoordinateDetail)
	for x := max_negative_x; x <= max_positive_x; x++ {
		for y := max_negative_y; y <= max_positive_y; y++ {
			grid[fmt.Sprintf("%d:%d", x, y)] = nil
		}
	}

	if len(grid) != int(total_coordinates) {
		s.log.Error("failed to create map", info.LoggingContext(
			"game", req.GameUid,
			"total_coordinates", total_coordinates,
			"max_positive_y", max_positive_y,
			"max_negative_y", max_negative_y,
			"max_positive_x", max_positive_x,
			"max_negative_x", max_negative_x,
			"grid", len(grid),
			"keys", maps.Keys(grid),
		)...)
		return nil, status.Error(codes.Internal, "failed to create map")
	}

	// randomly determine where the actors will start on the map
	startX, startY := s.randomCoordinate(max_positive_x, max_positive_y)
	s.log.Debug("starting position", info.LoggingContext("x", startX, "y", startY)...)
	if _, ok := grid[fmt.Sprintf("%d:%d", startX, startY)]; !ok {
		s.log.Error("failed to find starting position", info.LoggingContext("x", startX, "y", startY)...)
		return nil, status.Error(codes.Internal, "failed to create map starting position")
	}
	startingCoordinate := fmt.Sprintf("%d:%d", startX, startY)
	startingCoordinateDetail := &v1.MapCoordinateDetail{
		Uid:     common.GenerateUniqueId(),
		GameUid: req.GameUid,
		MapUid:  newMap.Uid,
		Position: &v1.MapPosition{
			X: startX,
			Y: startY,
		},
		Actors:  make([]*v1.Actor, 0),
		Sprites: make([]*v1.Sprite, 0),
	}
	grid[startingCoordinate] = startingCoordinateDetail
	// generate their sprites
	for _, actor := range req.Actors {
		sprite := &v1.Sprite{
			Actor: actor,
		}
		startingCoordinateDetail.Actors = append(grid[fmt.Sprintf("%d:%d", startX, startY)].Actors, actor)
		startingCoordinateDetail.Sprites = append(grid[fmt.Sprintf("%d:%d", startX, startY)].Sprites, sprite)
	}
	grid[startingCoordinate] = startingCoordinateDetail
	// move iteratively through the map in a sweep pattern generating sprites for each coordinate determined by the previous coordiate
	stillWalking := true
	currentX, currentY := max_negative_x, max_negative_y
	directionOfMovement := north
	for stillWalking {
		potentialPos := fmt.Sprintf("%d:%d", currentX, currentY)
		s.log.Debug("walking", info.LoggingContext("x", currentX, "y", currentY, "coordinate", potentialPos, "game", req.GameUid)...)

		// check if we have visited this coordinate
		if _, ok := grid[potentialPos]; !ok {
			s.log.Error("coordinate out of bounds", info.LoggingContext("position", potentialPos, "game", req.GameUid)...)
			return nil, status.Error(codes.Internal, "failed to create map -- generation ran out of bounds")
		}

		coord, err := s.generateCoordinate(ctx, req.GameUid, newMap.Uid, req.Theme, req.SpriteDensity, req.DifficultTerrainChance, currentX, currentY, grid)
		if err != nil {
			s.log.Error("failed to generate coordinate", info.LoggingContext("error", err)...)
			return nil, err
		}

		s.log.Debug("map coordinate generated", info.LoggingContext(
			"x", coord.Position.X,
			"y", coord.Position.Y,
			"actors", len(coord.Actors),
			"sprites", len(coord.Sprites),
			"game", req.GameUid,
			"type", v1.MapCoordinateDetail_CoordinateType_name[int32(coord.Type)],
		)...)

		grid[potentialPos] = coord

		// check if we have visited all coordinates
		remaining := common.Reduce(maps.Keys(grid), func(k string) bool {
			return grid[k] == nil
		})
		if len(remaining) == 0 {
			stillWalking = false
		} else {
			var action string
			// determine the next coordinate
			switch directionOfMovement {
			case north:
				if currentY == max_positive_y {
					directionOfMovement = east
					currentX++
					action = "turning east from top of map"
				} else {
					currentY++
					action = "moving north"
				}
			case south:
				if currentY == max_negative_y {
					directionOfMovement = east
					currentX++
					action = "turning east from bottom of map"
				} else {
					currentY--
					action = "moving south"
				}
			case east:
				if currentY == max_positive_y {
					directionOfMovement = south
					currentY--
					action = "turning south from top map"
				} else if currentY == max_negative_y {
					directionOfMovement = north
					currentY++
					action = "turning north from bottom map"
				} else {
					s.log.Error("invalid direction of movement -- trying to turn outside of maximums", info.LoggingContext("direction", directionOfMovement)...)
					return nil, status.Error(codes.Internal, "failed to create map -- invalid direction of movement during map generation")
				}
			default:
				s.log.Error("invalid direction of movement", info.LoggingContext("direction", directionOfMovement)...)
				return nil, status.Error(codes.Internal, "failed to create map -- invalid direction of movement during map generation")
			}
			s.log.Debug("still walking", info.LoggingContext("remaining", len(remaining), "action", action)...)
		}
	}

	s.log.Info("map generation complete", info.LoggingContext("game", req.GameUid, "duration", time.Since(generationStartTime), "size", len(grid))...)
	// persist the map to the database
	startPersistence := time.Now()

	for _, coord := range grid {
		err = s.mapsDb.CreateCoordinate(ctx, coord)
		if err != nil {
			s.log.Error("failed to persist map coordinate", info.LoggingContext("error", err)...)
			return nil, err
		}
	}

	s.log.Info("map created", info.LoggingContext(
		"game", req.GameUid,
		"map", newMap.Uid,
		"persistence_duration", time.Since(startPersistence),
		"total_duration", time.Since(generationStartTime),
	)...)

	return newMap, nil
}

func (s *defaultMapServer) randomCoordinate(x int64, y int64) (int64, int64) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randX := r.Int63n(2*x+1) - x
	randY := r.Int63n(2*y+1) - y
	return randX, randY
}

func (s *defaultMapServer) generateCoordinate(ctx context.Context, gameUid string, mapUid string, gameTheme v1.GameTheme, terrainDifficultyChance float32, spriteDensity float32, x int64, y int64, grid map[string]*v1.MapCoordinateDetail) (*v1.MapCoordinateDetail, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		return nil, err
	}

	coord, ok := grid[fmt.Sprintf("%d:%d", x, y)]
	if !ok || coord == nil {
		coord = &v1.MapCoordinateDetail{
			Uid:     common.GenerateUniqueId(),
			GameUid: gameUid,
			MapUid:  mapUid,
			Position: &v1.MapPosition{
				X: x,
				Y: y,
			},
			Actors:  make([]*v1.Actor, 0),
			Sprites: make([]*v1.Sprite, 0),
		}
	} else {
		s.log.Info("pre-existing coordinate available on the grid", info.LoggingContext("x", coord.Position.X, "y", coord.Position.Y)...)
	}

	neighbors := common.FindNeighbors(coord.Position, grid)
	s.log.Debug("neighbors found", info.LoggingContext("neighbors", len(neighbors))...)
	coord, err = s.mapGenerator.GenerateCoordinate(ctx, gameTheme, spriteDensity, coord, neighbors)
	if err != nil {
		s.log.Error("failed to generate coordinate", info.LoggingContext("error", err)...)
		return nil, err
	}

	// generate a random value between zero and one
	// hack: this should be more dynamic and algoritmic to determine the difficulty of the terrain based on a coefficient
	// bug: values larger than 1 will always be true right now ALL THE TIME
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	if r.Float32() < terrainDifficultyChance {
		s.log.Debug("difficult terrain selected", info.LoggingContext(
			"x", x,
			"y", y,
		)...)
		coord.DifficultTerrain = true
	} else {
		coord.DifficultTerrain = false
	}

	return coord, nil
}

func (s *defaultMapServer) validateCreateMapRequest(req *v1.CreateMapRequest) error {
	if req.MaxX < 1 || req.MaxY < 1 {
		return status.Error(codes.InvalidArgument, "invalid map dimensions")
	}

	if req.DifficultTerrainChance < common.GetConfiguration().MapGeneration.MinimumTerrainDifficulty {
		return status.Error(codes.InvalidArgument, "difficult terrain chance is below the minimum threshold")
	}
	if req.DifficultTerrainChance > common.GetConfiguration().MapGeneration.MaximumTerrainDifficulty {
		return status.Error(codes.InvalidArgument, "difficult terrain chance is above the maximum threshold")
	}

	if req.SpriteDensity < common.GetConfiguration().MapGeneration.MinimumSpriteDensity {
		return status.Error(codes.InvalidArgument, "sprite density is below the minimum threshold")
	}
	if req.SpriteDensity > common.GetConfiguration().MapGeneration.MaximumSpriteDensity {
		return status.Error(codes.InvalidArgument, "sprite density is above the maximum threshold")
	}

	if len(req.Actors) < 1 {
		return status.Error(codes.InvalidArgument, "There are no players in the game")
	}

	return nil
}

func (s *defaultMapServer) GetMap(ctx context.Context, req *v1.GetMapRequest) (*v1.Map, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		return nil, err
	}

	s.log.Info("getting map", info.LoggingContext("game", req.Uid)...)
	gameMap, err := s.mapsDb.GetMap(ctx, req.Uid)
	if err != nil {
		s.log.Error("failed to get map", info.LoggingContext("error", err)...)
		return nil, err
	}

	return gameMap, nil
}

func (s *defaultMapServer) GetMapDetail(ctx context.Context, req *v1.GetMapRequest) (*v1.MapDetail, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		return nil, err
	}

	s.log.Info("getting map detail", info.LoggingContext("game", req.Uid)...)
	gameMap, err := s.GetMap(ctx, req)
	if err != nil {
		s.log.Error("failed to get map detail", info.LoggingContext("error", err)...)
		return nil, err
	}

	coords, err := s.mapsDb.GetCoordinates(ctx, gameMap.Uid)
	if err != nil {
		s.log.Error("failed to get map coordinates", info.LoggingContext("error", err)...)
		return nil, err
	}

	return &v1.MapDetail{
		Map:         gameMap,
		Coordinates: coords,
	}, nil
}

func (s *defaultMapServer) GetPosition(ctx context.Context, req *v1.Actor) (*v1.MapPosition, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		return nil, err
	}

	s.log.Info("getting position", info.LoggingContext("actor", req.Uid)...)
	return nil, status.Error(codes.Unimplemented, "method GetPosition not implemented")
}

func (s *defaultMapServer) PeekCoordinate(ctx context.Context, req *v1.PeekCoordinateRequest) (*v1.MapCoordinateDetail, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		return nil, err
	}

	s.log.Info("peeking coordinate", info.LoggingContext("x", req.Coordinate.X, "y", req.Coordinate.Y, "game", req.GameUid, "map", req.GameUid)...)
	return nil, status.Error(codes.Unimplemented, "method PeekCoordinate not implemented")
}

func (s *defaultMapServer) PlayerMovement(ctx context.Context, req *v1.PlayerMovementRequest) (*v1.MovementResult, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		return nil, err
	}

	s.log.Info("player movement", info.LoggingContext("actor", req.ActorUid, "x", req.X, "y", req.Y, "game", req.GameUid)...)
	return nil, status.Error(codes.Unimplemented, "method PlayerMovement not implemented")
}
