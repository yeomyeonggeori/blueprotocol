package approvalcore

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/yeomyeonggeori/blueprotocol/agentcontract"
	"github.com/yeomyeonggeori/blueprotocol/holdrecord"
	"github.com/yeomyeonggeori/blueprotocol/taskstate"
)

type Call struct {
	TaskRunID        string
	ToolName         string
	ToolInput        json.RawMessage
	ApprovalScope    string
	SideEffectClass  string
	ResponseLanguage string
	HarnessSession   agentcontract.HarnessSession
}

type Verdict string

const (
	Approved     Verdict = "approved"
	Rejected     Verdict = "rejected"
	Unanswered   Verdict = ""
	Unanswerable Verdict = "unanswerable"
)

type Outcome struct {
	Verdict Verdict
	HoldID  string
}

type Question struct {
	Text          string
	Choices       []holdrecord.Choice
	ApprovedInput json.RawMessage
}

type Host interface {
	Prepare(ctx context.Context, call Call) (Question, bool)
	Ask(ctx context.Context, hold holdrecord.Hold) Verdict
}

type EventNames struct {
	ConfirmationRequested string
	AskRequested          string
	WordingFailed         string
}

type Core struct {
	taskRuns taskstate.TaskRunStore
	events   EventNames
}

func New(taskRuns taskstate.TaskRunStore, events EventNames) Core {
	return Core{taskRuns: taskRuns, events: events}
}

func (core Core) Await(ctx context.Context, call Call, host Host) Outcome {
	if call.TaskRunID == "" {
		return Outcome{Verdict: Unanswerable}
	}
	if holdID, isSpent := core.spendApproved(call); isSpent {
		return Outcome{Verdict: Approved, HoldID: holdID}
	}
	return core.holdAndAsk(ctx, call, host)
}

func (core Core) holdAndAsk(ctx context.Context, call Call, host Host) Outcome {
	question, isAskable := host.Prepare(ctx, call)
	if !isAskable {
		return Outcome{Verdict: Unanswerable}
	}
	hold := core.open(call, question)
	if verdict := host.Ask(ctx, hold); verdict != Approved {
		return Outcome{Verdict: verdict}
	}
	holdrecord.Spend(core.taskRuns, call.TaskRunID, hold.ID, call.ToolName, hold.Call.ApprovedInput())
	return Outcome{Verdict: Approved, HoldID: hold.ID}
}

func (core Core) spendApproved(call Call) (string, bool) {
	ledger := holdrecord.LedgerOf(core.taskRuns.ListTaskEvent(call.TaskRunID))
	hold, isHeld := holdrecord.ApprovedHoldForCall(ledger.Holds, call.ToolName, call.ToolInput)
	if !isHeld && !ledger.GrantsScope(call.ApprovalScope) {
		return "", false
	}
	toolInput := call.ToolInput
	if isHeld {
		toolInput = hold.Call.ApprovedInput()
	}
	holdrecord.Spend(core.taskRuns, call.TaskRunID, hold.ID, call.ToolName, toolInput)
	return hold.ID, true
}

var errNoQuestionWorder = errors.New("approval wording needs a question worder and none is configured")

func (core Core) Word(ctx context.Context, worder holdrecord.QuestionWorder, call Call, facts holdrecord.QuestionFacts) string {
	wording := holdrecord.QuestionWording{Text: strings.TrimSpace(call.ToolName), Failure: errNoQuestionWorder}
	if worder != nil {
		wording = worder.WordQuestion(ctx, facts)
	}
	if wording.Failure != nil {
		core.Record(call.TaskRunID, core.events.WordingFailed, map[string]string{"toolName": strings.TrimSpace(call.ToolName), "error": wording.Failure.Error()})
	}
	return wording.Text
}

func (core Core) open(call Call, question Question) holdrecord.Hold {
	hold := holdrecord.Open(core.taskRuns, call.TaskRunID, agentcontract.HeldCall{
		ToolName:          call.ToolName,
		ToolInput:         call.ToolInput,
		ApprovedToolInput: question.ApprovedInput,
		ApprovalScope:     strings.TrimSpace(call.ApprovalScope),
		Confirmation:      question.Text,
		HarnessSession:    call.HarnessSession,
	}, question.Choices)
	core.Record(call.TaskRunID, core.events.ConfirmationRequested, confirmationRecord(call, question.Text))
	core.Record(call.TaskRunID, core.events.AskRequested, askRecord(call, question.Text))
	return hold
}

func (core Core) Record(taskRunID string, eventName string, body any) {
	document, errorValue := json.Marshal(body)
	if errorValue == nil {
		core.taskRuns.AppendTaskEvent(taskRunID, eventName, string(document))
	}
}

func confirmationRecord(call Call, question string) map[string]string {
	return map[string]string{
		"userFacingMessage": question,
		"message":           question,
		"reasonCode":        reasonCode(call),
		"reasonDetail":      "approval gate for " + call.ToolName,
		"responseLanguage":  call.ResponseLanguage,
		"source":            "tool_catalog",
	}
}

func askRecord(call Call, question string) map[string]any {
	record := map[string]any{
		"kind":             "ask_confirm",
		"message":          question,
		"reasonCode":       reasonCode(call),
		"reasonDetail":     "approval gate for " + call.ToolName,
		"responseLanguage": call.ResponseLanguage,
	}
	if approvalScope := strings.TrimSpace(call.ApprovalScope); approvalScope != "" {
		record["approvalScope"] = approvalScope
		record["sessionApprovable"] = true
	}
	return record
}

func reasonCode(call Call) string {
	if sideEffectClass := strings.TrimSpace(call.SideEffectClass); sideEffectClass != "" {
		return sideEffectClass
	}
	return "approval_required"
}
