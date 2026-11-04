package agentcontract

import (
	"strings"
	"time"
)

const (
	CompanyZoneFallbackUnset      = "unset"
	CompanyZoneFallbackUnloadable = "unloadable"
)

const clockFormat = "2006-01-02 (Mon) 15:04 -07:00"

func BuildTemporalContextDescription(environmentNow time.Time, companyTimeZone string, requesterTimeZone string) string {
	if environmentNow.IsZero() {
		return ""
	}
	location := CompanyLocation(companyTimeZone)
	localTime := environmentNow.In(location)
	lines := []string{
		"Now: " + localTime.Format(clockFormat) + " " + location.String(),
		buildCurrentWeekDescription(localTime),
		"Resolve relative dates (오늘, 내일, 다음 주, next Friday) from this.",
	}
	if requesterClock := buildRequesterClockDescription(localTime, requesterTimeZone); requesterClock != "" {
		lines = append(lines, requesterClock)
	}
	return strings.Join(lines, "\n")
}

func buildRequesterClockDescription(companyTime time.Time, requesterTimeZone string) string {
	named := strings.TrimSpace(requesterTimeZone)
	if named == "" {
		return ""
	}
	requesterLocation, errorValue := time.LoadLocation(named)
	if errorValue != nil {
		return ""
	}
	requesterTime := companyTime.In(requesterLocation)
	if requesterTime.Format("-07:00") == companyTime.Format("-07:00") {
		return ""
	}
	return "The requester is writing from " + requesterLocation.String() + ", where it is " + requesterTime.Format(clockFormat) +
		". A day or time they give is theirs; every time shown here and every tool's date and time are the company's, so convert theirs before calling a tool, and say both when the difference matters."
}

func buildCurrentWeekDescription(localTime time.Time) string {
	daysSinceMonday := (int(localTime.Weekday()) + 6) % 7
	monday := localTime.AddDate(0, 0, -daysSinceMonday)
	anchors := make([]string, 0, 7)
	for dayOffset := 0; dayOffset < 7; dayOffset++ {
		anchors = append(anchors, monday.AddDate(0, 0, dayOffset).Format("Mon 01-02"))
	}
	return "This week: " + strings.Join(anchors, ", ")
}

func CompanyLocation(timeZone string) *time.Location {
	location, _ := companyLocationWithFallbackReason(timeZone)
	return location
}

func CompanyZoneFallbackReason(timeZone string) string {
	_, fallbackReason := companyLocationWithFallbackReason(timeZone)
	return fallbackReason
}

func companyLocationWithFallbackReason(timeZone string) (*time.Location, string) {
	named := strings.TrimSpace(timeZone)
	if named == "" {
		return time.UTC, CompanyZoneFallbackUnset
	}
	location, errorValue := time.LoadLocation(named)
	if errorValue != nil {
		return time.UTC, CompanyZoneFallbackUnloadable
	}
	return location, ""
}
