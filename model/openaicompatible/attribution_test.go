package openaicompatible

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yeomyeonggeori/blueprotocol/model"
)

func headersOfOneChatCompletion(t *testing.T, provider func(endpoint string) *Provider) http.Header {
	t.Helper()
	var received http.Header
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		received = request.Header.Clone()
		writer.Write([]byte(`{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()
	if _, errorValue := provider(server.URL).GenerateChatCompletion(context.Background(), model.ChatCompletionRequest{}); errorValue != nil {
		t.Fatal(errorValue)
	}
	return received
}

func TestRequestsCarryTheAttributionTheCallerSet(t *testing.T) {
	received := headersOfOneChatCompletion(t, func(endpoint string) *Provider {
		return NewProvider(endpoint, "", "example/model").WithAttribution(Attribution{ReferrerURL: "https://example.com/app", Title: "example app"})
	})
	if received.Get("HTTP-Referer") != "https://example.com/app" || received.Get("X-Title") != "example app" {
		t.Fatalf("attribution headers were %v", received)
	}
}

func TestRequestsCarryNoAttributionUntilTheCallerSetsOne(t *testing.T) {
	received := headersOfOneChatCompletion(t, func(endpoint string) *Provider {
		return NewProvider(endpoint, "", "example/model")
	})
	if received.Get("HTTP-Referer") != "" || received.Get("X-Title") != "" {
		t.Fatalf("a provider that was never told who the app is named one anyway: %v", received)
	}
}

func TestEmbeddingRequestsCarryTheAttributionTheCallerSet(t *testing.T) {
	var received http.Header
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		received = request.Header.Clone()
		writer.Write([]byte(`{"data":[{"embedding":[0.5]}]}`))
	}))
	defer server.Close()
	provider := NewEmbeddingProvider(server.URL, "", "example/embedding").WithAttribution(Attribution{Title: "example app"})
	if _, errorValue := provider.GenerateEmbedding(context.Background(), "text"); errorValue != nil {
		t.Fatal(errorValue)
	}
	if received.Get("X-Title") != "example app" || received.Get("HTTP-Referer") != "" {
		t.Fatalf("attribution headers were %v", received)
	}
}
