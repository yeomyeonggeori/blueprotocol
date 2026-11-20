package agentcontract

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/yeomyeonggeori/blueprotocol/model"
)

func TestFailureCauseFollowsTheEndpointStatusThroughWrapping(t *testing.T) {
	cases := []struct {
		statusCode    int
		expectedCause FailureCause
	}{
		{http.StatusBadGateway, FailureCauseTransient},
		{http.StatusServiceUnavailable, FailureCauseTransient},
		{http.StatusTooManyRequests, FailureCauseTransient},
		{http.StatusRequestTimeout, FailureCauseTransient},
		{0, FailureCauseTransient},
		{http.StatusUnauthorized, FailureCauseAdministrator},
		{http.StatusPaymentRequired, FailureCauseAdministrator},
		{http.StatusForbidden, FailureCauseAdministrator},
		{http.StatusBadRequest, ""},
		{http.StatusNotFound, ""},
	}
	for _, testCase := range cases {
		errorValue := fmt.Errorf("turn router: %w", model.EndpointError{StatusCode: testCase.statusCode, Reason: errors.New("endpoint failed")})
		if cause := FailureCauseOf(errorValue); cause != testCase.expectedCause {
			t.Errorf("status %d: expected cause %q, got %q", testCase.statusCode, testCase.expectedCause, cause)
		}
	}
}

func TestAFailureNoEndpointAnsweredHasNoCause(t *testing.T) {
	if cause := FailureCauseOf(errors.New("the spreadsheet tool could not open the file")); cause != "" {
		t.Fatalf("only an endpoint's answer names a cause, got %q", cause)
	}
}

type causeCarryingError struct {
	cause FailureCause
}

func (carryingError causeCarryingError) Error() string {
	return "the harness failed"
}

func (carryingError causeCarryingError) FailureCause() FailureCause {
	return carryingError.cause
}

func TestACauseCarriedAcrossAProcessBoundaryIsRead(t *testing.T) {
	errorValue := fmt.Errorf("prompt: %w", causeCarryingError{cause: FailureCauseAdministrator})

	if cause := FailureCauseOf(errorValue); cause != FailureCauseAdministrator {
		t.Fatalf("expected the carried cause, got %q", cause)
	}
}

func TestOnlyANamedCauseParses(t *testing.T) {
	for value, expectedCause := range map[string]FailureCause{
		"transient":     FailureCauseTransient,
		"administrator": FailureCauseAdministrator,
		"Transient":     "",
		"":              "",
		"502":           "",
	} {
		if cause := ParseFailureCause(value); cause != expectedCause {
			t.Errorf("%q: expected %q, got %q", value, expectedCause, cause)
		}
	}
}

func TestARawNoticeCarriesTheCauseOfItsReport(t *testing.T) {
	report := FailureReport{RawError: "turn router: decisions endpoint answered 502 Bad Gateway", Cause: FailureCauseTransient}

	notice, status := (FailureNoticeGenerator{}).Generate(context.Background(), report)

	if status.Source != "raw_error" || notice.Cause != FailureCauseTransient {
		t.Fatalf("a notice no model could word must say why it failed in a form the screen can read, got %+v from %+v", notice, status)
	}
}
