package embeddingprompt

import "strings"

const (
	InputTypeQuery    = "query"
	InputTypeDocument = "document"
)

func Apply(modelName string, inputType string, text string) string {
	if !isEmbeddingGemma(modelName) {
		return text
	}
	switch inputType {
	case InputTypeQuery:
		return "task: search result | query: " + text
	case InputTypeDocument:
		return "title: none | text: " + text
	}
	return text
}

func isEmbeddingGemma(modelName string) bool {
	lastSegment := modelName[strings.LastIndex(modelName, "/")+1:]
	return strings.HasPrefix(strings.ToLower(lastSegment), "embeddinggemma")
}
