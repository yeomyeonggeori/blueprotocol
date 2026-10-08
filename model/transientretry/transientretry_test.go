package transientretry

import (
	"testing"
	"time"
)

func TestRetryDelayHonoursTheEndpointsRequestedWaitWithinTheCeiling(t *testing.T) {
	if delay := Delay(time.Second, 0, 5*time.Second); delay != 5*time.Second {
		t.Fatalf("an endpoint that names its recovery time knows it better than our backoff, got %v", delay)
	}
	if delay := Delay(time.Second, 0, time.Hour); delay != DelayCeiling {
		t.Fatalf("a server asking for an hour would silently eat the elapsed budget, got %v", delay)
	}
	if delay := Delay(time.Second, 2, 0); delay != 4*time.Second {
		t.Fatalf("expected exponential backoff, got %v", delay)
	}
}
