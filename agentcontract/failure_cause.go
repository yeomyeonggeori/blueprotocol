package agentcontract

import (
	"errors"
	"net/http"

	"github.com/yeomyeonggeori/blueprotocol/model"
	"github.com/yeomyeonggeori/blueprotocol/model/transientretry"
)

type FailureCause string

const (
	FailureCauseTransient     FailureCause = "transient"
	FailureCauseAdministrator FailureCause = "administrator"
)

var FailureCauses = []FailureCause{FailureCauseTransient, FailureCauseAdministrator}

type failureCauseCarrier interface {
	FailureCause() FailureCause
}

func FailureCauseOf(errorValue error) FailureCause {
	var carrier failureCauseCarrier
	if errors.As(errorValue, &carrier) {
		return carrier.FailureCause()
	}
	var endpointError model.EndpointError
	if errors.As(errorValue, &endpointError) {
		return failureCauseOfEndpoint(endpointError)
	}
	return ""
}

func ParseFailureCause(value string) FailureCause {
	for _, cause := range FailureCauses {
		if string(cause) == value {
			return cause
		}
	}
	return ""
}

func failureCauseOfEndpoint(endpointError model.EndpointError) FailureCause {
	if !endpointError.WasReached() || transientretry.IsTransientStatus(endpointError.StatusCode) {
		return FailureCauseTransient
	}
	switch endpointError.StatusCode {
	case http.StatusUnauthorized, http.StatusPaymentRequired, http.StatusForbidden:
		return FailureCauseAdministrator
	}
	return ""
}
