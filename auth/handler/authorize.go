package handler

import (
	"net/http"
	"time"

	"github.com/gopherd/doge/crypto/cryptoutil"
	"github.com/gopherd/doge/erron"
	"github.com/gopherd/doge/net/httputil"
	"github.com/gopherd/doge/net/netutil"
	"github.com/gopherd/jwt"

	"github.com/gopherd/gopherd/auth"
	"github.com/gopherd/gopherd/auth/api"
	"github.com/gopherd/gopherd/auth/provider"
)

func Authorize(service auth.Service, w http.ResponseWriter, r *http.Request) {
	const tag = "authorize"
	w.Header().Set("Access-Control-Allow-Origin", "*")
	lang := r.Header.Get("X-Lang")
	req := new(api.AuthorizeRequest)
	err := req.Parse(r)
	if err != nil {
		service.Logger().Info().
			String("api", tag).
			Error("error", err).
			Print("parse arguments error")
		httputil.JSONResponse(w, erron.Errno(api.BadArgument, err))
		return
	}
	if req.Channel <= 0 {
		httputil.JSONResponse(w, erron.Errnof(api.BadArgument, "invalid channel: %d", req.Channel))
		return
	}

	ip := netutil.IP(r)

	service.Logger().Debug().
		String("api", tag).
		String("type", req.Type).
		String("os", req.Os).
		Int("channel", req.Channel).
		Print("received an authorize request")

	resp := new(api.AuthorizeResponse)
	resp.Channel = req.Channel
	if req.Type == "" {
		httputil.JSONResponse(w, resp)
		return
	}
	var user *provider.UserInfo
	if req.Type == provider.Device {
		req.Device = req.Account
	} else {
		// lookup provider
		p, err := service.Provider(req.Type)
		if err != nil {
			service.Logger().Error().
				String("api", tag).
				String("provider", req.Type).
				Print("provider not found")
			httputil.JSONResponse(w, erron.AsErrno(err))
			return
		}
		// authorize for provider
		user, err := p.Authorize(req.Account, req.Secret)
		if err != nil {
			service.Logger().Warn().
				String("api", tag).
				String("provider", req.Type).
				Print("provider.authorize error")
			httputil.JSONResponse(w, erron.AsErrno(err))
			return
		}
		if req.Device == "" {
			if user.OpenId == "" {
				service.Logger().Warn().
					String("api", tag).
					String("provider", req.Type).
					Print("openId required")
				httputil.JSONResponse(w, erron.Errnof(api.BadAuthorization, "openId not found"))
				return
			}
			req.Device = joinDeviceByOpenId(req.Type, user.OpenId)
		}
		if user.Key == "" {
			req.Account = req.Device
		}
	}
	var key = req.Account
	if user != nil && user.Key != "" {
		key = user.Key
	}
	// load or create account
	account, isNew, err := service.AccountModule().LoadOrCreate(req.Type, key, req.Device)
	if err != nil {
		service.Logger().Error().
			String("api", tag).
			String("provider", req.Type).
			String("key", key).
			String("device", req.Device).
			Error("error", err).
			Print("load or create account error")
		httputil.JSONResponse(w, erron.AsErrno(err))
		return
	}
	if user != nil {
		if user.Name != "" {
			account.SetName(user.Name)
		}
		if user.Avatar != "" {
			account.SetAvatar(user.Avatar)
		}
		if user.Location != "" {
			account.SetLocation(user.Location)
		}
	} else {
		if country, province, city, err := service.GeoModule().QueryLocation(ip, lang); err == nil {
			if location := provider.Location(country, province, city); location != "" {
				account.SetLocation(location)
			}
		}
	}
	// authorized success
	claims, err := authorized(service, ip, req, account, isNew)
	if err != nil {
		httputil.JSONResponse(w, erron.Errnof(api.InternalServerError, "internal server error"))
		return
	}

	// sign access_token and refresh_token
	options := service.Config()
	claims.Issuer = options.JWT.Issuer
	claims.IssuedAt = time.Now().Unix()
	claims.ExpiresAt = claims.IssuedAt + options.AccessTokenTTL
	resp.AccessToken, err = service.Signer().Sign(claims)
	if err != nil {
		service.Logger().Error().
			String("api", tag).
			Error("error", err).
			Print("signed access token error")
		httputil.JSONResponse(w, erron.AsErrno(err))
		return
	}
	resp.AccessTokenExpiredAt = claims.ExpiresAt
	claims.ExpiresAt = claims.IssuedAt + options.RefreshTokenTTL
	claims.Payload = jwt.Payload{
		Salt:  cryptoutil.GenerateSalt(16),
		Scope: "*",
		ID:    claims.Payload.ID,
		IP:    ip,
	}
	resp.RefreshToken, err = service.Signer().Sign(claims)
	if err != nil {
		service.Logger().Error().
			String("api", tag).
			Error("error", err).
			Print("signed refresh token error")
		httputil.JSONResponse(w, erron.AsErrno(err))
		return
	}
	resp.RefreshTokenExpiredAt = claims.ExpiresAt
	httputil.JSONResponse(w, resp)
}

func joinDeviceByOpenId(provider, openId string) string {
	return provider + ":" + openId + "@" + cryptoutil.MD5(openId)
}

func authorized(service auth.Service, ip string, req *api.AuthorizeRequest, account auth.Account, isNew bool) (*jwt.Claims, error) {
	if banned, reason := account.GetBanned(); banned {
		service.Logger().Info().
			Int64("uid", account.GetID()).
			String("banned_reason", reason).
			Print("account banned")
		return nil, erron.Errnof(api.Banned, "banned")
	}

	var claims = new(jwt.Claims)
	claims.Payload.Salt = cryptoutil.GenerateSalt(16)
	claims.Payload.Scope = "*"
	claims.Payload.ID = account.GetID()
	claims.Payload.IP = ip
	claims.Payload.Values = map[string]interface{}{
		"providers": account.GetProviders(),
	}

	service.Logger().Info().
		Int64("uid", account.GetID()).
		String("ip", ip).
		Print("authorized successfully")

	now := time.Now()
	if isNew {
		account.SetRegister(now, ip)
	}
	account.SetLastLogin(now, ip)
	return claims, service.AccountModule().Store(req.Type, account)
}
