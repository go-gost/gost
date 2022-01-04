package auth

// Authenticator is an interface for user authentication.
type Authenticator interface {
	Authenticate(user, password string) bool
}

// LocalAuthenticator is an Authenticator that authenticates client by local key-value pairs.
type MapAuthenticator struct {
	kvs map[string]string
}

// NewMapAuthenticator creates an Authenticator that authenticates client by local infos.
func NewMapAuthenticator(kvs map[string]string) Authenticator {
	return &MapAuthenticator{
		kvs: kvs,
	}
}

// Authenticate checks the validity of the provided user-password pair.
func (au *MapAuthenticator) Authenticate(user, password string) bool {
	if au == nil || len(au.kvs) == 0 {
		return true
	}

	v, ok := au.kvs[user]
	return ok && (v == "" || password == v)
}
