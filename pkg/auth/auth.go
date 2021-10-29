package auth

// Authenticator is an interface for user authentication.
type Authenticator interface {
	Authenticate(user, password string) bool
}

// LocalAuthenticator is an Authenticator that authenticates client by local key-value pairs.
type LocalAuthenticator struct {
	kvs map[string]string
}

// NewLocalAuthenticator creates an Authenticator that authenticates client by local infos.
func NewLocalAuthenticator(kvs map[string]string) *LocalAuthenticator {
	if kvs == nil {
		kvs = make(map[string]string)
	}
	return &LocalAuthenticator{
		kvs: kvs,
	}
}

// Authenticate checks the validity of the provided user-password pair.
func (au *LocalAuthenticator) Authenticate(user, password string) bool {
	if au == nil {
		return true
	}

	if len(au.kvs) == 0 {
		return true
	}

	v, ok := au.kvs[user]
	return ok && (v == "" || password == v)
}

// Add adds a key-value pair to the Authenticator.
func (au *LocalAuthenticator) Add(k, v string) {
	au.kvs[k] = v
}
