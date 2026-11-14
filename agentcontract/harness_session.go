package agentcontract

import (
	"encoding/json"
	"strings"
)

type HarnessSessionOpened struct {
	HarnessName   string `json:"harnessName"`
	SessionID     string `json:"sessionID"`
	ResumeFailure string `json:"resumeFailure,omitempty"`
}

func (opened HarnessSessionOpened) Body() string {
	return marshalEventBody(opened)
}

func LatestHarnessSessionID(taskEvents []TaskEvent, harnessName string) string {
	for index := len(taskEvents) - 1; index >= 0; index-- {
		if taskEvents[index].Name != TaskEventHarnessSessionOpened {
			continue
		}
		var opened HarnessSessionOpened
		if json.Unmarshal([]byte(taskEvents[index].Body), &opened) != nil || opened.HarnessName != harnessName {
			continue
		}
		return strings.TrimSpace(opened.SessionID)
	}
	return ""
}
