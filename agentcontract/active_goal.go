package agentcontract

import (
	"encoding/json"
	"strings"

	"github.com/yeomyeonggeori/blueprotocol/toolcontract"
)

type ActiveGoalStatus string

const (
	ActiveGoalStatusActive           ActiveGoalStatus = "active"
	ActiveGoalStatusWaitingUserInput ActiveGoalStatus = "waiting_user_input"
	ActiveGoalStatusWaitingApproval  ActiveGoalStatus = "waiting_approval"
	ActiveGoalStatusCompleted        ActiveGoalStatus = "completed"
	ActiveGoalStatusBlocked          ActiveGoalStatus = "blocked"
)

const (
	ArtifactRequirementNone      = "none"
	ArtifactRequirementPreferred = "preferred"
	ArtifactRequirementRequired  = "required"
)

const (
	ExpectedResultTypeFile = "file"
	ExpectedResultTypeLink = "link"
)

type ActiveGoal struct {
	GoalID              string           `json:"goalID,omitempty"`
	TaskRunID           string           `json:"taskRunID,omitempty"`
	OriginalInstruction string           `json:"originalInstruction,omitempty"`
	CurrentObjective    string           `json:"currentObjective,omitempty"`
	KnownContext        []string         `json:"knownContext,omitempty"`
	MissingInformation  []string         `json:"missingInformation,omitempty"`
	RequiredNextTools   []string         `json:"requiredNextTools,omitempty"`
	SelectedToolNames   []string         `json:"selectedToolNames,omitempty"`
	SelectedSkillNames  []string         `json:"selectedSkillNames,omitempty"`
	OutcomeContract     OutcomeContract  `json:"outcomeContract,omitempty"`
	Status              ActiveGoalStatus `json:"status,omitempty"`
	RestoreError        string           `json:"-"`
}

type OutcomeContract struct {
	RequiredEvidenceTools      []string         `json:"requiredEvidenceTools,omitempty"`
	RequiredEvidenceAnyOf      [][]string       `json:"requiredEvidenceAnyOf,omitempty"`
	RequiredAttachmentSuffixes []string         `json:"requiredAttachmentSuffixes,omitempty"`
	RequiredEffects            []OutcomeEffect  `json:"requiredEffects,omitempty"`
	ExpectedResults            []ExpectedResult `json:"expectedResults,omitempty"`
	ArtifactRequirement        string           `json:"artifactRequirement,omitempty"`
	SelectedEvidenceHints      []string         `json:"selectedEvidenceHints,omitempty"`
	Source                     string           `json:"source,omitempty"`
}

type OutcomeEffect struct {
	ObjectType         string   `json:"objectType"`
	Effect             string   `json:"effect"`
	Description        string   `json:"description,omitempty"`
	SuggestedNextTools []string `json:"suggestedNextTools,omitempty"`
}

type ExpectedResult struct {
	ID              string   `json:"id,omitempty"`
	Type            string   `json:"type,omitempty"`
	Description     string   `json:"description,omitempty"`
	Required        bool     `json:"required"`
	AcceptanceHints []string `json:"acceptanceHints,omitempty"`
}

type PriorTaskContext struct {
	TaskRunID              string             `json:"taskRunID,omitempty"`
	Status                 string             `json:"status,omitempty"`
	Prompt                 string             `json:"prompt,omitempty"`
	Result                 string             `json:"result,omitempty"`
	FailureReason          string             `json:"failureReason,omitempty"`
	OutcomeContract        OutcomeContract    `json:"outcomeContract,omitempty"`
	RequestedOutputFormats []string           `json:"requestedOutputFormats,omitempty"`
	RecordedAttempts       []PriorTaskAttempt `json:"recordedAttempts,omitempty"`
	OmittedAttemptCount    int                `json:"omittedAttemptCount,omitempty"`
}

type PriorTaskAttempt struct {
	ObservationID    string                        `json:"observationID"`
	Tool             string                        `json:"tool"`
	ToolInput        json.RawMessage               `json:"toolInput,omitempty"`
	ToolInputOmitted bool                          `json:"toolInputOmitted,omitempty"`
	Failure          *toolcontract.ToolFailure     `json:"failure,omitempty"`
	Effects          []toolcontract.ResourceEffect `json:"effects,omitempty"`
}

func OutcomeContractHasRequirements(contract OutcomeContract) bool {
	artifactRequirement := strings.TrimSpace(contract.ArtifactRequirement)
	return len(contract.ExpectedResults) > 0 ||
		len(contract.RequiredEvidenceTools) > 0 ||
		len(contract.RequiredEvidenceAnyOf) > 0 ||
		len(contract.RequiredAttachmentSuffixes) > 0 ||
		len(contract.RequiredEffects) > 0 ||
		(artifactRequirement != "" && artifactRequirement != ArtifactRequirementNone)
}
