package qq

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gopherd/gopherd/auth/provider"
)

const (
	name           = "qq"
	accessTokenURL = "https://graph.qq.com/oauth2.0/token"
	openIdURL      = "https://graph.qq.com/oauth2.0/me"
	userinfoURL    = "https://graph.qq.com/user/get_user_info"
)

const (
	codeOK = 0
)

func init() {
	provider.Register(name, open)
}

func open(_ string) (provider.Provider, error) {
	return qq, nil
}

type qqClient struct{}

var qq = qqClient{}

type response interface {
	ErrorCode() int
	ErrorMsg() string
}

type errorResponse struct {
	Errcode int    `json:"ret"`
	Errmsg  string `json:"msg"`
}

func (er errorResponse) ErrorCode() int   { return er.Errcode }
func (er errorResponse) ErrorMsg() string { return er.Errmsg }

type openIdResponse struct {
	errorResponse
	ClientId string `json:"client_id"`
	OpenId   string `json:"openid"`
}

type userInfoResponse struct {
	errorResponse
	Nickname       string `json:"nickname"`
	FigureURL      string `json:"figureurl"`
	FigureURL_1    string `json:"figureurl_1"`
	FigureURL_2    string `json:"figureurl_2"`
	FigureURL_qq_1 string `json:"figureurl_qq_1"`
	FigureURL_qq_2 string `json:"figureurl_qq_2"`
	Gender         string `json:"gender"`
}

func (c qqClient) request(url string, respObj response) error {
	resp, err := http.Get(url)
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

func (c qqClient) Authorize(accessToken, _ string) (*provider.UserInfo, error) {
	url := openIdURL + fmt.Sprintf("?access_token=%s", accessToken)
	openIdResp := openIdResponse{}
	if err := c.request(url, &openIdResp); err != nil {
		return nil, err
	}
	url = userinfoURL + fmt.Sprintf("?access_token=%s&oauth_consumer_key=%s&openid=%s", accessToken, openIdResp.ClientId, openIdResp.OpenId)
	respObj := userInfoResponse{}
	if err := c.request(url, &respObj); err != nil {
		return nil, err
	}
	var gender provider.Gender
	if respObj.Gender == "男" {
		gender = provider.Male
	} else {
		gender = provider.Female
	}
	var avatar string
	switch {
	case respObj.FigureURL_qq_2 != "":
		avatar = respObj.FigureURL_qq_2
	case respObj.FigureURL_qq_1 != "":
		avatar = respObj.FigureURL_qq_1
	case respObj.FigureURL_2 != "":
		avatar = respObj.FigureURL_2
	case respObj.FigureURL_1 != "":
		avatar = respObj.FigureURL_1
	case respObj.FigureURL != "":
		avatar = respObj.FigureURL
	}
	return &provider.UserInfo{
		Key:    openIdResp.OpenId,
		Name:   respObj.Nickname,
		Gender: gender,
		Avatar: avatar,
		OpenId: openIdResp.OpenId,
	}, nil
}
