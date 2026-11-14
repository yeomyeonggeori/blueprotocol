package acpupdate

import (
	"encoding/json"

	acp "github.com/coder/acp-go-sdk"
	"github.com/yeomyeonggeori/blueprotocol/agentcontract"
	"github.com/yeomyeonggeori/blueprotocol/taskstate"
)

func ForEvent(rawTurnEvent taskstate.RawTurnEvent, position int) acp.SessionUpdate {
	meta := ledgerMeta(LedgerRecordOf(rawTurnEvent, position))
	if toolCallUpdate, isToolCall := toolCallFor(rawTurnEvent, meta); isToolCall {
		return toolCallUpdate
	}
	return acp.SessionUpdate{AgentThoughtChunk: &acp.SessionUpdateAgentThoughtChunk{
		Content: acp.TextBlock(rawTurnEvent.Name),
		Meta:    meta,
	}}
}

func ToolCallForEvent(rawTurnEvent taskstate.RawTurnEvent) (acp.SessionUpdate, bool) {
	return toolCallFor(rawTurnEvent, ledgerMeta(LedgerRecordOf(rawTurnEvent, 0)))
}

func toolCallFor(rawTurnEvent taskstate.RawTurnEvent, meta map[string]any) (acp.SessionUpdate, bool) {
	if toolName, isRequest := agentcontract.ToolTaskEventToolName(rawTurnEvent.Name, agentcontract.ToolTaskEventRequestedSuffix); isRequest {
		requested := agentcontract.ToolCallRecordOf[agentcontract.ToolCallRequested](rawTurnEvent.Body)
		return acp.SessionUpdate{ToolCall: &acp.SessionUpdateToolCall{
			ToolCallId: acp.ToolCallId(requested.ObservationID),
			Title:      ToolCallTitle(toolName, requested.Input),
			Status:     acp.ToolCallStatusInProgress,
			RawInput:   rawInputOf(requested.Input),
			Meta:       meta,
		}}, true
	}
	if _, isResult := agentcontract.ToolTaskEventToolName(rawTurnEvent.Name, agentcontract.ToolTaskEventResultSuffix); isResult {
		result := agentcontract.ToolCallRecordOf[agentcontract.ToolCallResult](rawTurnEvent.Body)
		status := acp.ToolCallStatusCompleted
		if !result.Succeeded() {
			status = acp.ToolCallStatusFailed
		}
		return acp.SessionUpdate{ToolCallUpdate: &acp.SessionToolCallUpdate{
			ToolCallId: acp.ToolCallId(result.ObservationID),
			Status:     &status,
			RawOutput:  json.RawMessage(rawTurnEvent.Body),
			Meta:       meta,
		}}, true
	}
	return acp.SessionUpdate{}, false
}

func LedgerRecordOf(rawTurnEvent taskstate.RawTurnEvent, position int) agentcontract.LedgerRecord {
	record := agentcontract.LedgerRecord{Name: rawTurnEvent.Name, Text: rawTurnEvent.Body, Position: position}
	if json.Valid([]byte(rawTurnEvent.Body)) {
		record.Body = json.RawMessage(rawTurnEvent.Body)
	} else {
		quoted, _ := json.Marshal(rawTurnEvent.Body)
		record.Body = quoted
	}
	return record
}

func ledgerMeta(record agentcontract.LedgerRecord) map[string]any {
	return map[string]any{agentcontract.LedgerMetaKey: record}
}

func rawInputOf(input json.RawMessage) any {
	if len(input) == 0 {
		return nil
	}
	return input
}
