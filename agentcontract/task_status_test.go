package agentcontract

import "testing"

func TestOnlyARunHeldForApprovalOrInputAwaitsAnAnswer(t *testing.T) {
	for status, expected := range map[TaskStatus]bool{
		TaskStatusWaitingApproval:  true,
		TaskStatusWaitingUserInput: true,
		TaskStatusBlocked:          false,
		TaskStatusRunning:          false,
		TaskStatusInterrupted:      false,
		TaskStatusCompleted:        false,
	} {
		if status.IsAwaitingAnAnswer() != expected {
			t.Fatalf("%s awaiting an answer: expected %v", status, expected)
		}
	}
}
