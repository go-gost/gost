package ssh

import (
	"io/ioutil"
	"strings"

	"github.com/go-gost/gost/pkg/auth"
	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	md "github.com/go-gost/gost/pkg/metadata"
	"golang.org/x/crypto/ssh"
)

type metadata struct {
	authenticator  auth.Authenticator
	signer         ssh.Signer
	authorizedKeys map[string]bool
}

func (h *forwardHandler) parseMetadata(md md.Metadata) (err error) {
	const (
		users          = "users"
		authorizedKeys = "authorizedKeys"
		privateKeyFile = "privateKeyFile"
		passphrase     = "passphrase"
	)

	if v, _ := md.Get(users).([]interface{}); len(v) > 0 {
		authenticator := auth.NewLocalAuthenticator(nil)
		for _, auth := range v {
			if s, _ := auth.(string); s != "" {
				ss := strings.SplitN(s, ":", 2)
				if len(ss) == 1 {
					authenticator.Add(ss[0], "")
				} else {
					authenticator.Add(ss[0], ss[1])
				}
			}
		}
		h.md.authenticator = authenticator
	}

	if key := md.GetString(privateKeyFile); key != "" {
		data, err := ioutil.ReadFile(key)
		if err != nil {
			return err
		}

		pp := md.GetString(passphrase)
		if pp == "" {
			h.md.signer, err = ssh.ParsePrivateKey(data)
		} else {
			h.md.signer, err = ssh.ParsePrivateKeyWithPassphrase(data, []byte(pp))
		}
		if err != nil {
			return err
		}
	}
	if h.md.signer == nil {
		signer, err := ssh.NewSignerFromKey(tls_util.DefaultConfig.Clone().Certificates[0].PrivateKey)
		if err != nil {
			return err
		}
		h.md.signer = signer
	}

	if name := md.GetString(authorizedKeys); name != "" {
		m, err := parseAuthorizedKeysFile(name)
		if err != nil {
			return err
		}
		h.md.authorizedKeys = m
	}

	return
}

// parseSSHAuthorizedKeysFile parses ssh authorized keys file.
func parseAuthorizedKeysFile(name string) (map[string]bool, error) {
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
