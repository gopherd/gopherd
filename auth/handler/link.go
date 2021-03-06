package handler

import (
	"net/http"
	"strings"

	"github.com/gopherd/doge/erron"
	"github.com/gopherd/doge/net/httputil"
	"github.com/gopherd/doge/net/netutil"

	"github.com/gopherd/gopherd/auth"
	"github.com/gopherd/gopherd/auth/api"
	"github.com/gopherd/gopherd/auth/provider"
)

func Link(service auth.Service, w http.ResponseWriter, r *http.Request) {
	const tag = "link"
	req := new(api.LinkRequest)
	lang := r.Header.Get("X-Lang")
	err := req.Parse(r)
	if err != nil {
		service.Logger().Info().
			String("api", tag).
			Error("error", err).
			Print("parse arguments error")
		httputil.JSONResponse(w, erron.Errno(api.BadArgument, err))
		return
	}

	var (
		options     = service.Config()
		accessToken = req.Token
	)
	if accessToken == "" {
		credentials := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(credentials, prefix) {
			service.Logger().Warn().
				String("api", tag).
				String("credentials", credentials).
				Print("unsupported Authorization header")
			httputil.JSONResponse(w, erron.Errno(api.Unauthorized, err))
			return
		}
		accessToken = strings.TrimPrefix(credentials, prefix)
	}

	// get account by access token
	claims, err := service.Signer().Verify(options.JWT.Issuer, accessToken)
	if err != nil {
		service.Logger().Warn().
			String("api", tag).
			Error("error", err).
			Print("invalid access token")
		httputil.JSONResponse(w, erron.Errno(api.Unauthorized, err))
		return
	}
	account, err := service.AccountModule().Load(auth.ByID(claims.Payload.ID))
	if err != nil {
		service.Logger().Warn().
			String("api", tag).
			Error("error", err).
			Print("get account error")
		httputil.JSONResponse(w, erron.AsErrno(err))
		return
	}
	if account == nil {
		service.Logger().Info().
			String("api", tag).
			Int64("uid", claims.Payload.ID).
			Print("account not found by access token")
		httputil.JSONResponse(w, erron.AsErrno(err))
		return
	}

	// provider authorize
	p, err := service.Provider(req.Type)
	if err != nil {
		service.Logger().Error().
			String("api", tag).
			String("provider", req.Type).
			Print("provider not found")
		httputil.JSONResponse(w, erron.AsErrno(err))
		return
	}
	user, err := p.Authorize(req.Account, req.Secret)
	if err != nil {
		service.Logger().Error().
			String("api", tag).
			String("provider", req.Type).
			Print("provider.authorize error")
		httputil.JSONResponse(w, erron.AsErrno(err))
		return
	}
	if user.Key == "" {
		service.Logger().Error().
			String("api", tag).
			String("provider", req.Type).
			Print("key not present")
		httputil.JSONResponse(w, erron.Errno(api.Unauthorized, err))
		return
	}

	// check account
	if found, err := service.AccountModule().Contains(auth.ByProvider(req.Type, user.Key)); err != nil {
		service.Logger().Error().
			String("api", tag).
			String("provider", req.Type).
			String("key", user.Key).
			Error("error", err).
			Print("check account error")
		httputil.JSONResponse(w, erron.AsErrno(err))
		return
	} else if found {
		service.Logger().Error().
			String("api", tag).
			String("provider", req.Type).
			String("key", user.Key).
			Print("account already exist")
		httputil.JSONResponse(w, erron.Errnof(api.AccountFound, "account found"))
		return
	}

	// update account
	if user.Name != "" {
		account.SetName(user.Name)
	}
	if user.Avatar != "" {
		account.SetAvatar(user.Avatar)
	}
	if user.Location != "" {
		account.SetLocation(user.Location)
	} else if country, province, city, err := service.GeoModule().QueryLocation(netutil.IP(r), lang); err == nil {
		if location := provider.Location(country, province, city); location != "" {
			account.SetLocation(location)
		}
	}
	account.SetProvider(req.Type, user.Key)
	if err := service.AccountModule().Store(req.Type, account); err != nil {
		service.Logger().Error().
			String("api", tag).
			String("provider", req.Type).
			String("key", user.Key).
			Int64("uid", account.GetID()).
			Error("error", err).
			Print("store account error")
		httputil.JSONResponse(w, erron.AsErrno(err))
		return
	}

	var resp = new(api.LinkResponse)
	resp.OpenId = user.OpenId
	httputil.JSONResponse(w, resp)
}
