package acpfailure

import (
	acp "github.com/coder/acp-go-sdk"
	"github.com/yeomyeonggeori/blueprotocol/agentcontract"
)

const causeField = "cause"

func PromptError(errorValue error) error {
	cause := agentcontract.FailureCauseOf(errorValue)
	if cause == "" {
		return errorValue
	}
	return acp.NewInternalError(map[string]any{"error": errorValue.Error(), causeField: string(cause)})
}

func CauseOf(requestError *acp.RequestError) agentcontract.FailureCause {
	data, isObject := requestError.Data.(map[string]any)
	if !isObject {
		return ""
	}
	cause, _ := data[causeField].(string)
	return agentcontract.ParseFailureCause(cause)
}
