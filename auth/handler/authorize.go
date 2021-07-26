package handler

import (
	"net/http"
	"time"

	"github.com/gopherd/doge/crypto/cryptoutil"
	"github.com/gopherd/doge/erron"
	"github.com/gopherd/doge/net/httputil"
	"github.com/gopherd/jwt"

	"github.com/gopherd/gopherd/auth"
	"github.com/gopherd/gopherd/auth/api"
)

func Authorize(service auth.Service, w http.ResponseWriter, r *http.Request) {
	const tag = "authorize"
	w.Header().Set("Access-Control-Allow-Origin", "*")
	req := new(api.AuthorizeRequest)
	err := req.Parse(r)
	if err != nil {
		service.Logger().Info().
			String("api", tag).
			Error("error", err).
			Print("parse arguments error")
		service.Response(w, r, erron.Errno(api.BadArgument, err))
		return
	}
	if req.Channel <= 0 {
		service.Response(w, r, erron.Errnof(api.BadArgument, "invalid channel: %d", req.Channel))
		return
	}

	ip := httputil.IP(r)

	service.Logger().Debug().
		String("api", tag).
		String("type", req.Type).
		String("os", req.Os).
		Int("channel", req.Channel).
		Print("received an authorize request")

	resp := new(api.AuthorizeResponse)
	resp.Channel = req.Channel
	if req.Type == "" {
		service.Response(w, r, resp)
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
	account, isNew, err := provider.Authorize(ip, req)
	if err != nil {
		service.Logger().Error().
			String("api", tag).
			String("provider", req.Type).
			Print("provider.authorize error")
		service.Response(w, r, erron.AsErrno(err))
		return
	}
	claims, err := authorizedSuccess(service, account, ip, req, isNew)
	if err != nil {
		service.Response(w, r, erron.Errnof(api.InternalServerError, "internal server error"))
		return
	}

	options := service.Options()
	claims.Issuer = options.JWT.Issuer
	resp.OpenId = account.OpenId
	resp.AccessToken, err = service.Signer().Sign(claims)
	if err != nil {
		service.Logger().Error().
			String("api", tag).
			Error("error", err).
			Print("signed access token error")
		service.Response(w, r, erron.AsErrno(err))
		return
	}
	resp.AccessTokenExpiredAt = claims.ExpiresAt
	claims.ExpiresAt = claims.IssuedAt + int64(options.RefreshTokenTTL)
	claims.Payload = jwt.Payload{
		Salt:   cryptoutil.GenerateSalt(16),
		Scopes: []string{"*"},
		ID:     claims.Payload.ID,
	}
	resp.RefreshToken, err = service.Signer().Sign(claims)
	if err != nil {
		service.Logger().Error().
			String("api", tag).
			Error("error", err).
			Print("signed refresh token error")
		service.Response(w, r, erron.AsErrno(err))
		return
	}
	resp.RefreshTokenExpiredAt = claims.ExpiresAt
	service.Response(w, r, resp)
}

func authorizedSuccess(service auth.Service, account *auth.Account, ip string, req *api.AuthorizeRequest, isNew bool) (*jwt.Claims, error) {
	// 玩家被冻结账号
	if account.Banned {
		service.Logger().Info().
			Int64("uid", account.Uid).
			String("banned_reason", account.BannedReason).
			Print("account banned")
		return nil, erron.Errnof(api.Banned, "banned")
	}
	options := service.Options()

	var claims = new(jwt.Claims)
	claims.IssuedAt = time.Now().Unix()
	claims.ExpiresAt = claims.IssuedAt + int64(options.AccessTokenTTL)
	claims.Payload.Salt = cryptoutil.GenerateSalt(16)
	claims.Payload.ID = account.Uid
	claims.Payload.IP = ip

	// (TODO): set claims

	service.Logger().Info().
		Int64("uid", account.Uid).
		String("ip", ip).
		Print("authorized successfully")

	if !isNew {
		account.LastLoginAt = time.Now()
		account.LastLoginIP = ip
	}
	return claims, nil
}
