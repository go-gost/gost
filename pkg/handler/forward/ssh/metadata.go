package ssh

import (
	"io/ioutil"
	"strings"

	"github.com/go-gost/gost/pkg/auth"
	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	ssh_util "github.com/go-gost/gost/pkg/internal/util/ssh"
	mdata "github.com/go-gost/gost/pkg/metadata"
	"golang.org/x/crypto/ssh"
)

type metadata struct {
	authenticator  auth.Authenticator
	signer         ssh.Signer
	authorizedKeys map[string]bool
}

func (h *forwardHandler) parseMetadata(md mdata.Metadata) (err error) {
	const (
		users          = "users"
		authorizedKeys = "authorizedKeys"
		privateKeyFile = "privateKeyFile"
		passphrase     = "passphrase"
	)

	if auths := mdata.GetStrings(md, users); len(auths) > 0 {
		authenticator := auth.NewLocalAuthenticator(nil)
		for _, auth := range auths {
			ss := strings.SplitN(auth, ":", 2)
			if len(ss) == 1 {
				authenticator.Add(ss[0], "")
			} else {
				authenticator.Add(ss[0], ss[1])
			}
		}
		h.md.authenticator = authenticator
	}

	if key := mdata.GetString(md, privateKeyFile); key != "" {
		data, err := ioutil.ReadFile(key)
		if err != nil {
			return err
		}

		pp := mdata.GetString(md, passphrase)
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

	if name := mdata.GetString(md, authorizedKeys); name != "" {
		m, err := ssh_util.ParseAuthorizedKeysFile(name)
		if err != nil {
			return err
		}
		h.md.authorizedKeys = m
	}

	return
}
