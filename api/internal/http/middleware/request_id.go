package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"regexp"
)

type requestIDKey struct{}

var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestID := request.Header.Get("X-Request-ID")
		if !validRequestID.MatchString(requestID) {
			requestID = newRequestID()
		}
		writer.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(writer, request.WithContext(context.WithValue(request.Context(), requestIDKey{}, requestID)))
	})
}

func GetRequestID(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey{}).(string)
	return requestID
}

func newRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "unavailable"
	}
	return hex.EncodeToString(bytes)
}
