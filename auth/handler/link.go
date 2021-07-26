package handler

import (
	"net/http"
	"strings"

	"github.com/gopherd/doge/erron"
	"github.com/gopherd/doge/net/httputil"

	"github.com/gopherd/gopherd/auth"
	"github.com/gopherd/gopherd/auth/api"
)

func Link(service auth.Service, w http.ResponseWriter, r *http.Request) {
	const tag = "link"
	req := new(api.LinkRequest)
	err := req.Parse(r)
	if err != nil {
		service.Logger().Info().
			String("api", tag).
			Error("error", err).
			Print("parse arguments error")
		service.Response(w, r, erron.Errno(api.BadArgument, err))
		return
	}

	ip := httputil.IP(r)

	credentials := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(credentials, prefix) {
		service.Logger().Warn().
			String("api", tag).
			String("credentials", credentials).
			Print("unsupported Authorization header")
		service.Response(w, r, erron.Errno(api.Unauthorized, err))
		return
	}
	var accessToken = strings.TrimPrefix(credentials, prefix)
	options := service.Options()
	claims, err := service.Signer().Verify(options.JWT.Issuer, accessToken)
	if err != nil {
		service.Logger().Warn().
			String("api", tag).
			Error("error", err).
			Print("invalid access token")
		service.Response(w, r, erron.Errno(api.Unauthorized, err))
		return
	}

	provider, err := service.Provider(req.Type)
	if err != nil {
		service.Logger().Error().
			String("api", tag).
			String("provider", req.Type).
			Print("provider not found")
		service.Response(w, r, erron.AsErrno(err))
		return
	}
	account, err := provider.Link(ip, req, claims)
	if err != nil {
		service.Logger().Error().
			String("api", tag).
			String("provider", req.Type).
			Print("provider.link error")
		service.Response(w, r, erron.AsErrno(err))
		return
	}

	var resp = new(api.LinkResponse)
	resp.OpenId = account.OpenId
	resp.Userdata = claims.Payload.Userdata
	service.Response(w, r, resp)
}
