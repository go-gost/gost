package ssh

import (
	"io/ioutil"

	tls_util "github.com/go-gost/gost/pkg/common/util/tls"
	ssh_util "github.com/go-gost/gost/pkg/internal/util/ssh"
	mdata "github.com/go-gost/gost/pkg/metadata"
	"golang.org/x/crypto/ssh"
)

type metadata struct {
	signer         ssh.Signer
	authorizedKeys map[string]bool
}

func (h *forwardHandler) parseMetadata(md mdata.Metadata) (err error) {
	const (
		authorizedKeys = "authorizedKeys"
		privateKeyFile = "privateKeyFile"
		passphrase     = "passphrase"
	)

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
