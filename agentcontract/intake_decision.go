package agentcontract

import (
	"context"

	"github.com/yeomyeonggeori/blueprotocol/toolcontract"
)

type IntakeAttachmentFact struct {
	Kind        string `json:"kind"`
	MimeType    string `json:"mimeType,omitempty"`
	FileName    string `json:"fileName,omitempty"`
	SizeBytes   int64  `json:"sizeBytes,omitempty"`
	Description string `json:"description,omitempty"`
}

func AttachmentFactsFromParts(parts []AgentPart) []IntakeAttachmentFact {
	facts := []IntakeAttachmentFact{}
	for _, part := range parts {
		switch part.Type {
		case AgentPartTypeImage:
			if part.Image == nil {
				continue
			}
			facts = append(facts, IntakeAttachmentFact{Kind: "image", MimeType: part.Image.MimeType, FileName: part.Image.Filename})
		case AgentPartTypeFile:
			if part.File == nil {
				continue
			}
			facts = append(facts, IntakeAttachmentFact{Kind: "file", MimeType: part.File.ContentType, FileName: part.File.Filename, SizeBytes: part.File.SizeBytes})
		}
	}
	return facts
}

const (
	IntakeQuestionWork                    = "work"
	IntakeQuestionRelation                = "relation"
	IntakeQuestionClarify                 = "clarify"
	IntakeQuestionExpectedToolCount       = "expectedToolCount"
	IntakeQuestionSingleToolChoice        = "singleToolChoice"
	IntakeQuestionIsExternalSendRequested = "isExternalSendRequested"
	IntakeQuestionTaskShape               = "taskShape"
	IntakeQuestionDeliverableKind         = "deliverableKind"
	IntakeQuestionPriorTaskReference      = "priorTaskReference"
	IntakeQuestionResponseLanguage        = "responseLanguage"

	IntakeQuestionPrefixFormat = "format."
	IntakeQuestionPrefixTool   = "tool."

	IntakeChoiceOptionNone = "none_of_these"
)

type ToolSelectionNeed struct {
	Need              string
	ToolSet           *toolcontract.ToolSet
	CallableToolNames []string
	CountLimit        int
	CallObserver      LLMCallObserver
}

type ToolSelector interface {
	SelectToolNames(context.Context, ToolSelectionNeed) ([]SelectedTool, error)
}

type SelectedTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type EquippedTools struct {
	SelectedTools []SelectedTool `json:"selectedTools"`
}
