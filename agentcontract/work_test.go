package agentcontract

import "testing"

func TestDoableWorkMapsOntoTheIntakeTaskLevels(t *testing.T) {
	levels := []string{}
	for _, name := range DoableWorkNames {
		levels = append(levels, string(Work(name).TaskLevel()))
	}
	for index, level := range IntakeTaskLevelNames {
		if levels[index] != level {
			t.Fatalf("doable work %v maps to %v, expected the intake levels %v", DoableWorkNames, levels, IntakeTaskLevelNames)
		}
	}
	for _, work := range []Work{WorkNone, WorkImpossible} {
		if work.IsDoable() || work.TaskLevel() != "" {
			t.Fatalf("%q is not doable and has no level", work)
		}
	}
}

func TestUnknownWorkNormalizesToNothing(t *testing.T) {
	if NormalizeWork(" easy ") != WorkEasy || NormalizeWork("medium") != "" {
		t.Fatal("only the listed work names normalize")
	}
}
