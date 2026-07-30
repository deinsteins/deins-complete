package completion

import "errors"

const (
	MaxPrefixCharacters    = 50000
	MaxSuffixCharacters    = 25000
	MaxLanguageLength      = 100
	MaxFilePathLength      = 4096
	MaxClientFieldLength   = 100
	MaxRepositoryFiles     = 8
	MaxRepositoryChars     = 30000
	MaxRepositoryFileChars = 10000
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
	if repository := request.RepositoryContext; repository != nil {
		if len(repository.Files) > MaxRepositoryFiles {
			return ErrInvalidRequest
		}
		total := 0
		for _, file := range repository.Files {
			total += len(file.Content)
			if file.Path == "" || len(file.Path) > MaxFilePathLength || len(file.Language) > MaxLanguageLength || len(file.Reason) > 100 || len(file.Content) > MaxRepositoryFileChars || isAbsolutePath(file.Path) {
				return ErrInvalidRequest
			}
		}
		if total > MaxRepositoryChars || len(repository.Symbols) > 100 {
			return ErrInvalidRequest
		}
		for _, symbol := range repository.Symbols {
			if symbol.Name == "" || len(symbol.Name) > 300 || len(symbol.Kind) > 100 || len(symbol.FilePath) > MaxFilePathLength || len(symbol.Signature) > 1000 || isAbsolutePath(symbol.FilePath) {
				return ErrInvalidRequest
			}
		}
	}
	return nil
}

func isAbsolutePath(path string) bool {
	return len(path) > 0 && (path[0] == '/' || path[0] == '\\' || (len(path) > 2 && path[1] == ':'))
}
