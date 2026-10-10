package agentcontract

import "strings"

type Work string

const (
	WorkNone       Work = "none"
	WorkEasy       Work = "easy"
	WorkNormal     Work = "normal"
	WorkHard       Work = "hard"
	WorkImpossible Work = "impossible"
)

var WorkNames = []string{string(WorkNone), string(WorkEasy), string(WorkNormal), string(WorkHard), string(WorkImpossible)}

var DoableWorkNames = []string{string(WorkEasy), string(WorkNormal), string(WorkHard)}

func NormalizeWork(value string) Work {
	trimmedValue := strings.TrimSpace(value)
	if !isListedName(WorkNames, trimmedValue) {
		return ""
	}
	return Work(trimmedValue)
}

func (work Work) IsDoable() bool {
	return isListedName(DoableWorkNames, string(work))
}

func (work Work) TaskLevel() TaskLevel {
	switch work {
	case WorkEasy:
		return TaskLevelLow
	case WorkNormal:
		return TaskLevelMedium
	case WorkHard:
		return TaskLevelHigh
	default:
		return ""
	}
}
