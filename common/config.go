package common

import (
	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var configuration Configuration

func GetConfiguration() Configuration {
	return configuration
}

type Configuration struct {
	ChannelBuffer              int64                      `yaml:"channelBuffer" mapstructure:"channelBuffer" json:"channelBuffer"`
	Server                     ServerConfiguration        `yaml:"server" mapstructure:"server" json:"server"`
	Templating                 TemplatingConfiguration    `yaml:"templating" mapstructure:"templating" json:"templating"`
	MapGeneration              MapGenerationConfiguration `yaml:"mapGeneration" mapstructure:"mapGeneration" json:"mapGeneration"`
	GenerativeFeaturesProvider GenerativeFeatureProvider  `yaml:"generativeFeaturesProvider" mapstructure:"generativeFeaturesProvider" json:"generativeFeaturesProvider"`
	Ollama                     OllamaConfiguration        `yaml:"ollama" mapstructure:"ollama" json:"ollama"`
}

type ServerConfiguration struct {
	EnableSystemToken bool   `yaml:"enableSystemToken" mapstructure:"enableSystemToken" json:"enableSystemToken"`
	SystemToken       string `yaml:"systemToken" mapstructure:"systemToken" json:"systemToken"`
}

type TemplatingConfiguration struct {
	TemplateBasePath string `yaml:"templateBasePath" mapstructure:"templateBasePath" json:"templateBasePath"`
}

type MapGenerationConfiguration struct {
	ChanceOfRandomTheme         float64 `yaml:"chanceOfRandomTheme" mapstructure:"chanceOfRandomTheme" json:"chanceOfRandomTheme"`
	MaximumTerrainDifficulty    float32 `yaml:"maximumTerrainDifficulty" mapstructure:"maximumTerrainDifficulty" json:"maximumTerrainDifficulty"`
	MinimumTerrainDifficulty    float32 `yaml:"minimumTerrainDifficulty" mapstructure:"minimumTerrainDifficulty" json:"minimumTerrainDifficulty"`
	MaximumSpriteDensity        float32 `yaml:"maximumSpriteDensity" mapstructure:"maximumSpriteDensity" json:"maximumSpriteDensity"`
	MinimumSpriteDensity        float32 `yaml:"minimumSpriteDensity" mapstructure:"minimumSpriteDensity" json:"minimumSpriteDensity"`
	MaximumSpritesPerCoordinate int     `yaml:"maximumSpritesPerCoordinate" mapstructure:"maximumSpritesPerCoordinate" json:"maximumSpritesPerCoordinate"`
}

type GenerativeFeatureProvider string

// todo: should we offer OpenAI?
const (
	OpenAIProvider GenerativeFeatureProvider = "openai"
	OllamaProvider GenerativeFeatureProvider = "ollama"
)

func (g GenerativeFeatureProvider) String() string {
	return string(g)
}

func ParseGenerativeFeatureProvider(s string) (GenerativeFeatureProvider, error) {
	switch s {
	case "openai":
		return OpenAIProvider, nil
	case "ollama":
		return OllamaProvider, nil
	default:
		return OllamaProvider, status.Error(codes.InvalidArgument, "invalid generative feature provider")
	}
}

type OllamaModel string

const (
	Llama3    OllamaModel = "llama3"
	Llama370B OllamaModel = "llama3:70b"
	Mistral   OllamaModel = "mistral"
)

func (o OllamaModel) String() string {
	return string(o)
}

func ParseOllamaModel(s string) (OllamaModel, error) {
	switch s {
	case "llama3":
		return Llama3, nil
	case "llama3:70b":
		return Llama370B, nil
	case "mistral":
		return Mistral, nil
	default:
		return Llama3, status.Error(codes.InvalidArgument, "invalid ollama model")
	}
}

type OllamaConfiguration struct {
	BaseUrl  string      `yaml:"baseUrl" mapstructure:"baseUrl" json:"baseUrl"`
	Model    OllamaModel `yaml:"model" mapstructure:"model" json:"model"`
	Insecure bool        `yaml:"insecure" mapstructure:"insecure" json:"insecure"`
}

func init() {
	viper.SetConfigName("overseer")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/overseer/")
	viper.AddConfigPath("$HOME/.overseer")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetDefault("channelBuffer", 10000)
	viper.SetDefault("server.enableSystemToken", false)
	viper.SetDefault("server.systemToken", "")
	viper.SetDefault("templating.templateBasePath", "./templates")
	viper.SetDefault("mapGeneration.chanceOfRandomTheme", 0.1)
	viper.SetDefault("mapGeneration.maximumTerrainDifficulty", 2.0)
	viper.SetDefault("mapGeneration.minimumTerrainDifficulty", 0.01)
	viper.SetDefault("mapGeneration.maximumSpriteDensity", 2.0)
	viper.SetDefault("mapGeneration.minimumSpriteDensity", 0.01)
	viper.SetDefault("mapGeneration.maximumSpritesPerCoordinate", 12)
	viper.SetDefault("generativeFeaturesProvider", OllamaProvider.String())
	viper.SetDefault("ollama.baseUrl", "http://localhost:11434")
	viper.SetDefault("ollama.model", Llama3.String())
	viper.SetDefault("ollama.insecure", false)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			GetLogger("config").Warn("config file not found - using defaults")
		} else {
			GetLogger("config").Fatal("failed to read config file", "error", err)
		}
	}

	if err := viper.Unmarshal(&configuration); err != nil {
		GetLogger("config").Fatal("failed to unmarshal configuration", "error", err)
	}

	str := viper.GetString("generativeFeaturesProvider")
	if provider, err := ParseGenerativeFeatureProvider(str); err != nil {
		GetLogger("config").Fatal("failed to parse generative feature provider", "error", err)
	} else {
		configuration.GenerativeFeaturesProvider = provider
	}

	str = viper.GetString("ollama.model")
	if model, err := ParseOllamaModel(str); err != nil {
		GetLogger("config").Fatal("failed to parse ollama model", "error", err)
	} else {
		configuration.Ollama.Model = model
	}
}
