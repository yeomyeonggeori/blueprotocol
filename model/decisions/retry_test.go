package decisions

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yeomyeonggeori/blueprotocol/model"
	"github.com/yeomyeonggeori/blueprotocol/model/transientretry"
)

type failingTransport struct {
	failures     int
	attemptCount atomic.Int32
	next         http.RoundTripper
}

func (transport *failingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if int(transport.attemptCount.Add(1)) <= transport.failures {
		return nil, &net.OpError{Op: "dial", Net: "tcp", Err: &net.DNSError{Err: "server misbehaving", Name: "openrouter.ai", IsTemporary: true}}
	}
	return transport.next.RoundTrip(request)
}

func retryingDecisionModel(serverURL string, httpClient *http.Client, retryBaseDelay time.Duration) decisionModel {
	return decisionModel{
		endpoint:       Endpoint{URL: serverURL, ModelName: "test-model", APIKey: "test-key", HTTPClient: httpClient},
		retryBaseDelay: retryBaseDelay,
	}
}

func countingServer(statusCode int, counter *atomic.Int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		counter.Add(1)
		writer.WriteHeader(statusCode)
		writer.Write([]byte(`{"answers":{},"usage":{}}`))
	}))
}

func TestADecisionRetriesATransportFailureAndSucceeds(t *testing.T) {
	var served atomic.Int32
	server := countingServer(http.StatusOK, &served)
	defer server.Close()
	transport := &failingTransport{failures: 2, next: http.DefaultTransport}

	_, errorValue := retryingDecisionModel(server.URL, &http.Client{Transport: transport}, time.Millisecond).Decide(context.Background(), model.DecisionRequest{})

	if errorValue != nil {
		t.Fatalf("a DNS failure on the first dial must not end the turn: %v", errorValue)
	}
	if transport.attemptCount.Load() != 3 || served.Load() != 1 {
		t.Fatalf("expected 3 dials and 1 served request, got %d and %d", transport.attemptCount.Load(), served.Load())
	}
}

func TestADecisionTheEndpointRejectedIsNotRetried(t *testing.T) {
	var served atomic.Int32
	server := countingServer(http.StatusBadRequest, &served)
	defer server.Close()

	_, errorValue := retryingDecisionModel(server.URL, nil, time.Millisecond).Decide(context.Background(), model.DecisionRequest{})

	if errorValue == nil || served.Load() != 1 {
		t.Fatalf("a 400 names a defect in the request, got %d requests and error %v", served.Load(), errorValue)
	}
}

func TestADecisionGivesUpAfterTheRetryBudgetWithTheLastError(t *testing.T) {
	var served atomic.Int32
	server := countingServer(http.StatusServiceUnavailable, &served)
	defer server.Close()

	_, errorValue := retryingDecisionModel(server.URL, nil, time.Millisecond).Decide(context.Background(), model.DecisionRequest{})

	if errorValue == nil || !strings.Contains(errorValue.Error(), "503") {
		t.Fatalf("expected the last 503 to surface, got %v", errorValue)
	}
	if int(served.Load()) != 1+transientretry.Retries {
		t.Fatalf("expected %d attempts, got %d", 1+transientretry.Retries, served.Load())
	}
}

func TestADecisionRetriesRateLimiting(t *testing.T) {
	var served atomic.Int32
	server := countingServer(http.StatusTooManyRequests, &served)
	defer server.Close()

	retryingDecisionModel(server.URL, nil, time.Millisecond).Decide(context.Background(), model.DecisionRequest{})

	if int(served.Load()) != 1+transientretry.Retries {
		t.Fatalf("expected 429 to be retried to the budget, got %d attempts", served.Load())
	}
}

func TestAMalformedDecisionAnswerIsNotRetried(t *testing.T) {
	var served atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		served.Add(1)
		writer.Write([]byte(`not json`))
	}))
	defer server.Close()

	_, errorValue := retryingDecisionModel(server.URL, nil, time.Millisecond).Decide(context.Background(), model.DecisionRequest{})

	if errorValue == nil || served.Load() != 1 {
		t.Fatalf("an invalid answer is not a transient failure, got %d requests and error %v", served.Load(), errorValue)
	}
}

func TestACancelledDecisionStopsRetrying(t *testing.T) {
	var served atomic.Int32
	server := countingServer(http.StatusBadGateway, &served)
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	startedAt := time.Now()

	_, errorValue := retryingDecisionModel(server.URL, nil, time.Minute).Decide(ctx, model.DecisionRequest{})

	if errorValue == nil || served.Load() != 1 {
		t.Fatalf("a cancelled run must stop after the attempt in flight, got %d requests and %v", served.Load(), errorValue)
	}
	if time.Since(startedAt) > 10*time.Second {
		t.Fatal("cancellation must cut the backoff short")
	}
}
