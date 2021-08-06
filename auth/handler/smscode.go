package handler

import (
	"net/http"
	"time"

	"github.com/gopherd/doge/erron"
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
		service.Response(w, r, erron.Errno(api.BadArgument, err))
		return
	}
	if req.Channel <= 0 {
		service.Response(w, r, erron.Errnof(api.BadArgument, "invalid channel: %d", req.Channel))
		return
	}

	ttl, err := service.GenerateSMSCode(req.Channel, netutil.IP(r), req.Mobile)
	if err != nil {
		service.Response(w, r, erron.AsErrno(err))
	} else {
		service.Response(w, r, api.SmsCodeResponse{
			Seconds: int(ttl / time.Second),
		})
	}
}
