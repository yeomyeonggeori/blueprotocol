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
