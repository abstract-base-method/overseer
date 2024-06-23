package generative

import (
	"context"
	"os"
	v1 "overseer/build/go"
	"overseer/common"
	"overseer/generative/ollama"
	"time"

	charm "github.com/charmbracelet/log"
)

type ollamaMapGeneration struct {
	client     ollama.Client
	templating TemplatingService
	log        *charm.Logger
}

func NewOllamaMapGenerationService(templating TemplatingService, client ollama.Client) (MapGenerationService, error) {

	server := &ollamaMapGeneration{
		client:     client,
		templating: templating,
		log:        common.GetLogger("service.generative.map.ollama"),
	}

	return server, nil
}

func readTemplateFile(path string) (string, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(contents), nil
}

func (s *ollamaMapGeneration) GenerateCoordinate(ctx context.Context, gameTheme v1.GameTheme, spriteDensity float32, coordinate *v1.MapCoordinateDetail, neighbors []*v1.MapCoordinateDetail) (*v1.MapCoordinateDetail, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("failed to get context information", "error", err)
		return nil, err
	}

	s.log.Debug("generating new map coordinate",
		info.LoggingContext(
			"x", coordinate.Position.X,
			"y", coordinate.Position.Y,
		)...)
	generationStartTime := time.Now()

	if coordinate.Actors == nil {
		coordinate.Actors = make([]*v1.Actor, 0)
	}
	if coordinate.Sprites == nil {
		coordinate.Sprites = make([]*v1.Sprite, 0)
	}

	themeGenerationStarted := time.Now()
	coordType, err := s.chooseTheme(neighbors)
	if err != nil {
		s.log.Error("failed to choose theme", info.LoggingContext("error", err)...)
		return nil, err
	}
	s.log.Debug("theme chosen for coordinate", info.LoggingContext(
		"theme", coordType.String(),
		"coordinate_x", coordinate.Position.X,
		"coordinate_y", coordinate.Position.Y,
		"duration", time.Since(themeGenerationStarted),
	)...)
	coordinate.Type = coordType

	spriteGenerationStarted := time.Now()
	sprites, err := s.generateSprites(ctx, gameTheme, spriteDensity, coordinate, neighbors)
	if err != nil {
		s.log.Error("failed to generate sprites", info.LoggingContext(
			"error", err,
			"x", coordinate.Position.X,
			"y", coordinate.Position.Y,
		)...)
		return nil, err
	}
	s.log.Debug("generated sprites", info.LoggingContext(
		"x", coordinate.Position.X,
		"y", coordinate.Position.Y,
		"duration", time.Since(spriteGenerationStarted),
		"sprites", len(sprites),
	)...)

	for _, sprite := range sprites {
		s.log.Debug("generated sprite", info.LoggingContext(
			"sprite", sprite.Uid,
			"x", coordinate.Position.X,
			"y", coordinate.Position.Y,
		)...)
	}

	coordinate.Sprites = append(coordinate.Sprites, sprites...)

	loreGenerationStarted := time.Now()
	lore, err := s.generateLore(ctx, gameTheme, coordinate, neighbors)
	if err != nil {
		s.log.Error("failed to generate lore", info.LoggingContext(
			"error", err,
			"x", coordinate.Position.X,
			"y", coordinate.Position.Y,
		)...)
		return nil, err
	}
	s.log.Debug("generated lore", info.LoggingContext(
		"x", coordinate.Position.X,
		"y", coordinate.Position.Y,
		"duration", time.Since(loreGenerationStarted),
	)...)

	coordinate.Lore = lore

	s.log.Debug("map coordinate generated", info.LoggingContext(
		"x", coordinate.Position.X,
		"y", coordinate.Position.Y,
		"actors", len(coordinate.Actors),
		"sprites", len(coordinate.Sprites),
		"duration", time.Since(generationStartTime),
	)...)
	return coordinate, nil
}
