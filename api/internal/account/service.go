package account

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"deinscomplete/api/internal/accountauth"
	"deinscomplete/api/internal/usage"
)

// EmailSender is deliberately small so SMTP/SES/etc. remain deployment details.
type EmailSender interface {
	SendMagicCode(context.Context, string, string) error
}

type Service struct {
	repo                 *Repository
	tokens               *accountauth.Service
	mailer               EmailSender
	registration         string
	refreshTTL, magicTTL time.Duration
	monthly              usage.MonthlyTracker
}

func NewService(repo *Repository, tokens *accountauth.Service, mailer EmailSender, registration string, refreshTTL, magicTTL time.Duration, monthly usage.MonthlyTracker) *Service {
	return &Service{repo: repo, tokens: tokens, mailer: mailer, registration: registration, refreshTTL: refreshTTL, magicTTL: magicTTL, monthly: monthly}
}
func validEmail(email string) bool {
	email = NormalizeEmail(email)
	return len(email) <= 254 && strings.Count(email, "@") == 1 && !strings.HasPrefix(email, "@") && !strings.HasSuffix(email, "@")
}
func randomCode() (string, error) {
	b := make([]byte, 18)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// RequestMagicCode intentionally returns no account-existence information.
func (s *Service) RequestMagicCode(ctx context.Context, email, invite string) error {
	if !validEmail(email) {
		return nil
	}
	email = NormalizeEmail(email)
	_, err := s.repo.FindUserByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		if s.registration == "disabled" {
			return nil
		}
		params := CreateUserParams{Email: email}
		if s.registration == "invite" {
			if invite == "" {
				return nil
			}
			params.InviteCodeHash = accountauth.HashToken(invite)
		}
		if _, err = s.repo.CreateUser(ctx, params); err != nil {
			return nil
		}
	} else if err != nil {
		return err
	}
	if s.mailer == nil {
		return fmt.Errorf("account email sender is unavailable")
	}
	code, err := randomCode()
	if err != nil {
		return err
	}
	if _, err = s.repo.CreateMagicCode(ctx, email, accountauth.HashToken(code), time.Now().Add(s.magicTTL)); err != nil {
		return err
	}
	return s.mailer.SendMagicCode(ctx, email, code)
}

type TokenPair struct {
	AccessToken, RefreshToken string
	ExpiresIn                 int
}

func (s *Service) VerifyMagicCode(ctx context.Context, email, code string) (TokenPair, error) {
	if !validEmail(email) || code == "" {
		return TokenPair{}, ErrNotFound
	}
	bound, err := s.repo.ConsumeMagicCode(ctx, accountauth.HashToken(code))
	if err != nil || bound != NormalizeEmail(email) {
		return TokenPair{}, ErrNotFound
	}
	u, err := s.repo.FindUserByEmail(ctx, bound)
	if err != nil || u.Status != "active" {
		return TokenPair{}, ErrNotFound
	}
	return s.newSession(ctx, u.ID)
}
func (s *Service) newSession(ctx context.Context, userID string) (TokenPair, error) {
	raw, err := accountauth.NewOpaqueToken()
	if err != nil {
		return TokenPair{}, err
	}
	if _, err = s.repo.CreateSession(ctx, Session{UserID: userID, RefreshTokenHash: accountauth.HashToken(raw), ClientName: "vscode-extension", ExpiresAt: time.Now().Add(s.refreshTTL)}); err != nil {
		return TokenPair{}, err
	}
	access, err := s.tokens.IssueAccessToken(userID)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{access, raw, int(s.tokens.TTL().Seconds())}, nil
}
func (s *Service) Refresh(ctx context.Context, raw string) (TokenPair, error) {
	if raw == "" {
		return TokenPair{}, ErrNotFound
	}
	old := accountauth.HashToken(raw)
	if _, err := s.repo.FindActiveSessionByTokenHash(ctx, old); err != nil {
		return TokenPair{}, err
	}
	next, err := accountauth.NewOpaqueToken()
	if err != nil {
		return TokenPair{}, err
	}
	session, err := s.repo.RotateSession(ctx, old, Session{RefreshTokenHash: accountauth.HashToken(next), ClientName: "vscode-extension", ExpiresAt: time.Now().Add(s.refreshTTL)})
	if err != nil {
		return TokenPair{}, err
	}
	access, err := s.tokens.IssueAccessToken(session.UserID)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{access, next, int(s.tokens.TTL().Seconds())}, nil
}
func (s *Service) Logout(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	return s.repo.RevokeSessionByTokenHash(ctx, accountauth.HashToken(raw))
}
func (s *Service) User(ctx context.Context, id string) (User, error) {
	return s.repo.FindUserByID(ctx, id)
}
func (s *Service) Entitlements(ctx context.Context, id string) (Entitlements, error) {
	return s.repo.ResolveEntitlementsForUser(ctx, id)
}
func (s *Service) Installations(ctx context.Context, id string) ([]Installation, error) {
	return s.repo.ListInstallations(ctx, id)
}
func (s *Service) RevokeInstallation(ctx context.Context, user, id string) error {
	return s.repo.RevokeInstallation(ctx, id, user)
}
func (s *Service) LinkInstallation(ctx context.Context, userID, installationID string) (Installation, error) {
	i, err := s.repo.LinkInstallation(ctx, installationID, userID)
	if err != nil {
		return Installation{}, err
	}
	if s.monthly != nil {
		linked, e := s.repo.MarkInstallationUsageLinked(ctx, i.ID)
		if e != nil {
			return Installation{}, e
		}
		if linked {
			if e = s.monthly.MergeInstallationIntoUser(ctx, "installation:"+i.ID, "user:"+userID); e != nil {
				return Installation{}, e
			}
		}
	}
	return i, nil
}
