// Package signutil —— OKX REST 签名（okx-exec skill：sign=base64(HMAC-SHA256(
// secret, ts+method+path含query+body))；ts=ISO8601 毫秒串）。
package signutil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

// OkxTS ISO8601 毫秒时戳（签名与 OK-ACCESS-TIMESTAMP 头同源）。
func OkxTS(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

// OkxSign 签名串（path 须含 query 部分）。
func OkxSign(secret, ts, method, pathWithQuery, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%s%s%s%s", ts, method, pathWithQuery, body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// Credentials OKX key 三件套（自 app_secrets 密文列解出，仅存内存）。
type Credentials struct {
	Key, Secret, Passphrase string
}
