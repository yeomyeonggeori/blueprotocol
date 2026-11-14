package toolcontract

import (
	"context"
	"strings"
)

type toolContextKey string

const taskRunIDContextKey toolContextKey = "taskRunID"
const observationIDContextKey toolContextKey = "observationID"
const responseLanguageContextKey toolContextKey = "responseLanguage"
const delegatedTurnContextKey toolContextKey = "delegatedTurn"
const holdIDContextKey toolContextKey = "holdID"

func WithTaskRunID(ctx context.Context, taskRunID string) context.Context {
	if taskRunID == "" {
		return ctx
	}
	return context.WithValue(ctx, taskRunIDContextKey, taskRunID)
}

func TaskRunIDFromContext(ctx context.Context) string {
	taskRunID, _ := ctx.Value(taskRunIDContextKey).(string)
	return taskRunID
}

func WithDelegatedTurn(ctx context.Context) context.Context {
	return context.WithValue(ctx, delegatedTurnContextKey, true)
}

func IsDelegatedTurn(ctx context.Context) bool {
	isDelegatedTurn, _ := ctx.Value(delegatedTurnContextKey).(bool)
	return isDelegatedTurn
}

func WithObservationID(ctx context.Context, observationID string) context.Context {
	if observationID == "" {
		return ctx
	}
	return context.WithValue(ctx, observationIDContextKey, observationID)
}

func ObservationIDFromContext(ctx context.Context) string {
	observationID, _ := ctx.Value(observationIDContextKey).(string)
	return observationID
}

func WithResponseLanguage(ctx context.Context, responseLanguage string) context.Context {
	normalizedLanguage := ResolveResponseLanguage(responseLanguage)
	if normalizedLanguage == "" {
		return ctx
	}
	return context.WithValue(ctx, responseLanguageContextKey, normalizedLanguage)
}

func ResponseLanguageFromContext(ctx context.Context) string {
	responseLanguage, _ := ctx.Value(responseLanguageContextKey).(string)
	return ResolveResponseLanguage(responseLanguage)
}

func WithHoldID(ctx context.Context, holdID string) context.Context {
	if strings.TrimSpace(holdID) == "" {
		return ctx
	}
	return context.WithValue(ctx, holdIDContextKey, holdID)
}

func HoldIDFromContext(ctx context.Context) string {
	holdID, _ := ctx.Value(holdIDContextKey).(string)
	return holdID
}
