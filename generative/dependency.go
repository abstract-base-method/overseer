package generative

import (
	"context"
	"fmt"
	"net/url"
	"overseer/common"
	"overseer/generative/ollama"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var initializeOllama sync.Once

func NewMapGenerationService() (MapGenerationService, error) {
	switch common.GetConfiguration().GenerativeFeaturesProvider {
	case common.OllamaProvider:
		base, err := url.Parse(common.GetConfiguration().Ollama.BaseUrl)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid ollama base url: %s", err))
		}
		client := ollama.NewOllamaClient(base)
		initializeOllama.Do(func() {
			common.GetLogger("generative.map").Info("Initializing Ollama client")
			err = client.InitializeOllama(context.Background())
		})
		tmpl, err := NewTemplatingService()
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to initialize templating service: %s", err))
		}
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to initialize ollama client: %s", err))
		}
		return NewOllamaMapGenerationService(tmpl, client)
	default:
		return nil, status.Error(codes.InvalidArgument, "invalid generative feature provider")
	}
}
