package approvalcore

import "github.com/yeomyeonggeori/blueprotocol/toolcontract"

func Refusal(notice string) toolcontract.ToolResult {
	return toolcontract.ToolFailureResult(toolcontract.FailureUnknown, toolcontract.FailureCodes.PolicyBlocked, "approval", notice)
}

func Declined() toolcontract.ToolResult {
	return Refusal("The requester declined this call. Do not retry it; choose another way or stop.")
}

func DelegatedTurn() toolcontract.ToolResult {
	return Refusal("This call needs the requester's approval, and a delegated turn has no one to ask: only the turn that was asked for the work can hold a call for approval. Do not wait for an approval; take another route, or report this back as the part you could not do.")
}
