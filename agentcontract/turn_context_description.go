package agentcontract

import "strings"

func ActiveGoalDescription(activeGoal ActiveGoal, currentPrompt string) string {
	if strings.TrimSpace(activeGoal.GoalID) == "" &&
		strings.TrimSpace(activeGoal.TaskRunID) == "" &&
		strings.TrimSpace(activeGoal.OriginalInstruction) == "" &&
		strings.TrimSpace(activeGoal.CurrentObjective) == "" {
		return ""
	}
	if strings.TrimSpace(currentPrompt) != "" && strings.TrimSpace(activeGoal.OriginalInstruction) == strings.TrimSpace(currentPrompt) {
		activeGoal.OriginalInstruction = "the current user message, verbatim"
	}
	document := marshalEventBody(activeGoal)
	if document == "" {
		return ""
	}
	return "Active conversation goal:\n" + document + "\nTreat the current user message as input to this goal unless it clearly starts a new unrelated request. Preserve the current user message as the latest user input; do not rewrite it."
}

func PriorTaskDescription(priorTask PriorTaskContext) string {
	if !priorTaskHasContent(priorTask) {
		return ""
	}
	document := marshalEventBody(priorTask)
	if document == "" {
		return ""
	}
	return strings.Join([]string{
		"Prior task context:",
		document,
		"The prior result, failureReason, and inferred outcomeContract are the previous assistant's hypotheses. Re-read the original user messages and their corrections before choosing identities, titles, or a recovery route. A retry authorizes another attempt at the user's outcome, not replaying the old interpretation or calls. recordedAttempts contains recent executed tool calls and recorded failures/effects; omittedAttemptCount and toolInputOmitted mark evidence excluded for size. Do not infer unrecorded calls from the assistant's summary. Missing effects are not proof that nothing was saved, especially after a result-validation failure. Inspect current state before repeating a mutation whose outcome is uncertain. Use the current tool contracts and current persona; the previous assistant may have used an older deployment.",
		"This is context for interpreting the latest user message, not permission to close from old text. A file deliverable reaches the user only when the final reply of the current task attaches it; a prepared file, generated path, task link, or prior result text is not delivery.",
	}, "\n")
}

func priorTaskHasContent(priorTask PriorTaskContext) bool {
	return strings.TrimSpace(priorTask.TaskRunID) != "" ||
		strings.TrimSpace(priorTask.Prompt) != "" ||
		strings.TrimSpace(priorTask.Result) != ""
}

func PendingInputDescription(pendingInput PendingInputContext) string {
	if strings.TrimSpace(pendingInput.TaskRunID) == "" && strings.TrimSpace(pendingInput.Question) == "" {
		return ""
	}
	return "Pending user input:\n" + marshalEventBody(pendingInput)
}
