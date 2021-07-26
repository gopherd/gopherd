//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
// NOTE: auto generated by midc, DON'T edit
//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
package api

import (
	"net/http"
	"net/url"

	"github.com/gopherd/doge/query"
)

var (
	_ = query.ParseURL
	_ = http.MethodGet
	_ url.Values
)

// Authorize
type AuthorizeRequest struct {
	Channel  int    `json:"channel"`
	Os       string `json:"os"`
	Version  int    `json:"version"`
	Type     string `json:"type"`
	Account  string `json:"account"`
	Secret   string `json:"secret"`
	Device   string `json:"device"`
	Userdata string `json:"userdata"`
	Source   string `json:"source"`
}

func (argv *AuthorizeRequest) form(r *http.Request) url.Values {
	const defaultMaxMemory = 32 << 20 // 32 MB
	if r.Form == nil {
		r.ParseMultipartForm(defaultMaxMemory)
	}
	return r.Form
}

func (argv *AuthorizeRequest) Parse(r *http.Request) error {
	var err error

	argv.Channel, err = query.RequiredInt(argv.form(r), "channel")
	if err != nil {
		return err
	}

	argv.Os = query.String(argv.form(r), "os", "")
	if err != nil {
		return err
	}

	argv.Version, err = query.Int(argv.form(r), "version", 0)
	if err != nil {
		return err
	}

	argv.Type, err = query.RequiredString(argv.form(r), "type")
	if err != nil {
		return err
	}

	argv.Account = query.String(argv.form(r), "account", "")
	if err != nil {
		return err
	}

	argv.Secret = query.String(argv.form(r), "secret", "")
	if err != nil {
		return err
	}

	argv.Device = query.String(argv.form(r), "device", "")
	if err != nil {
		return err
	}

	argv.Userdata = query.String(argv.form(r), "userdata", "")
	if err != nil {
		return err
	}

	argv.Source = query.String(argv.form(r), "source", "")
	if err != nil {
		return err
	}

	return err
}

type AuthorizeResponse struct {
	AccessToken           string            `json:"access_token"`
	AccessTokenExpiredAt  int64             `json:"access_token_expired_at"`
	RefreshToken          string            `json:"refresh_token"`
	RefreshTokenExpiredAt int64             `json:"refresh_token_expired_at"`
	Channel               int               `json:"channel"`
	OpenId                string            `json:"open_id"`
	Data                  map[string]string `json:"data"`
}

// Link account
type LinkRequest struct {
	Type     string `json:"type"`
	Account  string `json:"account"`
	Secret   string `json:"secret"`
	Userdata string `json:"userdata"`
}

func (argv *LinkRequest) form(r *http.Request) url.Values {
	const defaultMaxMemory = 32 << 20 // 32 MB
	if r.Form == nil {
		r.ParseMultipartForm(defaultMaxMemory)
	}
	return r.Form
}

func (argv *LinkRequest) Parse(r *http.Request) error {
	var err error

	argv.Type, err = query.RequiredString(argv.form(r), "type")
	if err != nil {
		return err
	}

	argv.Account = query.String(argv.form(r), "account", "")
	if err != nil {
		return err
	}

	argv.Secret = query.String(argv.form(r), "secret", "")
	if err != nil {
		return err
	}

	argv.Userdata = query.String(argv.form(r), "userdata", "")
	if err != nil {
		return err
	}

	return err
}

type LinkResponse struct {
	OpenId   string `json:"open_id"`
	Userdata string `json:"userdata"`
}

// SMS code
type SmsCodeRequest struct {
	Channel int    `json:"channel"`
	Mobile  string `json:"mobile"`
}

func (argv *SmsCodeRequest) form(r *http.Request) url.Values {
	const defaultMaxMemory = 32 << 20 // 32 MB
	if r.Form == nil {
		r.ParseMultipartForm(defaultMaxMemory)
	}
	return r.Form
}

func (argv *SmsCodeRequest) Parse(r *http.Request) error {
	var err error

	argv.Channel, err = query.RequiredInt(argv.form(r), "channel")
	if err != nil {
		return err
	}

	argv.Mobile, err = query.RequiredString(argv.form(r), "mobile")
	if err != nil {
		return err
	}

	return err
}

type SmsCodeResponse struct {
	Ttl int `json:"ttl"` // seconds

}
