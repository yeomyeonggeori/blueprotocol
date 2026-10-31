package agentcontract

import (
	"strings"
	"testing"
	"time"
)

func TestAnUnstatedClockIsNotInvented(t *testing.T) {
	description := BuildTemporalContextDescription(time.Time{}, "Asia/Seoul", "")

	if description != "" {
		t.Fatalf("an unknown clock says nothing: any sentence about dates is what sends the model to the shell's wrong one, got %q", description)
	}
}

func TestAnEnvironmentThatStatesItsClockIsTakenAtItsWord(t *testing.T) {
	description := BuildTemporalContextDescription(time.Date(2023, 5, 18, 12, 0, 0, 0, time.UTC), "Asia/Seoul", "")

	if !strings.Contains(description, "Now: 2023-05-18") {
		t.Fatalf("a host that knows its environment's clock said so, and that is the clock: %q", description)
	}
}

func TestARequesterAbroadIsToldTheirOwnClock(t *testing.T) {
	description := BuildTemporalContextDescription(time.Date(2026, 10, 30, 1, 0, 0, 0, time.UTC), "Asia/Seoul", "America/Los_Angeles")

	for _, expected := range []string{
		"Now: 2026-10-30 (Fri) 10:00 +09:00 Asia/Seoul",
		"writing from America/Los_Angeles, where it is 2026-10-29 (Thu) 18:00 -07:00",
	} {
		if !strings.Contains(description, expected) {
			t.Fatalf("a requester whose day is not the company's is told both clocks, lacking %q: %q", expected, description)
		}
	}
}

func TestARequesterOnTheCompanyClockAddsNothing(t *testing.T) {
	now := time.Date(2026, 10, 30, 1, 0, 0, 0, time.UTC)
	companyOnly := BuildTemporalContextDescription(now, "Asia/Seoul", "")

	for _, requesterTimeZone := range []string{"Asia/Seoul", "Asia/Tokyo", "Not/AZone"} {
		if description := BuildTemporalContextDescription(now, "Asia/Seoul", requesterTimeZone); description != companyOnly {
			t.Fatalf("a requester whose clock reads the company's, or names no zone, costs no prompt (%q): %q", requesterTimeZone, description)
		}
	}
}
