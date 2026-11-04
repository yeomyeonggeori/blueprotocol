package acpupdate

import (
	"encoding/json"
	"strings"
)

const subjectLimit = 48

var subjectFields = []string{"path", "filePath", "query", "command", "title", "name", "url"}

func ToolCallTitle(toolName string, body string) string {
	if subject := subjectOfEvent(body); subject != "" {
		return toolName + "(" + subject + ")"
	}
	return toolName
}

func subjectOfEvent(body string) string {
	decoded := struct {
		Input map[string]any `json:"input"`
	}{}
	if json.Unmarshal([]byte(body), &decoded) != nil {
		return ""
	}
	for _, fieldName := range subjectFields {
		if subject := subjectText(decoded.Input[fieldName]); subject != "" {
			return subject
		}
	}
	return ""
}

func subjectText(value any) string {
	text, isText := value.(string)
	if !isText {
		return ""
	}
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	characters := []rune(text)
	if len(characters) <= subjectLimit {
		return text
	}
	return strings.TrimSpace(string(characters[:subjectLimit])) + "…"
}
