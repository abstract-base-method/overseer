package generative

import (
	"bytes"
	"context"
	"math/rand"
	v1 "overseer/build/go"
	"overseer/common"
	"overseer/generative/ollama"
	"reflect"
	"strings"
	"time"
)

// chooseTheme is a method that selects a theme for a coordinate based on its neighbors
// neighbors is a slice of previously generated Map Coordinate Details
// it is vital to only provide neighbors that are directly adjacent to the coordinate and previously generated since they are the "seed" for the theme selection
// the method returns a MapCoordinateDetail_CoordinateType and an error if one occurs
func (s *ollamaMapGeneration) chooseTheme(neighbors []*v1.MapCoordinateDetail) (v1.MapCoordinateDetail_CoordinateType, error) {
	var theme v1.MapCoordinateDetail_CoordinateType
	if len(neighbors) == 0 {
		theme = s.randomTheme()
		s.log.Debug("No neighbors found, selecting random theme", "theme", theme)
		return theme, nil
	}

	// sometimes we just choose a random theme because life is strange and sometimes a single forest is surrounded by deserts
	if rand.Float64() < common.GetConfiguration().MapGeneration.ChanceOfRandomTheme {
		theme = s.randomTheme()
		s.log.Debug("Random theme selected", "theme", theme)
		return theme, nil
	}

	biomes := make([]*v1.MapCoordinateDetail_CoordinateType, len(neighbors))
	for idx, neighbor := range neighbors {
		biomes[idx] = &neighbor.Type
	}

	// now choose a random value from biomes
	theme = *biomes[rand.Intn(len(biomes))]

	return theme, nil
}

func (s *ollamaMapGeneration) randomTheme() v1.MapCoordinateDetail_CoordinateType {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Use reflection to get the enum type
	enumType := reflect.TypeOf(v1.MapCoordinateDetail_CoordinateType(0))
	enumCount := enumType.NumMethod() // Get the number of enum values

	// Select a random enum value
	randomIndex := 0
	// hack: I was tired when I wrote this but you're probably awake because of it -- sorry
	for randomIndex == 0 {
		randomIndex = r.Intn(enumCount)
	}
	randomValue := reflect.New(enumType).Elem()
	randomValue.SetInt(int64(randomIndex))

	return v1.MapCoordinateDetail_CoordinateType(randomValue.Int())
}

func (s *ollamaMapGeneration) generateSprites(ctx context.Context, gameTheme v1.GameTheme, spriteDensity float32, coordinate *v1.MapCoordinateDetail, neighbors []*v1.MapCoordinateDetail) ([]*v1.Sprite, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		return nil, err
	}

	s.log.Debug("Generating sprites", info.LoggingContext(
		"x", coordinate.Position.X,
		"y", coordinate.Position.Y,
		"neighbors", len(neighbors),
		"sprite_density", spriteDensity,
	)...)
	startTime := time.Now()
	sprites := []*v1.Sprite{}

	if len(coordinate.Actors) != 0 {
		s.log.Debug("actors detected handling their sprites", info.LoggingContext(
			"x", coordinate.Position.X,
			"y", coordinate.Position.Y,
		)...)
		// todo: this is where we need to look up the DnD beyond account for the associated User and generate their sprite based on that content
		for _, actor := range coordinate.Actors {
			sprites = append(sprites, &v1.Sprite{
				Uid:             common.GenerateUniqueId(),
				Actor:           actor,
				Characteristics: make([]*v1.Characteristic, 0),
				IsObstacle:      true,
				IsMoveable:      true,
			})
		}
	}

	numberOfSprites := common.RandomizedProgressiveValue(
		common.GetConfiguration().MapGeneration.MinimumSpriteDensity,
		spriteDensity,
		common.GetConfiguration().MapGeneration.MaximumSpriteDensity,
		common.GetConfiguration().MapGeneration.MaximumSpritesPerCoordinate,
	)
	s.log.Debug("generating sprites", info.LoggingContext(
		"number_of_sprites", numberOfSprites,
		"x", coordinate.Position.X,
		"y", coordinate.Position.Y,
	)...)
	for i := 0; i < numberOfSprites; i++ {
		sprite, err := s.generateSprite(ctx, gameTheme, coordinate, sprites, neighbors)
		if err != nil {
			s.log.Error("failed to generate sprite", info.LoggingContext(
				"error", err,
				"x", coordinate.Position.X,
				"y", coordinate.Position.Y,
			)...)
			return nil, err
		}
		sprites = append(sprites, sprite)
	}

	s.log.Debug("sprite generation completed", info.LoggingContext(
		"duration", time.Since(startTime),
		"sprites", len(sprites),
		"x", coordinate.Position.X,
		"y", coordinate.Position.Y,
	)...)
	return sprites, nil
}

type ollamaSpriteInternalLoreGenerationTemplate struct {
	Theme            v1.GameTheme
	Name             string
	Characteristics  []*v1.Characteristic
	LocationTheme    v1.MapCoordinateDetail_CoordinateType
	DifficultTerrain string
}

type ollamaSpriteExternalLoreGenerationTemplate struct {
	Theme            v1.GameTheme
	Name             string
	Characteristics  []*v1.Characteristic
	LocationTheme    v1.MapCoordinateDetail_CoordinateType
	DifficultTerrain string
	InternalLore     string
}

type ollamaMapLoreGenerationTemplate struct {
	Theme         v1.GameTheme
	LocationTheme v1.MapCoordinateDetail_CoordinateType
	SpriteLores   []string
	NeighborLores []ollamaMapGenerationNeighborLoreTemplate
}

type ollamaMapGenerationNeighborLoreTemplate struct {
	Theme       v1.MapCoordinateDetail_CoordinateType
	Direction   string
	Lore        string
	SpriteLores []string
}

func (s *ollamaMapGeneration) generateSprite(ctx context.Context, gameTheme v1.GameTheme, coordinate *v1.MapCoordinateDetail, localSprites []*v1.Sprite, neighbors []*v1.MapCoordinateDetail) (*v1.Sprite, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		return nil, err
	}

	s.log.Debug("Generating sprite", info.LoggingContext(
		"x", coordinate.Position.X,
		"y", coordinate.Position.Y,
		"neighbors", len(neighbors),
		"prexisting_sprites", len(localSprites),
	)...)

	enumType := reflect.TypeOf(v1.Characteristic_Type(0))
	enumCount := enumType.NumMethod() // Get the number of enum values

	characteristics := make([]*v1.Characteristic, 0)
	for i := 1; i <= enumCount; i++ {
		randomCharacteristicValue := rand.Float32() * 100.0
		characteristics = append(characteristics, &v1.Characteristic{
			Type:  v1.Characteristic_Type(i),
			Value: randomCharacteristicValue,
		})
	}

	sprite := &v1.Sprite{
		Uid:             common.GenerateUniqueId(),
		Actor:           nil,
		Characteristics: characteristics,
		IsObstacle:      true,
		IsMoveable:      true,
	}

	var difficultTerrain string
	if coordinate.DifficultTerrain {
		difficultTerrain = "Difficult"
	} else {
		difficultTerrain = "Normal"
	}

	tmplVals := &ollamaSpriteInternalLoreGenerationTemplate{
		Theme:            gameTheme,
		Name:             "Sprite",
		Characteristics:  characteristics,
		LocationTheme:    coordinate.Type,
		DifficultTerrain: difficultTerrain,
	}

	var buf bytes.Buffer
	err = s.templating.InternalLoreTemplate().Execute(&buf, tmplVals)
	if err != nil {
		s.log.Error("failed to execute sprite template", info.LoggingContext("error", err)...)
		return nil, err
	}

	results, err := s.client.Generate(ctx, ollama.GenerateRequest{
		Model:  common.GetConfiguration().Ollama.Model.String(),
		Prompt: buf.String(),
		Stream: false,
	})
	if err != nil {
		s.log.Error("failed to generate sprite", info.LoggingContext("error", err)...)
		return nil, err
	}

	var internalLore strings.Builder
	for result := range results {
		s.log.Debug("received result from ollama", info.LoggingContext("result", result.Response)...)
		internalLore.WriteString(result.Response)
	}

	sprite.LoreInternal = internalLore.String()

	tmplValsExternal := &ollamaSpriteExternalLoreGenerationTemplate{
		Theme:            gameTheme,
		Name:             "Sprite",
		Characteristics:  characteristics,
		LocationTheme:    coordinate.Type,
		DifficultTerrain: difficultTerrain,
		InternalLore:     sprite.LoreInternal,
	}

	var bufExternal bytes.Buffer
	err = s.templating.PublicLoreTemplate().Execute(&bufExternal, tmplValsExternal)
	if err != nil {
		s.log.Error("failed to execute sprite template", info.LoggingContext("error", err)...)
		return nil, err
	}

	resultsExternal, err := s.client.Generate(ctx, ollama.GenerateRequest{
		Model:  common.GetConfiguration().Ollama.Model.String(),
		Prompt: bufExternal.String(),
		Stream: false,
	})
	if err != nil {
		s.log.Error("failed to generate sprite", info.LoggingContext("error", err)...)
		return nil, err
	}

	var externalLore strings.Builder
	for result := range resultsExternal {
		s.log.Debug("received result from ollama", info.LoggingContext("result", result.Response)...)
		externalLore.WriteString(result.Response)
	}

	sprite.LorePublic = externalLore.String()

	return sprite, nil
}

func (s *ollamaMapGeneration) generateLore(ctx context.Context, gameTheme v1.GameTheme, coordinate *v1.MapCoordinateDetail, neighbors []*v1.MapCoordinateDetail) (string, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		return "", err
	}

	s.log.Debug("Generating lore", info.LoggingContext(
		"x", coordinate.Position.X,
		"y", coordinate.Position.Y,
		"neighbors", len(neighbors),
	)...)
	startTime := time.Now()

	spriteLores := make([]string, 0)
	for _, sprite := range coordinate.Sprites {
		spriteLores = append(spriteLores, sprite.LoreInternal)
	}

	neighborLores := make([]ollamaMapGenerationNeighborLoreTemplate, 0)
	for _, neighbor := range neighbors {
		neighborSpriteLores := make([]string, 0)
		for _, sprite := range neighbor.Sprites {
			neighborSpriteLores = append(neighborSpriteLores, sprite.LoreInternal)
		}
		direction := common.GetDirection(coordinate.Position, neighbor.Position)
		neighborLores = append(neighborLores, ollamaMapGenerationNeighborLoreTemplate{
			Theme:       neighbor.Type,
			Direction:   direction,
			Lore:        neighbor.Lore,
			SpriteLores: neighborSpriteLores,
		})
	}

	templateVals := &ollamaMapLoreGenerationTemplate{
		Theme:         gameTheme,
		LocationTheme: coordinate.Type,
		SpriteLores:   spriteLores,
		NeighborLores: neighborLores,
	}

	var buf bytes.Buffer
	err = s.templating.CoordinateLoreTemplate().Execute(&buf, templateVals)
	if err != nil {
		s.log.Error("failed to execute lore template", info.LoggingContext("error", err)...)
		return "", err
	}

	results, err := s.client.Generate(ctx, ollama.GenerateRequest{
		Model:  common.GetConfiguration().Ollama.Model.String(),
		Prompt: buf.String(),
		Stream: false,
	})
	if err != nil {
		s.log.Error("failed to generate lore", info.LoggingContext("error", err)...)
		return "", err
	}

	var lore strings.Builder
	for result := range results {
		s.log.Debug("received result from ollama", info.LoggingContext("result", result.Response)...)
		lore.WriteString(result.Response)
	}

	s.log.Debug("lore generation completed", info.LoggingContext(
		"duration", time.Since(startTime),
		"x", coordinate.Position.X,
		"y", coordinate.Position.Y,
	)...)
	return lore.String(), nil
}
