package embeddingprompt

import (
	"encoding/json"
	"os"
	"testing"
)

func TestApplyMatchesTheSharedFixture(t *testing.T) {
	document, errorValue := os.ReadFile("testdata/embeddinggemma_prompts.json")
	if errorValue != nil {
		t.Fatal(errorValue)
	}
	var cases []struct {
		Model     string `json:"model"`
		InputType string `json:"inputType"`
		Text      string `json:"text"`
		Expected  string `json:"expected"`
	}
	if errorValue := json.Unmarshal(document, &cases); errorValue != nil {
		t.Fatal(errorValue)
	}
	for _, testCase := range cases {
		if actual := Apply(testCase.Model, testCase.InputType, testCase.Text); actual != testCase.Expected {
			t.Errorf("%s %q: expected %q, got %q", testCase.Model, testCase.InputType, testCase.Expected, actual)
		}
	}
}

func TestApplyWithCarriesTheTitleAndTheTask(t *testing.T) {
	cases := []struct {
		inputType string
		text      string
		options   Options
		expected  string
	}{
		{"document", "payroll notes", Options{Title: "payroll"}, "title: payroll | text: payroll notes"},
		{"doc", "payroll notes", Options{}, "title: none | text: payroll notes"},
		{"query", "who runs payroll", Options{Task: "qa"}, "task: question answering | query: who runs payroll"},
		{"query", "who runs payroll", Options{Task: "custom task"}, "task: custom task | query: who runs payroll"},
		{"query", "  task: search result | query: x ", Options{Task: "qa"}, "task: search result | query: x"},
	}
	for _, testCase := range cases {
		if actual := ApplyWith("google/embeddinggemma-2", testCase.inputType, testCase.text, testCase.options); actual != testCase.expected {
			t.Errorf("%s %+v: expected %q, got %q", testCase.inputType, testCase.options, testCase.expected, actual)
		}
	}
}

func TestOnlyTheLastSegmentOfTheModelNameDecidesGemma(t *testing.T) {
	if actual := Apply("embeddinggemma/other-model", "query", "slides"); actual != "slides" {
		t.Errorf("expected the text untouched, got %q", actual)
	}
}
