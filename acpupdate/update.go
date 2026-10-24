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
		return acp.SessionUpdate{ToolCall: &acp.SessionUpdateToolCall{
			ToolCallId: acp.ToolCallId(toolCallIDOfEvent(rawTurnEvent.Body)),
			Title:      ToolCallTitle(toolName, rawTurnEvent.Body),
			Status:     acp.ToolCallStatusPending,
			RawInput:   rawInputOfEvent(rawTurnEvent.Body),
			Meta:       meta,
		}}, true
	}
	if _, isResult := agentcontract.ToolTaskEventToolName(rawTurnEvent.Name, agentcontract.ToolTaskEventResultSuffix); isResult {
		status := acp.ToolCallStatusCompleted
		if isFailureEvent(rawTurnEvent.Body) {
			status = acp.ToolCallStatusFailed
		}
		return acp.SessionUpdate{ToolCallUpdate: &acp.SessionToolCallUpdate{
			ToolCallId: acp.ToolCallId(toolCallIDOfEvent(rawTurnEvent.Body)),
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

func toolCallIDOfEvent(body string) string {
	decoded := struct {
		ObservationID string `json:"observationID"`
	}{}
	json.Unmarshal([]byte(body), &decoded)
	return decoded.ObservationID
}

func rawInputOfEvent(body string) any {
	decoded := struct {
		Input json.RawMessage `json:"input"`
	}{}
	if json.Unmarshal([]byte(body), &decoded) != nil || len(decoded.Input) == 0 {
		return nil
	}
	return decoded.Input
}

func isFailureEvent(body string) bool {
	decoded := struct {
		Failure *json.RawMessage `json:"failure"`
	}{}
	return json.Unmarshal([]byte(body), &decoded) == nil && decoded.Failure != nil
}
