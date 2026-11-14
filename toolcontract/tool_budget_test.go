package toolcontract

import (
	"context"
	"strings"
	"testing"
	"time"
)

func toolSetWithBudget(timeoutMS int, handler func(context.Context, ToolInvocation) (ToolResult, error)) *ToolSet {
	return toolSetWithBudgetAndSideEffect(timeoutMS, ToolSideEffectRead, handler)
}

func toolSetWithBudgetAndSideEffect(timeoutMS int, sideEffectClass string, handler func(context.Context, ToolInvocation) (ToolResult, error)) *ToolSet {
	toolSet := NewToolSet([]string{"slow_tool"})
	if errorValue := toolSet.RegisterTool(ToolDescriptor{
		Name:            "slow_tool",
		Description:     "a tool with its own budget",
		Visibility:      ToolVisibilityModel,
		ResultContract:  &ToolResultContract{Schema: []byte(`{"type":"object"}`)},
		SideEffectClass: sideEffectClass,
		TimeoutMS:       timeoutMS,
	}, handler); errorValue != nil {
		panic(errorValue)
	}
	return toolSet
}

func waitForTheDeadline(ctx context.Context, _ ToolInvocation) (ToolResult, error) {
	<-ctx.Done()
	return ToolResult{}, ctx.Err()
}

func expiredFailure(t *testing.T, sideEffectClass string) ToolFailure {
	t.Helper()
	toolSet := toolSetWithBudgetAndSideEffect(20, sideEffectClass, waitForTheDeadline)
	result, _ := toolSet.Invoke(context.Background(), ToolInvocation{ToolName: "slow_tool"})
	if !result.Failed() {
		t.Fatalf("side effect %q: a call that ran past the budget its own descriptor declared did not finish, so it is not a success", sideEffectClass)
	}
	if result.Output.Content != result.Failure.UserSafeSummary {
		t.Fatalf("side effect %q: the content and the summary tell two stories: %q and %q", sideEffectClass, result.Output.Content, result.Failure.UserSafeSummary)
	}
	return *result.Failure
}

func TestAToolThatOverrunsItsOwnBudgetFailsAsATimeout(t *testing.T) {
	failure := expiredFailure(t, ToolSideEffectRead)

	if failure.Code != FailureCodes.Timeout.String() {
		t.Fatalf("an agent deciding what to do next needs to know it ran out of time, got %q", failure.Code)
	}
	if failure.Stage != "slow_tool" {
		t.Fatalf("the failure must name the tool that ran out of time, got %q", failure.Stage)
	}
}

func TestAReadOnlyToolThatOverrunsItsBudgetIsRetryableAndClaimsNoEffect(t *testing.T) {
	for _, sideEffectClass := range []string{ToolSideEffectRead, ToolSideEffectNone, ToolSideEffectComputation, "read_only"} {
		failure := expiredFailure(t, sideEffectClass)

		if !failure.Retryable || failure.RetryPolicy == RetryPolicyDoNotRetry {
			t.Fatalf("side effect %q: asking again for something that changes nothing repeats no effect, got retryable=%v policy=%q", sideEffectClass, failure.Retryable, failure.RetryPolicy)
		}
		if strings.Contains(failure.UserSafeSummary, "taken effect") {
			t.Fatalf("side effect %q: a tool that changes nothing has no effect to have taken, got %q", sideEffectClass, failure.UserSafeSummary)
		}
	}
}

func TestAWritingToolThatOverrunsItsBudgetIsNotRetryableAndSaysTheWorkMayHaveTakenEffect(t *testing.T) {
	for _, sideEffectClass := range []string{ToolSideEffectExternalSend, ToolSideEffectStateChange, ToolSideEffectWorkspaceWrite, ToolSideEffectDestructive} {
		failure := expiredFailure(t, sideEffectClass)

		if failure.Retryable || failure.RetryPolicy != RetryPolicyDoNotRetry {
			t.Fatalf("side effect %q: the deadline only stopped the waiting, so a second call can send or click twice, got retryable=%v policy=%q", sideEffectClass, failure.Retryable, failure.RetryPolicy)
		}
		if !strings.Contains(failure.UserSafeSummary, "the work it started may still have taken effect") {
			t.Fatalf("side effect %q: the failure must say the work may have happened, got %q", sideEffectClass, failure.UserSafeSummary)
		}
		if strings.Contains(strings.ToLower(failure.UserSafeSummary), "retry") {
			t.Fatalf("side effect %q: retryability is carried by the structured fields, never by an instruction in the text, got %q", sideEffectClass, failure.UserSafeSummary)
		}
	}
}

func TestAToolDeclaringNoSideEffectExpiresAsAWritingTool(t *testing.T) {
	failure := expiredFailure(t, "")

	if failure.Retryable || failure.RetryPolicy != RetryPolicyDoNotRetry {
		t.Fatalf("a tool that never said it only reads may have written, got retryable=%v policy=%q", failure.Retryable, failure.RetryPolicy)
	}
	if !strings.Contains(failure.UserSafeSummary, "may still have taken effect") {
		t.Fatalf("a tool that never said it only reads may have written, got %q", failure.UserSafeSummary)
	}
}

func TestAnExpiredBudgetIsReportedInSeconds(t *testing.T) {
	if summary := expiredFailure(t, ToolSideEffectRead).UserSafeSummary; summary != "the tool did not finish within 0.02 seconds" {
		t.Fatalf("an invoked tool must report its budget in seconds, got %q", summary)
	}
	for timeoutMS, expectedSummary := range map[int]string{
		125000: "the tool did not finish within 125 seconds",
		1000:   "the tool did not finish within 1 second",
		1500:   "the tool did not finish within 1.5 seconds",
		250:    "the tool did not finish within 0.25 seconds",
	} {
		failure := toolBudgetExpiredFailure(ToolDefinition{Name: "slow_tool", SideEffectClass: ToolSideEffectRead}, timeoutMS)
		if failure.Failure.UserSafeSummary != expectedSummary {
			t.Fatalf("timeoutMS %d: expected %q, got %q", timeoutMS, expectedSummary, failure.Failure.UserSafeSummary)
		}
	}
	writingFailure := toolBudgetExpiredFailure(ToolDefinition{Name: "slow_tool", SideEffectClass: ToolSideEffectExternalSend}, 125000)
	if expectedSummary := "the tool did not finish within 125 seconds, and the work it started may still have taken effect"; writingFailure.Failure.UserSafeSummary != expectedSummary {
		t.Fatalf("expected %q, got %q", expectedSummary, writingFailure.Failure.UserSafeSummary)
	}
}

func TestAToolThatFinishesInsideItsBudgetIsUntouched(t *testing.T) {
	toolSet := toolSetWithBudget(2000, func(_ context.Context, _ ToolInvocation) (ToolResult, error) {
		return ToolResult{Output: ToolOutput{Content: "done", Data: []byte(`{}`)}}, nil
	})

	result, errorValue := toolSet.Invoke(context.Background(), ToolInvocation{ToolName: "slow_tool"})

	if errorValue != nil || result.Failed() {
		t.Fatalf("a tool inside its budget must return exactly what it produced, got %+v", result.Failure)
	}
	if result.Output.Content != "done" {
		t.Fatalf("the budget must not touch the result, got %q", result.Output.Content)
	}
}

func TestAToolDeclaringNoBudgetIsNeverDeadlined(t *testing.T) {
	toolSet := toolSetWithBudget(0, func(ctx context.Context, _ ToolInvocation) (ToolResult, error) {
		select {
		case <-ctx.Done():
			return ToolResult{}, ctx.Err()
		case <-time.After(30 * time.Millisecond):
			return ToolResult{Output: ToolOutput{Content: "slow but allowed", Data: []byte(`{}`)}}, nil
		}
	})

	result, _ := toolSet.Invoke(context.Background(), ToolInvocation{ToolName: "slow_tool"})

	if result.Failed() {
		t.Fatalf("a tool that declared no budget has none, got %+v", result.Failure)
	}
}

func TestTheCallersOwnCancellationIsNotReportedAsABudgetOverrun(t *testing.T) {
	toolSet := toolSetWithBudget(5000, func(ctx context.Context, _ ToolInvocation) (ToolResult, error) {
		<-ctx.Done()
		return ToolResult{}, ctx.Err()
	})
	callerContext, cancelCaller := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancelCaller()
	}()

	result, _ := toolSet.Invoke(callerContext, ToolInvocation{ToolName: "slow_tool"})

	if result.Failed() && result.Failure.Code == FailureCodes.Timeout.String() {
		t.Fatal("the caller walking away is not the tool being slow, and telling the agent otherwise sends it to retry a task nobody is waiting for")
	}
}

func TestAToolThatIgnoresItsDeadlineAndSucceedsKeepsItsSuccess(t *testing.T) {
	toolSet := toolSetWithBudgetAndSideEffect(20, ToolSideEffectExternalSend, func(_ context.Context, _ ToolInvocation) (ToolResult, error) {
		time.Sleep(60 * time.Millisecond)
		return ToolResult{Output: ToolOutput{Content: "sent", Data: []byte(`{}`)}}, nil
	})

	result, errorValue := toolSet.Invoke(context.Background(), ToolInvocation{ToolName: "slow_tool"})

	if errorValue != nil || result.Failed() {
		t.Fatalf("the message went out, and calling it unfinished invites sending it twice, got %+v, %v", result.Failure, errorValue)
	}
	if result.Output.Content != "sent" {
		t.Fatalf("a finished call's result must stand as the handler produced it, got %q", result.Output.Content)
	}
}

func TestAToolThatGivesUpAtItsDeadlineGetsTheExpiryFailure(t *testing.T) {
	toolSet := toolSetWithBudgetAndSideEffect(20, ToolSideEffectExternalSend, func(ctx context.Context, _ ToolInvocation) (ToolResult, error) {
		time.Sleep(60 * time.Millisecond)
		return ToolResult{}, ctx.Err()
	})

	result, errorValue := toolSet.Invoke(context.Background(), ToolInvocation{ToolName: "slow_tool"})

	if errorValue != nil {
		t.Fatalf("an expired budget is a tool result, not an invocation error, got %v", errorValue)
	}
	if !result.Failed() || result.Failure.Code != FailureCodes.Timeout.String() {
		t.Fatalf("a call that ended unfinished after its budget ran out must say it ran out of time, got %+v", result.Failure)
	}
}
