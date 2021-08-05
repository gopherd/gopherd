package fbinstant

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gopherd/log"
)

type FBInstantPlayer struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type signedFBInstantPlayer struct {
	Algorithm      string `json:"algorithm"`
	IssuedAt       int64  `json:"issued_at"`
	PlayerId       string `json:"player_id"`
	RequestPayload string `json:"request_payload"`
}

func ParsePayload(appSecret, request string) ([]byte, error) {
	parts := strings.Split(request, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid length of request parts")
	}

	// decode parts
	sig, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		log.Error().
			String("part0", parts[0]).
			Print("invalid base64 in part 0")
		return nil, err
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		log.Error().
			String("part1", parts[1]).
			Print("invalid base64 in part 1")
		return nil, err
	}

	// verify sig
	h := hmac.New(sha256.New, []byte(appSecret))
	io.WriteString(h, parts[1])
	sig2 := hex.EncodeToString(h.Sum(nil))
	sig1 := hex.EncodeToString(sig)
	if sig1 != sig2 {
		return nil, errors.New("signature mismatch")
	}

	return payload, nil
}

// Split the signature into two parts delimited by the '.' character.
// Decode the first part (the encoded signature) with base64url encoding.
// Decode the second part (the response payload) with base64url encoding,
// which should be a string representation of a JSON object that has the following fields:
//	** algorithm - always equals to HMAC-SHA256
//	** issued_at - a unix timestamp of when this response was issued.
//	** player_id - unique identifier of the player.
//	** request_payload - the requestPayload string you specified when calling FBInstant.player.getSignedPlayerInfoAsync.
// Hash the whole response payload string using HMAC SHA-256 and your app secret and confirm that it is equal to the encoded signature.
// You may also wish to validate the issued_at timestamp in the response payload to ensure the request was made recently.
func DecryptAndVerify(appSecret, signature string) (*FBInstantPlayer, error) {
	payload, err := ParsePayload(appSecret, signature)
	if err != nil {
		return nil, err
	}

	// decode payload as json
	sp := new(signedFBInstantPlayer)
	if err := json.Unmarshal(payload, sp); err != nil {
		return nil, err
	}
	if sp.Algorithm != "HMAC-SHA256" {
		return nil, errors.New("unsupported algorithm: " + sp.Algorithm)
	}

	// verify issued_at
	const kMaxIssuedAtDiff = 15 * 60 // 15 minutes
	var now = time.Now().Unix()
	diff := now - sp.IssuedAt
	if diff > kMaxIssuedAtDiff || diff < -kMaxIssuedAtDiff {
		return nil, fmt.Errorf("invalid issued_at: timeout=%d", diff)
	}

	// decode request_payload as json
	player := new(FBInstantPlayer)
	if err := json.Unmarshal([]byte(sp.RequestPayload), player); err != nil {
		return nil, err
	}
	player.Id = sp.PlayerId
	return player, nil
}
