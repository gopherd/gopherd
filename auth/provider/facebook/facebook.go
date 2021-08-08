package facebook

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gopherd/gopherd/auth/provider"
)

const (
	name        = "facebook"
	userinfoURL = "https://graph.facebook.com/me"
)

const (
	codeOK = 0
)

func init() {
	provider.Register(name, open)
}

func open(_ string) (provider.Provider, error) {
	return facebook, nil
}

type response interface {
	ErrorCode() int
	ErrorMsg() string
}

type errorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e errorResponse) ErrorCode() int   { return e.Code }
func (e errorResponse) ErrorMsg() string { return e.Message }

type userInfoResponse struct {
	Error   errorResponse `json:"error,omitempty"`
	Id      string        `json:"id"`
	Name    string        `json:"name"`
	Picture struct {
		Data struct {
			IsSilhouette bool   `json:"is_silhouette"`
			URL          string `json:"url"`
		} `json:"data"`
	} `json:"picture"`
}

func (resp userInfoResponse) ErrorCode() int {
	return resp.Error.Code
}

func (resp userInfoResponse) ErrorMsg() string {
	return resp.Error.Message
}

type facebookClient struct{}

var facebook facebookClient

func (c facebookClient) request(url string, values url.Values, respObj response) error {
	resp, err := http.PostForm(url, values)
	if err != nil {
		return provider.Error{
			Name:        name,
			Code:        provider.NetworkError,
			Description: err.Error(),
		}
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(respObj)
	if err != nil {
		return provider.Error{
			Name:        name,
			Code:        provider.ResponseFormatError,
			Description: err.Error(),
		}
	}
	if respObj.ErrorCode() != codeOK {
		return provider.Error{
			Name:        name,
			Code:        strconv.Itoa(respObj.ErrorCode()),
			Description: respObj.ErrorMsg(),
		}
	}
	return nil
}

func (c facebookClient) Authorize(accessToken, _ string) (*provider.UserInfo, error) {
	respObj := userInfoResponse{}
	if err := c.request(userinfoURL, url.Values{
		"fields":       {"id,name,picture"},
		"access_token": {accessToken},
	}, &respObj); err != nil {
		return nil, err
	}
	var avatar string
	if !respObj.Picture.Data.IsSilhouette {
		avatar = respObj.Picture.Data.URL
	}
	return &provider.UserInfo{
		Key:    respObj.Id,
		Name:   respObj.Name,
		Avatar: avatar,
	}, nil
}

func (c facebookClient) Close() error { return nil }
