package agentcontract

import "context"

type Routing struct {
	Decision *TurnDecision
	IsExact  bool
}

type routingContextKey struct{}

func WithRouting(ctx context.Context, routing Routing) context.Context {
	return context.WithValue(ctx, routingContextKey{}, routing)
}

func RoutingFrom(ctx context.Context) Routing {
	routing, _ := ctx.Value(routingContextKey{}).(Routing)
	return routing
}
