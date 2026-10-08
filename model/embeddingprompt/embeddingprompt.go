package embeddingprompt

import "strings"

const (
	InputTypeQuery    = "query"
	InputTypeDocument = "document"
)

type Options struct {
	Task  string
	Title string
}

func Apply(modelName string, inputType string, text string) string {
	return ApplyWith(modelName, inputType, text, Options{})
}

func ApplyWith(modelName string, inputType string, text string, options Options) string {
	if !isEmbeddingGemma(modelName) {
		return text
	}
	trimmedText := strings.TrimSpace(text)
	switch normalizeInputType(inputType) {
	case InputTypeQuery:
		return promptQuery(trimmedText, options.Task)
	case InputTypeDocument:
		return promptDocument(trimmedText, options.Title)
	}
	return text
}

func promptQuery(text string, task string) string {
	if hasPrompt(text) {
		return text
	}
	return "task: " + taskDescription(task) + " | query: " + text
}

func promptDocument(text string, title string) string {
	if hasPrompt(text) {
		return text
	}
	if strings.TrimSpace(title) == "" {
		title = "none"
	}
	return "title: " + title + " | text: " + text
}

func hasPrompt(text string) bool {
	normalized := strings.ToLower(text)
	return strings.HasPrefix(normalized, "task: ") || strings.HasPrefix(normalized, "title: ")
}

func normalizeInputType(inputType string) string {
	switch strings.ToLower(strings.TrimSpace(inputType)) {
	case "document", "doc":
		return InputTypeDocument
	case "query":
		return InputTypeQuery
	}
	return ""
}

func taskDescription(task string) string {
	switch strings.ToLower(strings.TrimSpace(task)) {
	case "", "retrieval", "search":
		return "search result"
	case "question_answering", "question-answering", "question answering", "qa":
		return "question answering"
	case "fact_checking", "fact-checking", "fact checking", "fact_verification", "fact verification":
		return "fact checking"
	case "classification":
		return "classification"
	case "clustering":
		return "clustering"
	case "semantic_similarity", "semantic-similarity", "sentence_similarity", "sentence similarity", "sts":
		return "sentence similarity"
	case "code_retrieval", "code-retrieval", "code retrieval":
		return "code retrieval"
	}
	return strings.TrimSpace(task)
}

func isEmbeddingGemma(modelName string) bool {
	lastSegment := modelName[strings.LastIndex(modelName, "/")+1:]
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(lastSegment)), "embeddinggemma")
}
