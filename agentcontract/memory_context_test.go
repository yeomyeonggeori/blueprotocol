package agentcontract

import (
	"strings"
	"testing"
	"time"
)

func TestARememberedFactRendersAsItsStatementAndItsDate(t *testing.T) {
	rendered := BuildMemoryContext([]MemoryFact{
		{Content: "이샘플 keeps the quarterly ledger in Numbers", ValidAt: time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)},
		{Content: "이샘플 prefers reviews on Friday afternoons"},
	})
	expected := "What you already know:\n" +
		"- [2026-03-15] 이샘플 keeps the quarterly ledger in Numbers\n" +
		"- 이샘플 prefers reviews on Friday afternoons"
	if rendered != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, rendered)
	}
}

func TestAnOverlongStatementIsCut(t *testing.T) {
	rendered := BuildMemoryContext([]MemoryFact{{Content: strings.Repeat("가", 300)}})
	if !strings.HasSuffix(rendered, "...") {
		t.Fatalf("expected an overlong statement to be cut, got %d characters", len([]rune(rendered)))
	}
}

func TestNothingRememberedRendersNothing(t *testing.T) {
	if rendered := BuildMemoryContext(nil); rendered != "" {
		t.Fatalf("expected no heading when there is nothing to say, got %q", rendered)
	}
	if rendered := BuildMemoryContext([]MemoryFact{{Content: "   "}}); rendered != "" {
		t.Fatalf("expected a blank statement to render nothing, got %q", rendered)
	}
}
