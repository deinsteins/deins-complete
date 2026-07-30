package completion

import "errors"

const (
	MaxPrefixCharacters  = 50000
	MaxSuffixCharacters  = 25000
	MaxLanguageLength    = 100
	MaxFilePathLength    = 4096
	MaxClientFieldLength = 100
)

var ErrInvalidRequest = errors.New("invalid completion request")

func Validate(request Request) error {
	context := request.Context
	if context.Language == "" || len(context.Language) > MaxLanguageLength || len(context.FilePath) > MaxFilePathLength ||
		len(context.Prefix) > MaxPrefixCharacters || len(context.Suffix) > MaxSuffixCharacters || context.CursorOffset < 0 {
		return ErrInvalidRequest
	}
	if request.Client != nil && (len(request.Client.Name) > MaxClientFieldLength || len(request.Client.Version) > MaxClientFieldLength) {
		return ErrInvalidRequest
	}
	return nil
}
