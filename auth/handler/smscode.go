package handler

import (
	"net/http"
	"time"

	"github.com/gopherd/doge/erron"
	"github.com/gopherd/doge/net/httputil"
	"github.com/gopherd/doge/net/netutil"
	"github.com/gopherd/gopherd/auth"
	"github.com/gopherd/gopherd/auth/api"
)

func SMSCode(service auth.Service, w http.ResponseWriter, r *http.Request) {
	const tag = "smscode"
	req := new(api.SmsCodeRequest)
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

	ttl, err := service.SMSComponent().GenerateCode(req.Channel, netutil.IP(r), req.Mobile)
	if err != nil {
		httputil.JSONResponse(w, erron.AsErrno(err))
	} else {
		httputil.JSONResponse(w, api.SmsCodeResponse{
			Seconds: int(ttl / time.Second),
		})
	}
}
