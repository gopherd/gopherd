package wechat

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gopherd/gopherd/auth/provider"
)

const (
	name           = "wechat"
	accessTokenURL = "https://api.weixin.qq.com/sns/oauth2/access_token"
	userinfoURL    = "https://api.weixin.qq.com/sns/userinfo"
)

const (
	codeOk = 0
)

func init() {
	provider.Register(name, open)
}

func open(source string) (provider.Provider, error) {
	i := strings.IndexByte(source, ':')
	if i <= 0 || i >= len(source) {
		return nil, errors.New("invalid source for provider " + name)
	}
	return &wechatClient{
		appId:     source[:i],
		appSecret: source[i+1:],
	}, nil
}

type wechatClient struct {
	appId     string
	appSecret string
}

func (c *wechatClient) Authorize(accessToken, openId string) (*provider.UserInfo, error) {
	url := userinfoURL + fmt.Sprintf("?access_token=%s&openid=%s", accessToken, openId)
	obj := userInfoResponse{}
	if err := c.request(url, &obj); err != nil {
		return nil, err
	}
	var gender provider.Gender
	switch obj.Sex {
	case 1:
		gender = provider.Male
	case 2:
		gender = provider.Female
	default:
		gender = provider.Unknown
	}
	return &provider.UserInfo{
		Key:      obj.UnionId,
		Name:     obj.Nickname,
		Avatar:   obj.HeadImgURL,
		Gender:   gender,
		Location: provider.Location(obj.Country, obj.Province, obj.City),
		OpenId:   obj.OpenId,
	}, nil
}

type response interface {
	ErrorCode() int
	ErrorMsg() string
}

type errorResponse struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

func (er errorResponse) ErrorCode() int   { return er.Errcode }
func (er errorResponse) ErrorMsg() string { return er.Errmsg }

type accessTokenResponse struct {
	errorResponse

	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	OpenId       string `json:"openid"`
	UnionId      string `json:"unionid"`
}

func (c *wechatClient) request(url string, respObj response) error {
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
	if respObj.ErrorCode() != codeOk {
		return provider.Error{
			Name:        name,
			Code:        strconv.Itoa(respObj.ErrorCode()),
			Description: respObj.ErrorMsg(),
		}
	}
	return nil
}

func (c *wechatClient) getAccessToken(code string) (*accessTokenResponse, error) {
	url := accessTokenURL + fmt.Sprintf("?appid=%s&secret=%s&code=%s&grant_type=authorization_code", c.appId, c.appSecret, code)
	obj := accessTokenResponse{}
	if err := c.request(url, &obj); err != nil {
		return nil, err
	}
	return &accessTokenResponse{
		AccessToken:  obj.AccessToken,
		ExpiresIn:    obj.ExpiresIn,
		RefreshToken: obj.RefreshToken,
		Scope:        obj.Scope,
		OpenId:       obj.OpenId,
		UnionId:      obj.UnionId,
	}, nil
}

type userInfoResponse struct {
	errorResponse

	Nickname   string   `json:"nickname"`
	Sex        int      `json:"sex"`
	Province   string   `json:"province"`
	City       string   `json:"city"`
	Country    string   `json:"country"`
	HeadImgURL string   `json:"headimgurl"`
	Privilege  []string `json:"privilege"`
	UnionId    string   `json:"unionid"`
	OpenId     string   `json:"openid"`
}

func (c *wechatClient) Close() error { return nil }
