package google

import (
	"strings"

	googleAuthIDTokenVerifier "github.com/futurenda/google-auth-id-token-verifier"
	"github.com/gopherd/log"

	"github.com/gopherd/gopherd/auth/provider"
)

const name = "google"

func init() {
	provider.Register(name, open)
}

// source: <clientId>[,otherClientIds...]
func open(source string) (provider.Provider, error) {
	return &googleClient{
		clientIds: strings.Split(source, ","),
	}, nil
}

type googleClient struct {
	clientIds []string
}

func (c *googleClient) Name() string { return name }

func (c *googleClient) Authorize(accessToken, _ string) (*provider.UserInfo, error) {
	v := googleAuthIDTokenVerifier.Verifier{}
	err := v.VerifyIDToken(accessToken, c.clientIds)
	if err != nil {
		log.Warn().
			String("provider", name).
			Error("error", err).
			Print("verify access token error")
		return nil, err
	}
	claimSet, err := googleAuthIDTokenVerifier.Decode(accessToken)
	if err != nil {
		log.Warn().
			String("provider", name).
			Error("error", err).
			Print("decode access token error")
		return nil, err
	}
	return &provider.UserInfo{
		Key:    claimSet.Sub,
		Name:   claimSet.Name,
		Avatar: claimSet.Picture,
	}, nil
}
