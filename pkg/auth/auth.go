package auth

// Authenticator is an interface for user authentication.
type Authenticator interface {
	Authenticate(user, password string) bool
}

// authenticator is an Authenticator that authenticates client by key-value pairs.
type authenticator struct {
	kvs map[string]string
}

// NewAuthenticator creates an Authenticator that authenticates client by pre-defined user mapping.
func NewAuthenticator(kvs map[string]string) Authenticator {
	return &authenticator{
		kvs: kvs,
	}
}

// Authenticate checks the validity of the provided user-password pair.
func (au *authenticator) Authenticate(user, password string) bool {
	if au == nil || len(au.kvs) == 0 {
		return true
	}

	v, ok := au.kvs[user]
	return ok && (v == "" || password == v)
}
