package agentcontract

type RequesterIdentity struct {
	RequesterPersonID       string
	RequesterEmail          string
	RequesterName           string
	RequesterPlatformUserID string
	RequesterCallingName    string
	RequesterHandle         string
	RequesterCircles        []string
}

func (identity RequesterIdentity) Clone() RequesterIdentity {
	identity.RequesterCircles = append([]string(nil), identity.RequesterCircles...)
	return identity
}
