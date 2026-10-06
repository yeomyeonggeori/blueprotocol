package agentcontract

import (
	"encoding/json"
	"testing"
)

func TestOlderIntakeStateWithoutIndependentWorkDefaultsFalse(t *testing.T) {
	var intakeDecision IntakeDecision
	if errorValue := json.Unmarshal([]byte(`{"classification":"needs_confirmation","level":"low"}`), &intakeDecision); errorValue != nil {
		t.Fatalf("expected older intake state to deserialize: %v", errorValue)
	}
	if intakeDecision.HasIndependentWork {
		t.Fatal("expected a missing independent-work fact to default to false")
	}
}
