package agentcontract

import (
	"errors"
)

type IntakeClassification string
type TaskShape string
type TurnRoute string
type ApprovalSignal string
type PriorTaskReference string
type DeliverableKind string
type ExpectedToolCount string

const (
	IntakeClassificationQuickReply        IntakeClassification = "quick_reply"
	IntakeClassificationBoundedTask       IntakeClassification = "bounded_task"
	IntakeClassificationNeedsConfirmation IntakeClassification = "needs_confirmation"
	IntakeClassificationUnsupported       IntakeClassification = "unsupported"

	TaskShapeImmediateReply    TaskShape = "immediate_reply"
	TaskShapeResearchTask      TaskShape = "research_task"
	TaskShapeMaintenanceTask   TaskShape = "maintenance_task"
	TaskShapeScheduledTask     TaskShape = "scheduled_task"
	TaskShapeApprovalGatedTask TaskShape = "approval_gated_task"

	TurnRouteContinueTask   TurnRoute = "continue_task"
	TurnRouteReviseTask     TurnRoute = "revise_task"
	TurnRouteAnswerQuestion TurnRoute = "answer_question"
	TurnRouteStartTask      TurnRoute = "start_task"
	TurnRouteAnswerMeta     TurnRoute = "answer_meta"
	TurnRouteClarify        TurnRoute = "clarify"
	TurnRouteConsume        TurnRoute = "consume"
	TurnRouteGiveUp         TurnRoute = "give_up"

	ExpectedToolCountNone    ExpectedToolCount = "none"
	ExpectedToolCountOne     ExpectedToolCount = "one"
	ExpectedToolCountSeveral ExpectedToolCount = "several"

	PriorTaskReferenceNone            PriorTaskReference = "none"
	PriorTaskReferenceOutcomeRecovery PriorTaskReference = "outcome_recovery"

	ApprovalSignalApprove ApprovalSignal = "approve"
	ApprovalSignalReject  ApprovalSignal = "reject"

	DeliverableKindPresentation DeliverableKind = "presentation"
	DeliverableKindDocument     DeliverableKind = "document"
	DeliverableKindNone         DeliverableKind = "none"
)

type ClarificationOption struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Value string `json:"value,omitempty"`
}

type IntakeDecision struct {
	Classification          IntakeClassification  `json:"classification"`
	TaskShape               TaskShape             `json:"taskShape"`
	TaskLevel               TaskLevel             `json:"level"`
	RequestedOutputFormats  []string              `json:"requestedOutputFormats"`
	DeliverableKind         DeliverableKind       `json:"deliverableKind,omitempty"`
	ExpectedResults         []ExpectedResult      `json:"expectedResults,omitempty"`
	ResponseLanguage        string                `json:"responseLanguage"`
	Reason                  string                `json:"reason"`
	UserFacingReply         string                `json:"userFacingReply"`
	IsExternalSendRequested bool                  `json:"isExternalSendRequested"`
	InitialToolNames        []string              `json:"initialToolNames,omitempty"`
	PriorTaskReference      PriorTaskReference    `json:"priorTaskReference,omitempty"`
	ClarificationQuestion   string                `json:"clarificationQuestion,omitempty"`
	ClarificationOptions    []ClarificationOption `json:"clarificationOptions,omitempty"`
	HasIndependentWork      bool                  `json:"hasIndependentWork"`
	ExpectedToolCount       ExpectedToolCount     `json:"expectedToolCount,omitempty"`
	RawDecisionRoute        TurnRoute             `json:"rawDecisionRoute,omitempty"`
}

func (intakeDecision IntakeDecision) Validate() error {
	if NormalizeIntakeClassification(intakeDecision.Classification) == "" {
		return errors.New("intake classification is invalid")
	}
	if NormalizeTaskLevel(string(intakeDecision.TaskLevel)) == "" {
		return errors.New("intake task level is invalid")
	}
	return nil
}

func NormalizeIntakeClassification(classification IntakeClassification) IntakeClassification {
	if IsIntakeClassificationName(string(classification)) {
		return classification
	}
	return ""
}

var TurnRouteNames = []string{
	string(TurnRouteConsume), string(TurnRouteAnswerQuestion), string(TurnRouteAnswerMeta), string(TurnRouteClarify),
	string(TurnRouteStartTask), string(TurnRouteContinueTask), string(TurnRouteReviseTask), string(TurnRouteGiveUp),
}

var IntakeClassificationNames = []string{
	string(IntakeClassificationQuickReply), string(IntakeClassificationBoundedTask),
	string(IntakeClassificationNeedsConfirmation), string(IntakeClassificationUnsupported),
}

var TaskShapeNames = []string{
	string(TaskShapeImmediateReply), string(TaskShapeResearchTask), string(TaskShapeMaintenanceTask),
	string(TaskShapeScheduledTask), string(TaskShapeApprovalGatedTask),
}

var DeliverableKindNames = []string{
	string(DeliverableKindPresentation), string(DeliverableKindDocument), string(DeliverableKindNone),
}

var ApprovalSignalNames = []string{
	string(ApprovalSignalApprove), string(ApprovalSignalReject),
}

var PriorTaskReferenceNames = []string{string(PriorTaskReferenceOutcomeRecovery), string(PriorTaskReferenceNone)}

func IsTurnRouteName(name string) bool {
	return isListedName(TurnRouteNames, name)
}

func IsIntakeClassificationName(name string) bool {
	return isListedName(IntakeClassificationNames, name)
}

func IsTaskShapeName(name string) bool {
	return isListedName(TaskShapeNames, name)
}

func IsDeliverableKindName(name string) bool {
	return isListedName(DeliverableKindNames, name)
}

func IsApprovalSignalName(name string) bool {
	return isListedName(ApprovalSignalNames, name)
}

func IsPriorTaskReferenceName(name string) bool {
	return isListedName(PriorTaskReferenceNames, name)
}

func isListedName(names []string, name string) bool {
	for _, listedName := range names {
		if listedName == name {
			return true
		}
	}
	return false
}
