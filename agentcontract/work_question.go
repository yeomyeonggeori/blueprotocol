package agentcontract

import (
	"strings"

	"github.com/yeomyeonggeori/blueprotocol/model"
)

const CallableToolsStateKey = "callableTools"

func WorkQuestion(about string, agentName string) model.DecisionQuestion {
	return model.ChoiceQuestion{
		Instructions: about + "Does it ask " + agentName + " to do work that takes tools and time, and how much? " + CallableToolsStateKey + " in the state names the tools " + agentName + " can call, so work any of them does is work " + agentName + " can be asked for. Work asked of somebody else is none.",
		OptionDescriptions: map[string]string{
			string(WorkNone):       "nothing for " + agentName + " to do. Words alone answer it, from what is visible, common knowledge or judgment, including a translation, an explanation or a draft written in the reply; or nobody asked " + agentName + " for anything. News that something " + agentName + " records or keeps is done, changed or wrong is not none when the record should follow it",
			string(WorkEasy):       "a short piece of work: a lookup, one record or one change",
			string(WorkNormal):     "work in several steps: research, several records, or a document or file to produce",
			string(WorkHard):       "long, wide or verification-heavy work",
			string(WorkImpossible): "work that cannot be done: physically impossible, nonsensical, or plainly improper on its face. Never for a permission concern or a tool " + agentName + " might lack, which the work itself finds out",
			string(WorkUnclear):    "work none of which can start until the sender answers one question: the requested goal, target or outcome is still ambiguous after the visible context, and only the sender can resolve it. A missing detail the work can look up, infer from context or fill with the plain default is not unclear, and neither is an approval",
		},
	}.Question()
}

func ReadWork(answer model.DecisionAnswer) Work {
	if len(answer.Probabilities) == 0 {
		return NormalizeWork(strings.TrimSpace(answer.Choice))
	}
	doableWeight := weightOf(answer, DoableWorkNames)
	impossibleWeight := answer.ChoiceProbability(string(WorkImpossible))
	unclearWeight := answer.ChoiceProbability(string(WorkUnclear))
	if doableWeight+impossibleWeight+unclearWeight <= answer.ChoiceProbability(string(WorkNone)) {
		return WorkNone
	}
	if impossibleWeight > doableWeight && impossibleWeight >= unclearWeight {
		return WorkImpossible
	}
	if unclearWeight > doableWeight {
		return WorkUnclear
	}
	return likeliestWork(answer)
}

func ReadDoableWork(answer model.DecisionAnswer) Work {
	if len(answer.Probabilities) == 0 {
		if work := NormalizeWork(strings.TrimSpace(answer.Choice)); work.IsDoable() {
			return work
		}
		return WorkEasy
	}
	return likeliestWork(answer)
}

func weightOf(answer model.DecisionAnswer, optionNames []string) float64 {
	weight := 0.0
	for _, optionName := range optionNames {
		weight += answer.ChoiceProbability(optionName)
	}
	return weight
}

func likeliestWork(answer model.DecisionAnswer) Work {
	likeliest := WorkEasy
	for _, optionName := range DoableWorkNames {
		if answer.ChoiceProbability(optionName) > answer.ChoiceProbability(string(likeliest)) {
			likeliest = Work(optionName)
		}
	}
	return likeliest
}
