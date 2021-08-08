package line

import (
	"encoding/json"
	"net/http"

	"github.com/gopherd/gopherd/auth/provider"
)

const (
	name        = "line"
	userinfoURL = "https://api.line.me/v2/profile"
)

func init() {
	provider.Register(name, open)
}

func open(source string) (provider.Provider, error) {
	return line, nil
}

type lineClient struct{}

var line = lineClient{}

func (c lineClient) Authorize(accessToken, _ string) (*provider.UserInfo, error) {
	respObj := userInfoResponse{}
	if err := c.request(userinfoURL, accessToken, &respObj); err != nil {
		return nil, err
	}
	return &provider.UserInfo{
		Key:    respObj.UserId,
		Name:   respObj.DisplayName,
		Avatar: respObj.PictureURL,
	}, nil
}

type response interface {
	ErrorMessage() string
}

type userInfoResponse struct {
	Message     string `json:"message"`
	UserId      string `json:"userId"`
	DisplayName string `json:"displayName"`
	PictureURL  string `json:"pictureUrl"`
}

func (resp userInfoResponse) ErrorMessage() string {
	return resp.Message
}

func (c lineClient) request(url, accessToken string, respObj response) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return provider.Error{
			Name:        name,
			Code:        provider.NetworkError,
			Description: err.Error(),
		}
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return provider.Error{
			Name:        name,
			Code:        provider.NetworkError,
			Description: err.Error(),
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return provider.Error{
			Name:        name,
			Code:        resp.Status,
			Description: respObj.ErrorMessage(),
		}
	}
	err = json.NewDecoder(resp.Body).Decode(respObj)
	if err != nil {
		return provider.Error{
			Name:        name,
			Code:        provider.ResponseFormatError,
			Description: err.Error(),
		}
	}
	return nil
}

func (c lineClient) Close() error { return nil }
