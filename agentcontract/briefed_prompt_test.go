package agentcontract

import "testing"

func TestAModelThatWorksForThePersonReadsThePromptTheHostBriefed(t *testing.T) {
	request := AgentTurnRequest{Prompt: "회의록 정리해줘", BriefedPrompt: "You are 김인턴.\nAnswer them in en.\n\n회의록 정리해줘"}

	if request.PromptAsBriefed() != request.BriefedPrompt {
		t.Fatalf("the briefing the host wrote around the message has to reach the model, got %q", request.PromptAsBriefed())
	}
	if request.Prompt != "회의록 정리해줘" {
		t.Fatalf("the prompt stays what the person wrote, got %q", request.Prompt)
	}
}

func TestATurnNobodyBriefedReadsThePersonsWordsAsTheyAre(t *testing.T) {
	request := AgentRequest{Prompt: "회의록 정리해줘", BriefedPrompt: "  "}

	if request.PromptAsBriefed() != "회의록 정리해줘" {
		t.Fatalf("a host that briefs nothing leaves the person's words as the prompt, got %q", request.PromptAsBriefed())
	}
}
