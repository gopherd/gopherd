package google

import (
	"context"

	"github.com/gopherd/log"
	"google.golang.org/api/idtoken"

	"github.com/gopherd/gopherd/auth/provider"
)

const name = "google"

func init() {
	provider.Register(name, open)
}

// source: <audience>
func open(source string) (provider.Provider, error) {
	return &googleClient{
		audience: source,
	}, nil
}

type googleClient struct {
	audience string
}

func (c *googleClient) Authorize(accessToken, _ string) (*provider.UserInfo, error) {
	payload, err := idtoken.Validate(context.TODO(), accessToken, c.audience)
	if err != nil {
		log.Warn().
			String("provider", name).
			Error("error", err).
			Print("verify access token error")
		return nil, err
	}
	return &provider.UserInfo{
		Key:    getStringFromClaims(payload.Claims, "sub"),
		Name:   getStringFromClaims(payload.Claims, "name"),
		Avatar: getStringFromClaims(payload.Claims, "picture"),
	}, nil
}

func (c *googleClient) Close() error { return nil }

func getStringFromClaims(claims map[string]any, key string) string {
	if v, ok := claims[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
