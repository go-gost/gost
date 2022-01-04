package auth

import (
	"net/url"

	"github.com/go-gost/gost/pkg/auth"
)

func AuthFromUsers(users ...*url.Userinfo) auth.Authenticator {
	kvs := make(map[string]string)
	for _, v := range users {
		if v == nil || v.Username() == "" {
			continue
		}
		kvs[v.Username()], _ = v.Password()
	}

	var authenticator auth.Authenticator
	if len(kvs) > 0 {
		authenticator = auth.NewMapAuthenticator(kvs)
	}

	return authenticator
}
