package agentcontract

import "testing"

func TestATurnWorksInItsConversationsDirectoryAndOtherwiseAtTheWorkspaceRoot(t *testing.T) {
	if path := (AgentTurnRequest{WorkspaceRootPath: "/workspace", WorkspaceDefaultPath: "/workspace/private/people/person-1"}).WorkingDirectoryPath(); path != "/workspace/private/people/person-1" {
		t.Fatalf("a turn works in its conversation's directory, got %q", path)
	}
	if path := (AgentTurnRequest{WorkspaceRootPath: "/workspace"}).WorkingDirectoryPath(); path != "/workspace" {
		t.Fatalf("a turn with no conversation directory works at the workspace root, got %q", path)
	}
}
