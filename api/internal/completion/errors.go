package completion

import "errors"

type ProviderErrorKind string

const (
	ProviderAuthentication  ProviderErrorKind = "authentication"
	ProviderRateLimit       ProviderErrorKind = "rate_limit"
	ProviderTimeout         ProviderErrorKind = "timeout"
	ProviderUnavailable     ProviderErrorKind = "unavailable"
	ProviderInvalidResponse ProviderErrorKind = "invalid_response"
)

type ProviderError struct {
	Kind ProviderErrorKind
	Err  error
}

func (error *ProviderError) Error() string { return "completion provider " + string(error.Kind) }
func (error *ProviderError) Unwrap() error { return error.Err }

func NewProviderError(kind ProviderErrorKind, err error) error {
	return &ProviderError{Kind: kind, Err: err}
}

func AsProviderError(err error) (*ProviderError, bool) {
	var providerError *ProviderError
	if errors.As(err, &providerError) {
		return providerError, true
	}
	return nil, false
}
