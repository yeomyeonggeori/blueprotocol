package agentcontract

import (
	"encoding/json"

	"github.com/yeomyeonggeori/blueprotocol/toolcontract"
)

type ToolCallRequested struct {
	ObservationID   string          `json:"observationID"`
	ToolName        string          `json:"toolName"`
	Input           json.RawMessage `json:"input,omitempty"`
	SideEffectClass string          `json:"sideEffectClass,omitempty"`
}

type ToolCallResult struct {
	ObservationID string                        `json:"observationID"`
	ToolInput     json.RawMessage               `json:"toolInput,omitempty"`
	Summary       string                        `json:"summary,omitempty"`
	Failure       *toolcontract.ToolFailure     `json:"failure,omitempty"`
	Effects       []toolcontract.ResourceEffect `json:"effects,omitempty"`
}

type ToolCallCancelled struct {
	ObservationID  string `json:"observationID"`
	ToolName       string `json:"toolName"`
	TaskAttemptID  string `json:"taskAttemptID,omitempty"`
	Reason         string `json:"reason,omitempty"`
	TerminalStatus string `json:"terminalStatus,omitempty"`
}

func ToolCallRecordOf[Record ToolCallRequested | ToolCallResult | ToolCallCancelled](eventBody string) Record {
	var record Record
	_ = json.Unmarshal([]byte(eventBody), &record)
	return record
}

func (result ToolCallResult) Succeeded() bool {
	return result.Failure == nil
}

func (result ToolCallResult) Held() bool {
	return result.Failure != nil && result.Failure.RequiresApproval
}
