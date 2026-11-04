package agentcontract

import "strings"

func (request AgentRequest) PromptAsBriefed() string {
	return promptAsBriefed(request.BriefedPrompt, request.Prompt)
}

func (request AgentTurnRequest) PromptAsBriefed() string {
	return promptAsBriefed(request.BriefedPrompt, request.Prompt)
}

func promptAsBriefed(briefedPrompt string, prompt string) string {
	if strings.TrimSpace(briefedPrompt) == "" {
		return prompt
	}
	return briefedPrompt
}
