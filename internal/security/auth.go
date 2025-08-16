// Package security provides security utilities and authentication for AgentFlow
package security

// Authenticator provides authentication interface
type Authenticator interface {
	Authenticate(token string) (*Principal, error)
}

// Principal represents an authenticated user or service
type Principal struct {
	ID       string
	TenantID string
	Roles    []string
}

// NewAuthenticator creates a new authenticator
func NewAuthenticator() Authenticator {
	// Authentication implementation will be added
	return &noopAuthenticator{}
}

type noopAuthenticator struct{}

func (a *noopAuthenticator) Authenticate(token string) (*Principal, error) {
	// Placeholder implementation
	return &Principal{
		ID:       "anonymous",
		TenantID: "default",
		Roles:    []string{"user"},
	}, nil
}
