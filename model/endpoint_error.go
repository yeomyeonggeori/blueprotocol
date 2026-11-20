package model

import "context"

type EndpointError struct {
	StatusCode int
	Reason     error
}

func (endpointError EndpointError) Error() string {
	return endpointError.Reason.Error()
}

func (endpointError EndpointError) Unwrap() error {
	return endpointError.Reason
}

func (endpointError EndpointError) WasReached() bool {
	return endpointError.StatusCode != 0
}

func UnreachedEndpointError(ctx context.Context, reason error) error {
	if ctx.Err() != nil {
		return reason
	}
	return EndpointError{Reason: reason}
}
