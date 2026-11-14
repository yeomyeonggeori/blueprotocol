package agentcontract

import "testing"

func TestARecordedToolCallResultTellsSuccessFailureAndHoldApart(t *testing.T) {
	succeeded := ToolCallRecordOf[ToolCallResult](`{"observationID":"call-1","summary":"보냈습니다"}`)
	failed := ToolCallRecordOf[ToolCallResult](`{"observationID":"call-2","failure":{"kind":"unknown","code":"operation_failed"}}`)
	held := ToolCallRecordOf[ToolCallResult](`{"observationID":"call-3","failure":{"kind":"policy","code":"approval_required","requiresApproval":true}}`)

	if !succeeded.Succeeded() || succeeded.Held() {
		t.Fatalf("expected a call without a failure to have succeeded, got %+v", succeeded)
	}
	if failed.Succeeded() || failed.Held() {
		t.Fatalf("expected an operation failure to be a failure and not a hold, got %+v", failed)
	}
	if held.Succeeded() || !held.Held() {
		t.Fatalf("expected a call waiting for approval to be held, got %+v", held)
	}
}

func TestAnUnreadableToolCallRecordDecodesAsEmpty(t *testing.T) {
	if requested := ToolCallRecordOf[ToolCallRequested]("not json"); requested.ObservationID != "" || requested.ToolName != "" {
		t.Fatalf("expected an unreadable body to decode as an empty record, got %+v", requested)
	}
}
