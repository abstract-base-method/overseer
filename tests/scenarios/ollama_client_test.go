package scenarios

import (
	"context"
	"overseer/generative/ollama"

	"github.com/stretchr/testify/mock"
)

type MockOllamaClient struct {
	mock.Mock
}

func (m *MockOllamaClient) InitializeOllama(ctx context.Context) error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockOllamaClient) PullModel(ctx context.Context, req ollama.PullModelRequest) (chan ollama.PullModelResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(chan ollama.PullModelResponse), args.Error(1)
}

func (m *MockOllamaClient) ListModels(ctx context.Context) (*ollama.ListModelsResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*ollama.ListModelsResponse), args.Error(1)
}

func (m *MockOllamaClient) GetModel(ctx context.Context, req ollama.GetModelRequest) (*ollama.ModelInformation, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*ollama.ModelInformation), args.Error(1)
}

func (m *MockOllamaClient) Generate(ctx context.Context, req ollama.GenerateRequest) (chan ollama.GenerateResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(chan ollama.GenerateResponse), args.Error(1)
}

func (m *MockOllamaClient) Converse(ctx context.Context, req ollama.ConverseRequest) (chan ollama.ConverseResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(chan ollama.ConverseResponse), args.Error(1)
}
