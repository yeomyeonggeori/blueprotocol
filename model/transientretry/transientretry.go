package transientretry

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	Retries      = 3
	BaseDelay    = time.Second
	DelayCeiling = 30 * time.Second
)

type Outcome struct {
	IsTransient bool
	RetryAfter  time.Duration
}

func Do[Result any](ctx context.Context, baseDelay time.Duration, attempt func() (Result, Outcome, error)) (Result, error) {
	for attemptIndex := 0; ; attemptIndex++ {
		result, outcome, errorValue := attempt()
		if errorValue == nil {
			return result, nil
		}
		if attemptIndex >= Retries || !outcome.IsTransient || ctx.Err() != nil {
			return result, errorValue
		}
		if Wait(ctx, Delay(baseDelay, attemptIndex, outcome.RetryAfter)) != nil {
			return result, errorValue
		}
	}
}

func TransportOutcome(ctx context.Context) Outcome {
	return Outcome{IsTransient: ctx.Err() == nil}
}

func StatusOutcome(response *http.Response) Outcome {
	return Outcome{
		IsTransient: IsTransientStatus(response.StatusCode),
		RetryAfter:  RetryAfterDelay(response.Header.Get("Retry-After")),
	}
}

func IsTransientStatus(statusCode int) bool {
	return statusCode == http.StatusRequestTimeout ||
		statusCode == http.StatusTooManyRequests ||
		statusCode >= http.StatusInternalServerError
}

func RetryAfterDelay(headerValue string) time.Duration {
	seconds, errorValue := strconv.Atoi(strings.TrimSpace(headerValue))
	if errorValue != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func Delay(baseDelay time.Duration, attempt int, retryAfter time.Duration) time.Duration {
	delay := baseDelay << attempt
	if retryAfter > delay {
		delay = retryAfter
	}
	if delay > DelayCeiling {
		delay = DelayCeiling
	}
	return delay
}

func Wait(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
