package wechat

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gopherd/doge/crypto/cryptoutil"
	"github.com/gopherd/log"

	"github.com/gopherd/gopherd/auth/provider"
)

const (
	name            = "wxgame"
	code2SessionURL = "https://api.weixin.qq.com/sns/jscode2session"
)

func init() {
	provider.Register(name, open)
}

func open(source string) (provider.Provider, error) {
	i := strings.IndexByte(source, ':')
	if i <= 0 || i >= len(source) {
		return nil, errors.New("invalid source for provider " + name)
	}
	return &wxgameClient{
		appId:     source[:i],
		appSecret: source[i+1:],
	}, nil
}

type wxgameClient struct {
	appId     string
	appSecret string
}

type Code2SessionResponse struct {
	OpenId     string `json:"openid"`
	UnionId    string `json:"unionid"`
	SessionKey string `json:"session_key"`
	Errcode    int    `json:"errcode"`
	Errmsg     string `json:"errmsg"`
}

func (c *wxgameClient) Authorize(code, userdata string) (*provider.UserInfo, error) {
	sess, err := c.code2Session(code)
	if err != nil {
		return nil, err
	}

	var (
		user struct {
			Raw       string `json:"raw"`
			Encrypted string `json:"encrypted"`
			Sig       string `json:"sig"`
			IV        string `json:"iv"`
		}
		info *userInfo
		res  = &provider.UserInfo{
			Key:    sess.UnionId,
			OpenId: sess.OpenId,
		}
	)
	if err := json.Unmarshal([]byte(userdata), &user); err == nil && user.Raw != "" && user.Sig != "" {
		info = c.decryptAndVerify(user.Raw, user.Sig, sess.SessionKey, user.IV, user.Encrypted)
		if info != nil {
			if res.Key == "" {
				res.Key = info.UnionId
			}
			res.Name = info.Nickname
			res.Avatar = info.AvatarUrl
			res.Location = provider.Location(info.Country, info.Province, info.City)
		}
	}
	return res, nil
}

func (c *wxgameClient) code2Session(code string) (*Code2SessionResponse, error) {
	url := fmt.Sprintf("%s?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code", code2SessionURL, c.appId, c.appSecret, code)
	respObj := new(Code2SessionResponse)
	resp, err := http.Get(url)
	if err != nil {
		return nil, provider.Error{
			Name:        name,
			Code:        provider.NetworkError,
			Description: err.Error(),
		}
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(respObj)
	if err != nil {
		return nil, provider.Error{
			Name:        name,
			Code:        provider.ResponseFormatError,
			Description: err.Error(),
		}
	}
	if respObj.Errcode != 0 {
		return nil, provider.Error{
			Name:        name,
			Code:        strconv.Itoa(respObj.Errcode),
			Description: respObj.Errmsg,
		}
	}
	return respObj, nil
}

type userInfo struct {
	Nickname  string `json:"nickName"`
	AvatarUrl string `json:"avatarUrl"`
	Gender    int    `json:"gender"`
	Country   string `json:"country"`
	Province  string `json:"province"`
	City      string `json:"city"`

	UnionId   string `json:"unionId"`
	Watermark struct {
		AppId     string `json:"appid"`
		Timestamp int64  `json:"timestamp"`
	} `json:"watermark"`
}

func (c *wxgameClient) decryptAndVerify(raw, sig, sessionKey, ivStr, encryptedStr string) *userInfo {
	sig2 := cryptoutil.Sha1(raw + sessionKey)
	if sig2 != sig {
		log.Warn().
			String("want", sig).
			String("got", sig2).
			Print("signature mismatched")
		return nil
	}
	encrypted, err := base64.StdEncoding.DecodeString(encryptedStr)
	if err != nil {
		log.Warn().
			Error("error", err).
			Print("invalid encrypted data")
		return nil
	}
	aesKey, err := base64.StdEncoding.DecodeString(sessionKey)
	if err != nil {
		log.Warn().
			String("key", sessionKey).
			Error("error", err).
			Print("invalid session key")
		return nil
	}
	iv, err := base64.StdEncoding.DecodeString(ivStr)
	if err != nil {
		log.Warn().
			String("iv", ivStr).
			Error("error", err).
			Print("invalid iv")
		return nil
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		log.Warn().
			Bytes("aesKey", aesKey).
			Error("error", err).
			Print("invalid aes key")
		return nil
	}
	blockSize := block.BlockSize()
	if len(iv) != blockSize {
		log.Warn().
			Int("blockSize", blockSize).
			Int("iv.len", len(iv)).
			Print("invalid IV length: must be equal to block size")
		return nil
	}
	if len(encrypted) == 0 || len(encrypted)%blockSize != 0 {
		log.Warn().
			Int("len", len(encrypted)).
			Print("invalid encrypted data length")
		return nil
	}
	decrypter := cipher.NewCBCDecrypter(block, iv)
	decrypter.CryptBlocks(encrypted, encrypted)
	// unpadding
	size := len(encrypted)
	unpadding := int(encrypted[size-1])
	// unmarshal
	info := new(userInfo)
	if err := json.Unmarshal(encrypted[:size-unpadding], info); err != nil {
		log.Warn().
			Error("error", err).
			Print("invalid json data")
		return nil
	}
	return info
}
