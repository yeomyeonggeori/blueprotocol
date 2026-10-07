package approvalcore

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/yeomyeonggeori/blueprotocol/agentcontract"
	"github.com/yeomyeonggeori/blueprotocol/holdrecord"
	"github.com/yeomyeonggeori/blueprotocol/taskstate"
	"github.com/yeomyeonggeori/blueprotocol/toolcontract"
)

var testEvents = EventNames{ConfirmationRequested: "test.confirmation_requested", AskRequested: "test.ask_requested", WordingFailed: "test.wording_failed"}

type scriptedHost struct {
	question   Question
	isAskable  bool
	verdict    Verdict
	askedHolds []holdrecord.Hold
	store      taskstate.TaskRunStore
	taskRunID  string
	decision   string
}

func (host *scriptedHost) Prepare(context.Context, Call) (Question, bool) {
	return host.question, host.isAskable
}

func (host *scriptedHost) Ask(_ context.Context, hold holdrecord.Hold) Verdict {
	host.askedHolds = append(host.askedHolds, hold)
	if host.decision != "" {
		holdrecord.Decide(host.store, host.taskRunID, hold.ID, host.decision, "scripted")
	}
	return host.verdict
}

type fixture struct {
	core      Core
	store     *taskstate.TaskRunService
	taskRunID string
	host      *scriptedHost
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	store := taskstate.NewTaskRunService(taskstate.NewTaskEventService())
	taskRunID := store.CreateTaskRun("person-1", "conversation-1", "delete it").TaskRunID
	host := &scriptedHost{question: Question{Text: "Delete the event?"}, isAskable: true, verdict: Unanswered, store: store, taskRunID: taskRunID}
	return fixture{core: New(store, testEvents), store: store, taskRunID: taskRunID, host: host}
}

func (fixture fixture) call(toolInput string) Call {
	return Call{TaskRunID: fixture.taskRunID, ToolName: "event_delete", ToolInput: json.RawMessage(toolInput)}
}

func (fixture fixture) await(call Call) Outcome {
	return fixture.core.Await(context.Background(), call, fixture.host)
}

func (fixture fixture) approveNext() {
	fixture.host.verdict = Approved
	fixture.host.decision = holdrecord.DecisionApprove
}

func (fixture fixture) eventBody(t *testing.T, eventName string) string {
	t.Helper()
	for _, taskEvent := range fixture.store.ListTaskEvent(fixture.taskRunID) {
		if taskEvent.Name == eventName {
			return taskEvent.Body
		}
	}
	t.Fatalf("expected a %s event, got none", eventName)
	return ""
}

func (fixture fixture) holds() []holdrecord.Hold {
	return holdrecord.Holds(fixture.store.ListTaskEvent(fixture.taskRunID))
}

func TestACallWithNoTaskRunIsUnanswerableAndNeverAsked(t *testing.T) {
	fixture := newFixture(t)

	outcome := fixture.await(Call{ToolName: "event_delete"})

	if outcome.Verdict != Unanswerable || len(fixture.host.askedHolds) != 0 {
		t.Fatalf("expected no one to be asked, got %+v after %d asks", outcome, len(fixture.host.askedHolds))
	}
}

func TestACallTheHostCannotPrepareOpensNoHold(t *testing.T) {
	fixture := newFixture(t)
	fixture.host.isAskable = false

	outcome := fixture.await(fixture.call(`{"eventID":"event-1"}`))

	if outcome.Verdict != Unanswerable || len(fixture.holds()) != 0 {
		t.Fatalf("expected an unaskable call to leave no hold, got %+v with holds %+v", outcome, fixture.holds())
	}
}

func TestAHoldCarriesTheCallTheQuestionAndTheChoicesItWasOpenedFor(t *testing.T) {
	fixture := newFixture(t)
	fixture.host.question = Question{Text: "Which one?", Choices: []holdrecord.Choice{{Key: "1", Label: "A"}}, ApprovedInput: json.RawMessage(`{"eventID":"event-9"}`)}

	fixture.await(fixture.call(`{"eventID":"event-1"}`))

	hold := fixture.holds()[0]
	if hold.Call.ToolName != "event_delete" || hold.Call.Confirmation != "Which one?" || len(hold.Choices) != 1 || string(hold.Call.ApprovedToolInput) != `{"eventID":"event-9"}` || !strings.Contains(string(hold.Call.ToolInput), "event-1") {
		t.Fatalf("expected the hold to carry what the requester was asked about, got %+v", hold)
	}
}

func TestAnApprovalIsSpentByTheCallItAnswered(t *testing.T) {
	fixture := newFixture(t)
	fixture.approveNext()

	first := fixture.await(fixture.call(`{"eventID":"event-1"}`))
	reissued := fixture.await(fixture.call(`{"eventID":"event-1"}`))

	if first.Verdict != Approved || reissued.Verdict != Approved {
		t.Fatalf("expected both calls to be approved, got %+v %+v", first, reissued)
	}
	if len(fixture.host.askedHolds) != 2 {
		t.Fatalf("expected the approval to be spent by the call it answered so the next identical call is asked again, got %d asks", len(fixture.host.askedHolds))
	}
}

func TestAnApprovedHoldAnotherAskerLeftIsSpentWithoutAskingAgain(t *testing.T) {
	fixture := newFixture(t)
	hold := holdrecord.Open(fixture.store, fixture.taskRunID, agentcontract.HeldCall{ToolName: "event_delete", ToolInput: json.RawMessage(`{"eventID":"event-1"}`), ApprovedToolInput: json.RawMessage(`{"eventID":"event-1","calendarID":"team"}`)}, nil)
	holdrecord.Decide(fixture.store, fixture.taskRunID, hold.ID, holdrecord.DecisionApprove, "harness_permission")

	outcome := fixture.await(fixture.call(`{ "eventID": "event-1" }`))

	if outcome.Verdict != Approved || outcome.HoldID != hold.ID || len(fixture.host.askedHolds) != 0 {
		t.Fatalf("expected the approved hold to answer the call unasked, got %+v after %d asks", outcome, len(fixture.host.askedHolds))
	}
	if spent := fixture.eventBody(t, agentcontract.TaskEventApprovalHoldSpent); !strings.Contains(spent, "calendarID") {
		t.Fatalf("expected the call to be spent with the input the requester approved, got %s", spent)
	}
}

func TestASpentHoldIsNeverReused(t *testing.T) {
	fixture := newFixture(t)
	fixture.approveNext()
	fixture.await(fixture.call(`{"eventID":"event-1"}`))
	fixture.host.verdict = Unanswered
	fixture.host.decision = ""

	outcome := fixture.await(fixture.call(`{"eventID":"event-1"}`))

	if outcome.Verdict == Approved {
		t.Fatal("expected one approval to authorise one call")
	}
}

func TestAnUnansweredHoldDoesNotAuthoriseTheReissue(t *testing.T) {
	fixture := newFixture(t)
	fixture.await(fixture.call(`{"eventID":"event-1"}`))

	if outcome := fixture.await(fixture.call(`{"eventID":"event-1"}`)); outcome.Verdict == Approved {
		t.Fatal("expected a hold nobody approved to authorise nothing")
	}
}

func TestARejectedCallIsNotSpent(t *testing.T) {
	fixture := newFixture(t)
	fixture.host.verdict = Rejected
	fixture.host.decision = holdrecord.DecisionReject

	outcome := fixture.await(fixture.call(`{"eventID":"event-1"}`))

	if outcome.Verdict != Rejected || fixture.holds()[0].State != holdrecord.StateRejected {
		t.Fatalf("expected a rejection to spend nothing, got %+v with %+v", outcome, fixture.holds())
	}
}

func TestAnApprovalDoesNotCarryOverToACallTheRequesterNeverSaw(t *testing.T) {
	fixture := newFixture(t)
	fixture.approveNext()
	fixture.await(fixture.call(`{"eventID":"event-1"}`))
	fixture.host.verdict = Unanswered
	fixture.host.decision = ""
	hold := holdrecord.Open(fixture.store, fixture.taskRunID, agentcontract.HeldCall{ToolName: "event_delete", ToolInput: json.RawMessage(`{"eventID":"event-1"}`)}, nil)
	holdrecord.Decide(fixture.store, fixture.taskRunID, hold.ID, holdrecord.DecisionApprove, "chat_reply")

	if outcome := fixture.await(fixture.call(`{"eventID":"event-2"}`)); outcome.Verdict == Approved {
		t.Fatalf("expected a substituted target to be asked about again, got %+v", outcome)
	}
}

func TestAGrantedScopeLetsTheNextCallInThatScopeRunUnasked(t *testing.T) {
	fixture := newFixture(t)
	scoped := fixture.call(`{"eventID":"event-1"}`)
	scoped.ApprovalScope = "calendar"
	fixture.approveNext()
	fixture.await(scoped)
	nextCall := fixture.call(`{"eventID":"event-2"}`)
	nextCall.ApprovalScope = "calendar"

	outcome := fixture.await(nextCall)

	if outcome.Verdict != Approved || len(fixture.host.askedHolds) != 1 {
		t.Fatalf("expected approving the scope to cover the next call unasked, got %+v after %d asks", outcome, len(fixture.host.askedHolds))
	}
}

func TestAGrantedScopeDoesNotCoverAnotherScope(t *testing.T) {
	fixture := newFixture(t)
	fixture.store.AppendTaskEvent(fixture.taskRunID, agentcontract.TaskEventApprovalScopeGranted, `{"scope":"messaging"}`)
	call := fixture.call(`{"eventID":"event-1"}`)
	call.ApprovalScope = "calendar"

	if outcome := fixture.await(call); outcome.Verdict == Approved {
		t.Fatalf("expected a grant to cover its own scope only, got %+v", outcome)
	}
}

func TestAHoldRecordsTheQuestionEventsUnderTheNamesTheHostChose(t *testing.T) {
	fixture := newFixture(t)
	call := fixture.call(`{"eventID":"event-1"}`)
	call.ResponseLanguage = "ko"
	call.SideEffectClass = "external_send"

	fixture.await(call)

	for _, eventName := range []string{testEvents.ConfirmationRequested, testEvents.AskRequested} {
		body := fixture.eventBody(t, eventName)
		for _, fragment := range []string{"Delete the event?", `"responseLanguage":"ko"`, "external_send"} {
			if !strings.Contains(body, fragment) {
				t.Fatalf("expected %q in %s: %s", fragment, eventName, body)
			}
		}
	}
}

func TestAScopedHoldRecordsTheScopeItWouldGrantAndAnUnscopedOneDoesNot(t *testing.T) {
	scoped := newFixture(t)
	call := scoped.call(`{"eventID":"event-1"}`)
	call.ApprovalScope = "calendar"
	scoped.await(call)
	unscoped := newFixture(t)
	unscoped.await(unscoped.call(`{"eventID":"event-1"}`))

	if body := scoped.eventBody(t, testEvents.AskRequested); !strings.Contains(body, `"approvalScope":"calendar"`) || !strings.Contains(body, `"sessionApprovable":true`) {
		t.Fatalf("expected the scope in the ask record, got %s", body)
	}
	if body := unscoped.eventBody(t, testEvents.AskRequested); strings.Contains(body, "approvalScope") || strings.Contains(body, "sessionApprovable") {
		t.Fatalf("expected no scope offered, got %s", body)
	}
}

type fixedWorder struct{ wording holdrecord.QuestionWording }

func (worder fixedWorder) WordQuestion(context.Context, holdrecord.QuestionFacts) holdrecord.QuestionWording {
	return worder.wording
}

func TestAWordingFailureIsRecordedAndTheFallbackTextIsKept(t *testing.T) {
	fixture := newFixture(t)
	worder := fixedWorder{holdrecord.QuestionWording{Text: "event_delete {}", Failure: errors.New("the model is unreachable")}}

	text := fixture.core.Word(context.Background(), worder, fixture.call(`{}`), holdrecord.QuestionFacts{})

	if text != "event_delete {}" {
		t.Fatalf("expected the worder's fallback text, got %q", text)
	}
	if body := fixture.eventBody(t, testEvents.WordingFailed); !strings.Contains(body, "event_delete") || !strings.Contains(body, "the model is unreachable") {
		t.Fatalf("expected the failure and the tool in the event, got %s", body)
	}
}

func TestAMissingWorderFallsBackToTheToolNameAndSaysSo(t *testing.T) {
	fixture := newFixture(t)

	text := fixture.core.Word(context.Background(), nil, fixture.call(`{}`), holdrecord.QuestionFacts{})

	if text != "event_delete" {
		t.Fatalf("expected the tool name, got %q", text)
	}
	if body := fixture.eventBody(t, testEvents.WordingFailed); !strings.Contains(body, "none is configured") {
		t.Fatalf("expected the missing worder to be recorded, got %s", body)
	}
}

func TestARefusalIsAPolicyBlockedFailureTheModelIsToldNotToRetry(t *testing.T) {
	for name, result := range map[string]toolcontract.ToolResult{"declined": Declined(), "delegated turn": DelegatedTurn(), "other": Refusal("no one to ask")} {
		if result.Failure == nil || toolcontract.FailureCode(result.Failure.Code) != toolcontract.FailureCodes.PolicyBlocked {
			t.Fatalf("%s: expected a policy-blocked failure, got %+v", name, result)
		}
	}
	if !strings.Contains(Declined().Failure.UserSafeSummary, "Do not retry") || !strings.Contains(DelegatedTurn().Failure.UserSafeSummary, "delegated turn") {
		t.Fatalf("expected the notices to say what the model should do, got %+v %+v", Declined().Failure, DelegatedTurn().Failure)
	}
}
