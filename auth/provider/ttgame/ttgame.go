package toutiao

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
	name            = "ttgame"
	code2SessionURL = "https://developer.toutiao.com/api/apps/jscode2session"
)

func init() {
	provider.Register(name, open)
}

func open(source string) (provider.Provider, error) {
	i := strings.IndexByte(source, ':')
	if i <= 0 || i >= len(source) {
		return nil, errors.New("invalid source for provider " + name)
	}
	return &ttgameClient{
		appId:     source[:i],
		appSecret: source[i+1:],
	}, nil
}

type ttgameClient struct {
	appId     string
	appSecret string
}

func (c *ttgameClient) Authorize(code, userdata string) (*provider.UserInfo, error) {
	var codeInfo struct {
		Code          string `json:"code"`
		AnonymousCode string `json:"anonymous_code"`
	}
	if err := json.Unmarshal([]byte(code), &codeInfo); err != nil {
		return nil, err
	}
	sess, err := c.code2Session(codeInfo.AnonymousCode, code)
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
			Key:    sess.OpenId,
			OpenId: sess.AnonymousOpenId,
		}
	)
	if codeInfo.Code != "" {
		if err := json.Unmarshal([]byte(userdata), &user); err != nil {
			return nil, err
		}
		info = c.decryptAndVerify(user.Raw, user.Sig, sess.SessionKey, user.IV, user.Encrypted)
		if info != nil {
			if res.OpenId != "" {
				res.OpenId = info.OpenId
			}
			res.Name = info.Nickname
			res.Avatar = info.AvatarUrl
			res.Location = provider.Location(info.Country, info.Province, info.City)
		}
	}
	return res, nil
}

type code2SessionResponse struct {
	Errcode         int    `json:"errcode"`
	Errmsg          string `json:"errmsg"`
	SessionKey      string `json:"session_key"`
	OpenId          string `json:"openid"`
	AnonymousOpenId string `json:"anonymous_openid"`
}

func (c *ttgameClient) code2Session(acode, code string) (*code2SessionResponse, error) {
	var url string
	if code != "" {
		url = fmt.Sprintf("%s?appid=%s&secret=%s&anonymous_code=%s&code=%s", code2SessionURL, c.appId, c.appSecret, acode, code)
	} else {
		url = fmt.Sprintf("%s?appid=%s&secret=%s&anonymous_code=%s", code2SessionURL, c.appId, c.appSecret, acode)
	}
	respObj := new(code2SessionResponse)
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

	OpenId    string `json:"openId"`
	Watermark struct {
		AppId     string `json:"appid"`
		Timestamp int64  `json:"timestamp"`
	} `json:"watermark"`
}

func (c *ttgameClient) decryptAndVerify(raw, sig, sessionKey, ivStr, encryptedStr string) *userInfo {
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
