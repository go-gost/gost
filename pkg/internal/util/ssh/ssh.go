package ssh

import (
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/go-gost/gost/pkg/auth"
	"golang.org/x/crypto/ssh"
)

const (
	GostSSHTunnelRequest = "gost-tunnel" // extended request type for ssh tunnel
)

var (
	ErrSessionDead = errors.New("session is dead")
)

// PasswordCallbackFunc is a callback function used by SSH server.
// It authenticates user using a password.
type PasswordCallbackFunc func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error)

func PasswordCallback(au auth.Authenticator) PasswordCallbackFunc {
	if au == nil {
		return nil
	}
	return func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
		if au.Authenticate(conn.User(), string(password)) {
			return nil, nil
		}
		return nil, fmt.Errorf("password rejected for %s", conn.User())
	}
}

// PublicKeyCallbackFunc is a callback function used by SSH server.
// It offers a public key for authentication.
type PublicKeyCallbackFunc func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error)

func PublicKeyCallback(keys map[string]bool) PublicKeyCallbackFunc {
	if len(keys) == 0 {
		return nil
	}

	return func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
		if keys[string(pubKey.Marshal())] {
			return &ssh.Permissions{
				// Record the public key used for authentication.
				Extensions: map[string]string{
					"pubkey-fp": ssh.FingerprintSHA256(pubKey),
				},
			}, nil
		}
		return nil, fmt.Errorf("unknown public key for %q", c.User())
	}
}

// ParseSSHAuthorizedKeysFile parses ssh authorized keys file.
func ParseAuthorizedKeysFile(name string) (map[string]bool, error) {
	authorizedKeysBytes, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}
	authorizedKeysMap := make(map[string]bool)
	for len(authorizedKeysBytes) > 0 {
		pubKey, _, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
		if err != nil {
			return nil, err
		}
		authorizedKeysMap[string(pubKey.Marshal())] = true
		authorizedKeysBytes = rest
	}

	return authorizedKeysMap, nil
}
