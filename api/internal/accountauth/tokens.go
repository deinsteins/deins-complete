// Package accountauth keeps user-account credentials separate from installation tokens.
package accountauth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type Service struct {
	secret    []byte
	accessTTL time.Duration
}

func (s *Service) TTL() time.Duration { return s.accessTTL }

type claims struct {
	UserID    string `json:"sub"`
	ExpiresAt int64  `json:"exp"`
	Type      string `json:"typ"`
}

func New(secret string, accessTTL time.Duration) *Service {
	return &Service{secret: []byte(secret), accessTTL: accessTTL}
}

func (service *Service) IssueAccessToken(userID string) (string, error) {
	payload, err := json.Marshal(claims{UserID: userID, ExpiresAt: time.Now().Add(service.accessTTL).Unix(), Type: "account"})
	if err != nil {
		return "", err
	}
	raw := base64.RawURLEncoding.EncodeToString(payload)
	signer := hmac.New(sha256.New, service.secret)
	_, _ = signer.Write([]byte(raw))
	return raw + "." + base64.RawURLEncoding.EncodeToString(signer.Sum(nil)), nil
}

func (service *Service) ValidateAccessToken(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", errors.New("invalid account token")
	}
	signer := hmac.New(sha256.New, service.secret)
	_, _ = signer.Write([]byte(parts[0]))
	signature, err := decodeCanonical(parts[1])
	if err != nil || !hmac.Equal(signature, signer.Sum(nil)) {
		return "", errors.New("invalid account token")
	}
	payload, err := decodeCanonical(parts[0])
	if err != nil {
		return "", errors.New("invalid account token")
	}
	var value claims
	if json.Unmarshal(payload, &value) != nil || value.UserID == "" || value.Type != "account" || time.Now().Unix() >= value.ExpiresAt {
		return "", errors.New("invalid account token")
	}
	return value.UserID, nil
}

func NewOpaqueToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func decodeCanonical(value string) ([]byte, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || base64.RawURLEncoding.EncodeToString(decoded) != value {
		return nil, errors.New("invalid encoding")
	}
	return decoded, nil
}
