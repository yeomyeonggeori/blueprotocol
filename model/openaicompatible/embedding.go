package openaicompatible

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/yeomyeonggeori/blueprotocol/model/embeddingprompt"
)

type EmbeddingProvider struct {
	endpointURL string
	apiKey      string
	modelName   string
	httpClient  *http.Client
	attribution Attribution
}

func NewEmbeddingProvider(endpointURL string, apiKey string, modelName string) *EmbeddingProvider {
	return &EmbeddingProvider{
		endpointURL: strings.TrimSuffix(strings.TrimSpace(endpointURL), "/"),
		apiKey:      strings.TrimSpace(apiKey),
		modelName:   strings.TrimSpace(modelName),
		httpClient:  http.DefaultClient,
	}
}

func (provider *EmbeddingProvider) WithAttribution(attribution Attribution) *EmbeddingProvider {
	provider.attribution = attribution
	return provider
}

func (provider *EmbeddingProvider) UseHTTPClient(httpClient *http.Client) {
	provider.httpClient = httpClient
}

func (provider *EmbeddingProvider) ModelName() string {
	return provider.modelName
}

type embeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

func (provider *EmbeddingProvider) GenerateEmbedding(ctx context.Context, input string) ([]float32, error) {
	return provider.embed(ctx, input)
}

func (provider *EmbeddingProvider) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return provider.embed(ctx, embeddingprompt.Apply(provider.modelName, embeddingprompt.InputTypeQuery, text))
}

func (provider *EmbeddingProvider) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, 0, len(texts))
	for _, text := range texts {
		embedding, errorValue := provider.embed(ctx, embeddingprompt.Apply(provider.modelName, embeddingprompt.InputTypeDocument, text))
		if errorValue != nil {
			return nil, errorValue
		}
		embeddings = append(embeddings, embedding)
	}
	return embeddings, nil
}

func (provider *EmbeddingProvider) embed(ctx context.Context, input string) ([]float32, error) {
	body, errorValue := json.Marshal(embeddingRequest{Model: provider.modelName, Input: input})
	if errorValue != nil {
		return nil, errorValue
	}
	httpRequest, errorValue := http.NewRequestWithContext(ctx, http.MethodPost, provider.endpointURL+"/embeddings", bytes.NewReader(body))
	if errorValue != nil {
		return nil, errorValue
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	provider.attribution.apply(httpRequest)
	if provider.apiKey != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+provider.apiKey)
	}

	httpResponse, errorValue := provider.httpClient.Do(httpRequest)
	if errorValue != nil {
		return nil, errorValue
	}
	defer httpResponse.Body.Close()
	if httpResponse.StatusCode != http.StatusOK {
		return nil, errors.New("embedding endpoint returned " + httpResponse.Status)
	}

	var decoded embeddingResponse
	if errorValue := json.NewDecoder(httpResponse.Body).Decode(&decoded); errorValue != nil {
		return nil, errorValue
	}
	if len(decoded.Data) == 0 || len(decoded.Data[0].Embedding) == 0 {
		return nil, errors.New("embedding endpoint returned no vector")
	}
	return float32Embedding(decoded.Data[0].Embedding), nil
}

func float32Embedding(values []float64) []float32 {
	embedding := make([]float32, 0, len(values))
	for _, value := range values {
		embedding = append(embedding, float32(value))
	}
	return embedding
}
