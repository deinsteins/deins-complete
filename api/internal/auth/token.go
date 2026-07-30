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

type Service struct {
	secret  []byte
	version int
}
type claims struct {
	ID      string `json:"sub"`
	Issued  int64  `json:"iat"`
	Version int    `json:"ver"`
}

func New(secret string, version int) *Service {
	return &Service{secret: []byte(secret), version: version}
}
func (s *Service) Issue(id string) (string, error) {
	p, e := json.Marshal(claims{ID: id, Issued: time.Now().Unix(), Version: s.version})
	if e != nil {
		return "", e
	}
	raw := base64.RawURLEncoding.EncodeToString(p)
	m := hmac.New(sha256.New, s.secret)
	m.Write([]byte(raw))
	return raw + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil)), nil
}
func (s *Service) Validate(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", errors.New("invalid")
	}
	m := hmac.New(sha256.New, s.secret)
	m.Write([]byte(parts[0]))
	sig, e := base64.RawURLEncoding.DecodeString(parts[1])
	if e != nil || !hmac.Equal(sig, m.Sum(nil)) {
		return "", errors.New("invalid")
	}
	p, e := base64.RawURLEncoding.DecodeString(parts[0])
	if e != nil {
		return "", e
	}
	var c claims
	if json.Unmarshal(p, &c) != nil || c.ID == "" || c.Version != s.version {
		return "", errors.New("invalid")
	}
	return c.ID, nil
}
