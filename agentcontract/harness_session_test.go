package agentcontract

import "testing"

func TestTheLatestSessionAHarnessOpenedIsTheOneItResumes(t *testing.T) {
	taskEvents := []TaskEvent{
		{Name: TaskEventHarnessSessionOpened, Body: HarnessSessionOpened{HarnessName: "codex", SessionID: "first"}.Body()},
		{Name: TaskEventHarnessSessionOpened, Body: HarnessSessionOpened{HarnessName: "claude-code", SessionID: "other-harness"}.Body()},
		{Name: TaskEventHarnessSessionOpened, Body: HarnessSessionOpened{HarnessName: "codex", SessionID: "second"}.Body()},
	}

	if sessionID := LatestHarnessSessionID(taskEvents, "codex"); sessionID != "second" {
		t.Fatalf("a harness resumes the session it opened last, got %q", sessionID)
	}
	if sessionID := LatestHarnessSessionID(taskEvents, "antigravity"); sessionID != "" {
		t.Fatalf("a harness never resumes a session another harness opened, got %q", sessionID)
	}
}
