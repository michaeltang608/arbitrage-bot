package util

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
)

func Sha256AndBase64(text, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(text))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
