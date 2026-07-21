// Package auth provides authentication: password hashing, signed session
// tokens, HTTP handlers, and middleware that gates every protected route.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// claims is the payload embedded in a session token.
type claims struct {
	Sub   int64  `json:"sub"`
	Email string `json:"email"`
	Role  string `json:"role"`
	Exp   int64  `json:"exp"` // unix seconds
}

var (
	errBadToken     = errors.New("invalid token")
	errTokenExpired = errors.New("token expired")
)

// b64 is URL-safe, unpadded base64 (JWT-style).
var b64 = base64.RawURLEncoding

// sign produces a compact HMAC-SHA256 signed token: base64(payload).base64(sig).
func sign(secret []byte, c claims) (string, error) {
	payload, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	body := b64.EncodeToString(payload)
	return body + "." + b64.EncodeToString(mac(secret, body)), nil
}

// verify validates the signature and expiry, returning the claims.
func verify(secret []byte, token string) (claims, error) {
	var c claims
	body, sig, ok := strings.Cut(token, ".")
	if !ok {
		return c, errBadToken
	}
	want, err := b64.DecodeString(sig)
	if err != nil {
		return c, errBadToken
	}
	// Constant-time signature comparison.
	if !hmac.Equal(want, mac(secret, body)) {
		return c, errBadToken
	}
	payload, err := b64.DecodeString(body)
	if err != nil {
		return c, errBadToken
	}
	if err := json.Unmarshal(payload, &c); err != nil {
		return c, errBadToken
	}
	if time.Now().Unix() > c.Exp {
		return c, errTokenExpired
	}
	return c, nil
}

func mac(secret []byte, body string) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(body))
	return h.Sum(nil)
}
