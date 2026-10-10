package agentcontract

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/yeomyeonggeori/blueprotocol/model"
)

func TestWorkIsReadByWeighingNoneAgainstEverythingElse(t *testing.T) {
	testCases := []struct {
		name          string
		probabilities map[string]float64
		want          Work
	}{
		{"doable work split three ways outweighs a single none", map[string]float64{"none": 0.34, "easy": 0.31, "normal": 0.22, "hard": 0.05, "impossible": 0.08}, WorkEasy},
		{"none stands when it outweighs all the rest together", map[string]float64{"none": 0.55, "easy": 0.3, "normal": 0.15}, WorkNone},
		{"doable work split three ways outweighs a single impossible", map[string]float64{"none": 0.1, "impossible": 0.36, "easy": 0.24, "normal": 0.18, "hard": 0.12}, WorkEasy},
		{"impossible stands when it outweighs all doable work", map[string]float64{"none": 0.2, "impossible": 0.5, "easy": 0.3}, WorkImpossible},
		{"doable work split three ways outweighs a single unclear", map[string]float64{"none": 0.1, "unclear": 0.4, "easy": 0.3, "normal": 0.2}, WorkEasy},
		{"unclear stands when it outweighs all doable work", map[string]float64{"none": 0.1, "unclear": 0.55, "easy": 0.35}, WorkUnclear},
		{"the likeliest doable level is the one taken", map[string]float64{"none": 0.2, "easy": 0.2, "normal": 0.45, "hard": 0.15}, WorkNormal},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			answer := model.DecisionAnswer{Type: model.DecisionQuestionTypeChoice, Choice: "none", Probabilities: testCase.probabilities}
			if work := ReadWork(answer); work != testCase.want {
				t.Fatalf("expected %q, got %q", testCase.want, work)
			}
		})
	}
}

func TestWorkWithoutADistributionIsTheChoice(t *testing.T) {
	if work := ReadWork(model.DecisionAnswer{Choice: " hard "}); work != WorkHard {
		t.Fatalf("expected the bare choice, got %q", work)
	}
	if work := ReadWork(model.DecisionAnswer{Choice: "medium"}); work != "" {
		t.Fatalf("expected an unknown choice to read as nothing, got %q", work)
	}
}

func TestTheWorkQuestionOffersTheWorkVocabulary(t *testing.T) {
	document, errorValue := json.Marshal(WorkQuestion("About message m1. ", "인턴").Criteria)
	if errorValue != nil {
		t.Fatal(errorValue)
	}
	criteria := string(document)
	for _, name := range WorkNames {
		if !strings.Contains(criteria, `"`+name+`"`) {
			t.Fatalf("expected %q among the options, got %s", name, criteria)
		}
	}
}

func TestWorkKnownToExistIsReadAsItsLikeliestLevel(t *testing.T) {
	idle := model.DecisionAnswer{Choice: "none", Probabilities: map[string]float64{"none": 0.8, "easy": 0.05, "normal": 0.1, "impossible": 0.05}}
	if work := ReadDoableWork(idle); work != WorkNormal {
		t.Fatalf("expected the likeliest doable level when work is a given, got %q", work)
	}
	if work := ReadDoableWork(model.DecisionAnswer{Choice: "none"}); work != WorkEasy {
		t.Fatalf("expected easy when the bare choice is not doable, got %q", work)
	}
}
