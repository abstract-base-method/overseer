package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"overseer/common"
	"time"

	charm "github.com/charmbracelet/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Client interface {
	InitializeOllama(ctx context.Context) error
	PullModel(ctx context.Context, req PullModelRequest) (chan PullModelResponse, error)
	ListModels(ctx context.Context) (*ListModelsResponse, error)
	GetModel(ctx context.Context, req GetModelRequest) (*ModelInformation, error)
	Generate(ctx context.Context, req GenerateRequest) (chan GenerateResponse, error)
	Converse(ctx context.Context, req ConverseRequest) (chan ConverseResponse, error)
}

func NewOllamaClient(baseUrl *url.URL) Client {
	return &defaultClient{
		baseUrl: baseUrl,
		client:  &http.Client{},
		log:     common.GetLogger("generative.ollama"),
	}
}

type defaultClient struct {
	baseUrl *url.URL
	client  *http.Client
	log     *charm.Logger
}

func (c *defaultClient) InitializeOllama(ctx context.Context) error {
	installed, err := c.ListModels(ctx)
	if err != nil {
		return err
	}

	previouslyInstalled := false
	for _, model := range installed.Models {
		c.log.Info("model previously installed", "model", model.Name)
		if model.Name == common.GetConfiguration().Ollama.Model.String() {
			previouslyInstalled = true
			break
		}
	}

	if !previouslyInstalled {
		c.log.Info("installing model", "model", common.GetConfiguration().Ollama.Model)
		startTime := time.Now()
		results, err := c.PullModel(ctx, PullModelRequest{
			Name:     common.GetConfiguration().Ollama.Model.String(),
			Stream:   true,
			Insecure: common.GetConfiguration().Ollama.Insecure,
		})
		if err != nil {
			return err
		}
		for msg := range results {
			c.log.Info("model installation status",
				"status", msg.Status,
				"completed", msg.Completed,
				"total", msg.Total,
				"digest", msg.Digest,
			)
		}
		c.log.Info("model installed", "model", common.GetConfiguration().Ollama.Model, "duration", time.Since(startTime))
	}

	c.log.Info("model previously installed", "model", common.GetConfiguration().Ollama.Model)
	return nil
}

func (c *defaultClient) PullModel(ctx context.Context, req PullModelRequest) (chan PullModelResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		c.log.Error("failed to marshal request", "error", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to marshal request: %v", err))
	}

	request, error := http.NewRequestWithContext(ctx, http.MethodPost, c.baseUrl.String()+"/api/pull", bytes.NewReader(payload))
	if error != nil {
		c.log.Error("failed to create request", "error", error)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create request: %v", error))
	}

	startTime := time.Now()
	results := make(chan PullModelResponse, common.GetConfiguration().ChannelBuffer)
	go c.handleListModelsStream(ctx, request, results, startTime)

	return results, nil
}

func (c *defaultClient) handleListModelsStream(ctx context.Context, req *http.Request, results chan PullModelResponse, startTime time.Time) {
	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Error("failed to execute request", "error", err)
		close(results)
		return
	}

	if resp.StatusCode != http.StatusOK {
		c.log.Error("failed to pull model", "status", resp.Status)
		close(results)
		return
	}

	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)

	for {
		select {
		case <-ctx.Done():
			c.log.Debug("pull model stream cancelled", "duration", time.Since(startTime))
			close(results)
			return
		default:
			if !scanner.Scan() {
				close(results)
				c.log.Debug("pull model stream closed", "duration", time.Since(startTime))
				return
			}
			var msg PullModelResponse
			if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
				c.log.Error("failed to unmarshal response", "error", err)
				continue
			}
			c.log.Debug("pull model response", "status", msg.Status, "completed", msg.Completed, "total", msg.Total, "digest", msg.Digest)
			results <- msg
		}
	}
}

func (c *defaultClient) ListModels(ctx context.Context) (*ListModelsResponse, error) {
	startTime := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseUrl.String()+"/api/tags", nil)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create request: %v", err))
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to execute request: %v", err))
	}
	defer resp.Body.Close()
	c.log.Debug("list models request", "duration", time.Since(startTime), "status", resp.Status)

	if resp.StatusCode != http.StatusOK {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list models: %v", resp.Status))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.Error("failed to read response body", "error", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to read response body: %v", err))
	}

	var models ListModelsResponse
	if err := json.Unmarshal(body, &models); err != nil {
		c.log.Error("failed to unmarshal response body", "error", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to unmarshal response body: %v", err))
	}

	return &models, nil
}

func (c *defaultClient) GetModel(ctx context.Context, req GetModelRequest) (*ModelInformation, error) {
	c.log.Debug("get model request", "model", req.Name)
	payload, err := json.Marshal(req)
	if err != nil {
		c.log.Error("failed to marshal request", "error", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to marshal request: %v", err))
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseUrl.String()+"/api/show/", bytes.NewReader(payload))
	if err != nil {
		c.log.Error("failed to create request", "error", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create request: %v", err))
	}

	startTime := time.Now()
	resp, err := c.client.Do(request)
	if err != nil {
		c.log.Error("failed to execute request", "error", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to execute request: %v", err))
	}

	if resp.StatusCode != http.StatusOK {
		c.log.Error("failed to get model", "status", resp.Status)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get model: %v", resp.Status))
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.Error("failed to read response body", "error", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to read response body: %v", err))
	}

	var model ModelInformation
	if err := json.Unmarshal(body, &model); err != nil {
		c.log.Error("failed to unmarshal response body", "error", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to unmarshal response body: %v", err))
	}

	c.log.Debug("get model response", "model", model.Name, "duration", time.Since(startTime))
	return &model, nil
}

func (c *defaultClient) Generate(ctx context.Context, req GenerateRequest) (chan GenerateResponse, error) {
	c.log.Debug("generate request", "model", req.Model)
	payload, err := json.Marshal(req)
	if err != nil {
		c.log.Error("failed to marshal request", "error", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to marshal request: %v", err))
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseUrl.String()+"/api/generate", bytes.NewReader(payload))
	if err != nil {
		c.log.Error("failed to create request", "error", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create request: %v", err))
	}

	startTime := time.Now()
	results := make(chan GenerateResponse, common.GetConfiguration().ChannelBuffer)
	go c.handleGenerateStream(ctx, request, results, startTime)

	return results, nil
}

func (c *defaultClient) handleGenerateStream(ctx context.Context, req *http.Request, results chan GenerateResponse, startTime time.Time) {
	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Error("failed to execute request", "error", err)
		close(results)
		return
	}

	if resp.StatusCode != http.StatusOK {
		c.log.Error("failed to generate", "status", resp.Status)
		close(results)
		return
	}

	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)

	for {
		select {
		case <-ctx.Done():
			c.log.Debug("generate stream cancelled", "duration", time.Since(startTime))
			close(results)
			return
		default:
			if !scanner.Scan() {
				close(results)
				c.log.Debug("generate stream closed", "duration", time.Since(startTime))
				return
			}
			var msg GenerateResponse
			if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
				c.log.Error("failed to unmarshal response", "error", err)
				continue
			}
			c.log.Debug("generate response", "model", msg.Model, "response", msg.Response, "done", msg.Done)
			results <- msg
		}
	}
}

func (c *defaultClient) Converse(ctx context.Context, req ConverseRequest) (chan ConverseResponse, error) {
	c.log.Debug("converse request", "model", req.Model)
	payload, err := json.Marshal(req)
	if err != nil {
		c.log.Error("failed to marshal request", "error", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to marshal request: %v", err))
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseUrl.String()+"/api/chat", bytes.NewReader(payload))
	if err != nil {
		c.log.Error("failed to create request", "error", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create request: %v", err))
	}

	startTime := time.Now()
	results := make(chan ConverseResponse, common.GetConfiguration().ChannelBuffer)
	go c.handleConverseStream(ctx, request, results, startTime)

	return results, nil
}

func (c *defaultClient) handleConverseStream(ctx context.Context, req *http.Request, results chan ConverseResponse, startTime time.Time) {
	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Error("failed to execute request", "error", err)
		close(results)
		return
	}

	if resp.StatusCode != http.StatusOK {
		c.log.Error("failed to converse", "status", resp.Status)
		close(results)
		return
	}

	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)

	for {
		select {
		case <-ctx.Done():
			c.log.Debug("converse stream cancelled", "duration", time.Since(startTime))
			close(results)
			return
		default:
			if !scanner.Scan() {
				close(results)
				c.log.Debug("converse stream closed", "duration", time.Since(startTime))
				return
			}
			var msg ConverseResponse
			if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
				c.log.Error("failed to unmarshal response", "error", err)
				continue
			}
			c.log.Debug("converse response", "model", msg.Model, "done", msg.Done)
			results <- msg
		}
	}
}
