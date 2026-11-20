package acpfailure

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	acp "github.com/coder/acp-go-sdk"
	"github.com/yeomyeonggeori/blueprotocol/agentcontract"
	"github.com/yeomyeonggeori/blueprotocol/model"
)

func TestTheCauseOfAFailedPromptSurvivesTheWire(t *testing.T) {
	failure := fmt.Errorf("turn router: %w", model.EndpointError{StatusCode: http.StatusBadGateway, Reason: errors.New("decisions endpoint answered 502 Bad Gateway")})

	received := throughTheWire(t, PromptError(failure))

	if cause := CauseOf(received); cause != agentcontract.FailureCauseTransient {
		t.Fatalf("expected the transient cause on the far side, got %q from %+v", cause, received)
	}
	if reason, _ := received.Data.(map[string]any)["error"].(string); reason != failure.Error() {
		t.Fatalf("the reason must cross unchanged, got %q", reason)
	}
}

func TestAFailureWithoutACauseCrossesAsItWas(t *testing.T) {
	failure := errors.New("the workspace could not be opened")

	if PromptError(failure) != failure {
		t.Fatal("a failure no endpoint answered must cross the way the SDK already sends it")
	}
}

func TestAnUnknownCauseOnTheWireReadsAsNone(t *testing.T) {
	received := &acp.RequestError{Data: map[string]any{"error": "failed", "cause": "the network"}}

	if cause := CauseOf(received); cause != "" {
		t.Fatalf("only a named cause is read, got %q", cause)
	}
}

func throughTheWire(t *testing.T, errorValue error) *acp.RequestError {
	t.Helper()
	var sent *acp.RequestError
	if !errors.As(errorValue, &sent) {
		t.Fatalf("expected an ACP request error, got %#v", errorValue)
	}
	body, errorValue := json.Marshal(sent)
	if errorValue != nil {
		t.Fatal(errorValue)
	}
	var received acp.RequestError
	if errorValue := json.Unmarshal(body, &received); errorValue != nil {
		t.Fatal(errorValue)
	}
	return &received
}
