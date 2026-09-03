package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"
)

func CreateRecordingAccessToken(id uint, download, cursor bool) (string, int64, error) {
	if Config.Jwt.Key == "" {
		return "", 0, errors.New("JWT key is required for recording access")
	}
	expiresAt := time.Now().Add(5 * time.Minute).Unix()
	payload := strconv.FormatUint(uint64(id), 10) + ":" + strconv.FormatInt(expiresAt, 10) + ":" + strconv.FormatBool(download) + ":" + strconv.FormatBool(cursor)
	mac := hmac.New(sha256.New, []byte(Config.Jwt.Key))
	_, _ = mac.Write([]byte(payload))
	token := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return token, expiresAt, nil
}

func VerifyRecordingAccessToken(token string, id uint, download, cursor bool) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 || Config.Jwt.Key == "" {
		return false
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	payload := string(payloadBytes)
	fields := strings.Split(payload, ":")
	if len(fields) != 4 {
		return false
	}
	tokenId, err := strconv.ParseUint(fields[0], 10, 32)
	if err != nil || uint(tokenId) != id || fields[2] != strconv.FormatBool(download) || fields[3] != strconv.FormatBool(cursor) {
		return false
	}
	expiresAt, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil || time.Now().Unix() > expiresAt {
		return false
	}
	mac := hmac.New(sha256.New, []byte(Config.Jwt.Key))
	_, _ = mac.Write(payloadBytes)
	return hmac.Equal(signature, mac.Sum(nil))
}
