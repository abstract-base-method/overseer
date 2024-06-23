package generative

import (
	"context"
	v1 "overseer/build/go"
	"text/template"
)

type MapGenerationService interface {
	GenerateCoordinate(ctx context.Context, gameTheme v1.GameTheme, spriteDensity float32, coordinate *v1.MapCoordinateDetail, neighbors []*v1.MapCoordinateDetail) (*v1.MapCoordinateDetail, error)
}

type TemplatingService interface {
	InternalLoreTemplate() *template.Template
	PublicLoreTemplate() *template.Template
	CoordinateLoreTemplate() *template.Template
}
