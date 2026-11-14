package acpupdate

import (
	"encoding/json"
	"strings"
)

const subjectLimit = 48

var subjectFields = []string{"path", "filePath", "query", "command", "title", "name", "url"}

func ToolCallTitle(toolName string, input json.RawMessage) string {
	if subject := subjectOfInput(input); subject != "" {
		return toolName + "(" + subject + ")"
	}
	return toolName
}

func subjectOfInput(input json.RawMessage) string {
	decoded := map[string]any{}
	if json.Unmarshal(input, &decoded) != nil {
		return ""
	}
	for _, fieldName := range subjectFields {
		if subject := subjectText(decoded[fieldName]); subject != "" {
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
