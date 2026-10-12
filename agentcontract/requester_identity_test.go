package agentcontract

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestRequesterIdentityPreservesLegacyTurnWireFields(t *testing.T) {
	legacy := []byte(`{"RequesterPersonID":"sample-person-id","RequesterEmail":"sample@example.com","RequesterName":"이샘플","RequesterPlatformUserID":"sample-platform-id","RequesterCallingName":"샘플 님","RequesterHandle":"sample","RequesterCircles":["member"]}`)
	var request AgentTurnRequest
	if errorValue := json.Unmarshal(legacy, &request); errorValue != nil {
		t.Fatal(errorValue)
	}
	encoded, errorValue := json.Marshal(request)
	if errorValue != nil {
		t.Fatal(errorValue)
	}
	var expected, actual map[string]json.RawMessage
	if errorValue := json.Unmarshal(legacy, &expected); errorValue != nil {
		t.Fatal(errorValue)
	}
	if errorValue := json.Unmarshal(encoded, &actual); errorValue != nil {
		t.Fatal(errorValue)
	}
	if _, hasNestedIdentity := actual["RequesterIdentity"]; hasNestedIdentity {
		t.Fatal("persisted turns must retain their flat identity fields")
	}
	for name, value := range expected {
		if !bytes.Equal(actual[name], value) {
			t.Fatalf("legacy field %s changed: %s", name, actual[name])
		}
	}
}

func TestRequesterIdentityCloneKeepsCirclesIndependent(t *testing.T) {
	original := RequesterIdentity{RequesterPersonID: "sample-person-id", RequesterCircles: []string{"member"}}
	cloned := original.Clone()
	cloned.RequesterCircles[0] = "sample-circle"
	if original.RequesterCircles[0] != "member" || cloned.RequesterPersonID != original.RequesterPersonID {
		t.Fatal("changing a copied identity must not change the authenticated source")
	}
	if (RequesterIdentity{}).Clone().RequesterCircles != nil {
		t.Fatal("an absent circle list must remain absent")
	}
}

func TestRoutingRequestPreservesTheCompleteRequester(t *testing.T) {
	request := AgentTurnRequest{
		RequesterIdentity: RequesterIdentity{
			RequesterPersonID:       "sample-person-id",
			RequesterEmail:          "sample@example.com",
			RequesterName:           "이샘플",
			RequesterPlatformUserID: "sample-platform-id",
			RequesterCallingName:    "샘플 님",
			RequesterHandle:         "sample",
			RequesterCircles:        []string{"member"},
		},
		Platform: "virtual",
	}
	routing := request.RoutingRequest()
	if !reflect.DeepEqual(routing.RequesterIdentity, request.RequesterIdentity) || routing.Platform != request.Platform {
		t.Fatal("routing must preserve the authenticated requester")
	}
	routing.RequesterCircles[0] = "sample-circle"
	if request.RequesterCircles[0] != "member" {
		t.Fatal("routing must preserve the host circle snapshot")
	}
}
