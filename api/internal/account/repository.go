// Package account contains PostgreSQL persistence for the account domain.
package account

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound               = errors.New("account record not found")
	ErrEmailAlreadyRegistered = errors.New("email already registered")
	ErrInviteInvalid          = errors.New("invite is invalid")
	ErrInstallationOwned      = errors.New("installation belongs to another user")
	ErrInstallationRevoked    = errors.New("installation is not active")
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

type User struct {
	ID, Email, DisplayName, Status string
	CreatedAt, UpdatedAt           time.Time
}

type CreateUserParams struct {
	Email, DisplayName, InviteCodeHash string
}

type Plan struct {
	ID, Code, Name, Status string
	MonthlyCompletions     int
	RepositoryContext      bool
	Streaming              bool
	PremiumRouting         bool
}

type Entitlements struct{ Plan }

type Installation struct {
	ID, InstallationKey, UserID, Status string
	CreatedAt, UpdatedAt, LastSeenAt    time.Time
	UsageLinkedAt                       *time.Time
}

type Session struct {
	ID, UserID, RefreshTokenHash, ClientName string
	CreatedAt, ExpiresAt, LastUsedAt         time.Time
	RevokedAt                                *time.Time
}

type Invite struct {
	ID, CodeHash, Email  string
	ExpiresAt, CreatedAt time.Time
	UsedAt               *time.Time
}

type MagicCode struct {
	ID, EmailNormalized, TokenHash string
	ExpiresAt, CreatedAt           time.Time
	ConsumedAt                     *time.Time
}

func NormalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func (r *Repository) CreateUser(ctx context.Context, p CreateUserParams) (User, error) {
	email := NormalizeEmail(p.Email)
	if email == "" {
		return User{}, fmt.Errorf("email is required")
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)
	if p.InviteCodeHash != "" {
		var inviteEmail *string
		err = tx.QueryRow(ctx, `SELECT email_normalized FROM invites WHERE code_hash=$1 AND used_at IS NULL AND expires_at > now() FOR UPDATE`, p.InviteCodeHash).Scan(&inviteEmail)
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrInviteInvalid
		}
		if err != nil {
			return User{}, err
		}
		if inviteEmail != nil && *inviteEmail != email {
			return User{}, ErrInviteInvalid
		}
	}
	user := User{ID: uuid.NewString(), Email: strings.TrimSpace(p.Email), DisplayName: strings.TrimSpace(p.DisplayName), Status: "active"}
	err = tx.QueryRow(ctx, `INSERT INTO users (id,email,email_normalized,display_name,status) VALUES ($1,$2,$3,$4,'active') RETURNING created_at,updated_at`, user.ID, user.Email, email, user.DisplayName).Scan(&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if isUnique(err) {
			return User{}, ErrEmailAlreadyRegistered
		}
		return User{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO user_entitlements (user_id,plan_id) SELECT $1,id FROM plans WHERE code='free' AND status='active'`, user.ID); err != nil {
		return User{}, err
	}
	if p.InviteCodeHash != "" {
		if tag, err := tx.Exec(ctx, `UPDATE invites SET used_at=now() WHERE code_hash=$1 AND used_at IS NULL`, p.InviteCodeHash); err != nil || tag.RowsAffected() != 1 {
			if err != nil {
				return User{}, err
			}
			return User{}, ErrInviteInvalid
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return user, nil
}

func (r *Repository) FindUserByEmail(ctx context.Context, email string) (User, error) {
	return scanUser(r.pool.QueryRow(ctx, `SELECT id,email,display_name,status,created_at,updated_at FROM users WHERE email_normalized=$1`, NormalizeEmail(email)))
}
func (r *Repository) FindUserByID(ctx context.Context, id string) (User, error) {
	return scanUser(r.pool.QueryRow(ctx, `SELECT id,email,display_name,status,created_at,updated_at FROM users WHERE id=$1`, id))
}
func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.DisplayName, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (r *Repository) CreateInvite(ctx context.Context, codeHash, email string, expiresAt time.Time) (Invite, error) {
	i := Invite{ID: uuid.NewString(), CodeHash: codeHash, ExpiresAt: expiresAt}
	if email != "" {
		i.Email = NormalizeEmail(email)
	}
	err := r.pool.QueryRow(ctx, `INSERT INTO invites (id,code_hash,email,email_normalized,expires_at) VALUES ($1,$2,NULLIF($3,''),NULLIF($3,''),$4) RETURNING created_at`, i.ID, i.CodeHash, i.Email, i.ExpiresAt).Scan(&i.CreatedAt)
	return i, err
}

func (r *Repository) CreateMagicCode(ctx context.Context, email, tokenHash string, expiresAt time.Time) (MagicCode, error) {
	m := MagicCode{ID: uuid.NewString(), EmailNormalized: NormalizeEmail(email), TokenHash: tokenHash, ExpiresAt: expiresAt}
	if m.EmailNormalized == "" || m.TokenHash == "" {
		return MagicCode{}, fmt.Errorf("magic code email and token hash are required")
	}
	err := r.pool.QueryRow(ctx, `INSERT INTO magic_codes (id,email_normalized,token_hash,expires_at) VALUES ($1,$2,$3,$4) RETURNING created_at`, m.ID, m.EmailNormalized, m.TokenHash, m.ExpiresAt).Scan(&m.CreatedAt)
	return m, err
}

// ConsumeMagicCode makes a hashed, unexpired magic code unusable and returns its email.
func (r *Repository) ConsumeMagicCode(ctx context.Context, tokenHash string) (string, error) {
	var email string
	err := r.pool.QueryRow(ctx, `UPDATE magic_codes SET consumed_at=now() WHERE token_hash=$1 AND consumed_at IS NULL AND expires_at > now() RETURNING email_normalized`, tokenHash).Scan(&email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return email, err
}

func (r *Repository) CreateSession(ctx context.Context, s Session) (Session, error) {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	err := r.pool.QueryRow(ctx, `INSERT INTO user_sessions (id,user_id,refresh_token_hash,client_name,expires_at) VALUES ($1,$2,$3,$4,$5) RETURNING created_at,last_used_at`, s.ID, s.UserID, s.RefreshTokenHash, s.ClientName, s.ExpiresAt).Scan(&s.CreatedAt, &s.LastUsedAt)
	return s, err
}
func (r *Repository) FindActiveSessionByTokenHash(ctx context.Context, hash string) (Session, error) {
	var s Session
	err := r.pool.QueryRow(ctx, `SELECT id,user_id,refresh_token_hash,client_name,created_at,expires_at,revoked_at,last_used_at FROM user_sessions WHERE refresh_token_hash=$1 AND revoked_at IS NULL AND expires_at > now()`, hash).Scan(&s.ID, &s.UserID, &s.RefreshTokenHash, &s.ClientName, &s.CreatedAt, &s.ExpiresAt, &s.RevokedAt, &s.LastUsedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	return s, err
}
func (r *Repository) RotateSession(ctx context.Context, oldHash string, replacement Session) (Session, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback(ctx)
	var userID string
	err = tx.QueryRow(ctx, `UPDATE user_sessions SET revoked_at=now(),last_used_at=now() WHERE refresh_token_hash=$1 AND revoked_at IS NULL AND expires_at > now() RETURNING user_id`, oldHash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, err
	}
	replacement.UserID = userID
	if replacement.ID == "" {
		replacement.ID = uuid.NewString()
	}
	err = tx.QueryRow(ctx, `INSERT INTO user_sessions (id,user_id,refresh_token_hash,client_name,expires_at) VALUES ($1,$2,$3,$4,$5) RETURNING created_at,last_used_at`, replacement.ID, replacement.UserID, replacement.RefreshTokenHash, replacement.ClientName, replacement.ExpiresAt).Scan(&replacement.CreatedAt, &replacement.LastUsedAt)
	if err != nil {
		return Session{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Session{}, err
	}
	return replacement, nil
}
func (r *Repository) RevokeSessionByTokenHash(ctx context.Context, hash string) error {
	_, err := r.pool.Exec(ctx, `UPDATE user_sessions SET revoked_at=now() WHERE refresh_token_hash=$1 AND revoked_at IS NULL`, hash)
	return err
}

func (r *Repository) CreateInstallation(ctx context.Context, key string) (Installation, error) {
	i := Installation{ID: uuid.NewString(), InstallationKey: key, Status: "active"}
	err := r.pool.QueryRow(ctx, `INSERT INTO installations (id,installation_key,status) VALUES ($1,$2,'active') RETURNING created_at,updated_at,last_seen_at`, i.ID, i.InstallationKey).Scan(&i.CreatedAt, &i.UpdatedAt, &i.LastSeenAt)
	return i, err
}

// EnsureInstallation returns the durable record for a legacy installation key.
func (r *Repository) EnsureInstallation(ctx context.Context, key string) (Installation, error) {
	i := Installation{ID: uuid.NewString(), InstallationKey: key, Status: "active"}
	err := r.pool.QueryRow(ctx, `INSERT INTO installations (id,installation_key,status) VALUES ($1,$2,'active') ON CONFLICT (installation_key) DO NOTHING RETURNING created_at,updated_at,last_seen_at`, i.ID, i.InstallationKey).Scan(&i.CreatedAt, &i.UpdatedAt, &i.LastSeenAt)
	if err == nil {
		return i, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Installation{}, err
	}
	return r.FindInstallationByKey(ctx, key)
}
func (r *Repository) FindInstallationByKey(ctx context.Context, key string) (Installation, error) {
	return scanInstallation(r.pool.QueryRow(ctx, `SELECT id,installation_key,COALESCE(user_id::text,''),status,created_at,updated_at,last_seen_at,usage_linked_at FROM installations WHERE installation_key=$1`, key))
}
func (r *Repository) FindInstallationByID(ctx context.Context, id string) (Installation, error) {
	return scanInstallation(r.pool.QueryRow(ctx, `SELECT id,installation_key,COALESCE(user_id::text,''),status,created_at,updated_at,last_seen_at,usage_linked_at FROM installations WHERE id=$1`, id))
}
func scanInstallation(row pgx.Row) (Installation, error) {
	var i Installation
	err := row.Scan(&i.ID, &i.InstallationKey, &i.UserID, &i.Status, &i.CreatedAt, &i.UpdatedAt, &i.LastSeenAt, &i.UsageLinkedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Installation{}, ErrNotFound
	}
	return i, err
}
func (r *Repository) LinkInstallation(ctx context.Context, installationID, userID string) (Installation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Installation{}, err
	}
	defer tx.Rollback(ctx)
	i, err := scanInstallation(tx.QueryRow(ctx, `SELECT id,installation_key,COALESCE(user_id::text,''),status,created_at,updated_at,last_seen_at,usage_linked_at FROM installations WHERE id=$1 FOR UPDATE`, installationID))
	if err != nil {
		return Installation{}, err
	}
	if i.Status != "active" {
		return Installation{}, ErrInstallationRevoked
	}
	if i.UserID != "" && i.UserID != userID {
		return Installation{}, ErrInstallationOwned
	}
	if _, err = tx.Exec(ctx, `UPDATE installations SET user_id=$2,updated_at=now(),last_seen_at=now() WHERE id=$1`, installationID, userID); err != nil {
		return Installation{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Installation{}, err
	}
	i.UserID = userID
	return i, nil
}
func (r *Repository) MarkInstallationUsageLinked(ctx context.Context, installationID string) (bool, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE installations SET usage_linked_at=now(),updated_at=now() WHERE id=$1 AND usage_linked_at IS NULL`, installationID)
	return tag.RowsAffected() == 1, err
}
func (r *Repository) RevokeInstallation(ctx context.Context, installationID, userID string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE installations SET status='revoked',updated_at=now() WHERE id=$1 AND user_id=$2 AND status='active'`, installationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
func (r *Repository) TouchInstallation(ctx context.Context, installationID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE installations SET last_seen_at=now(),updated_at=now() WHERE id=$1 AND status='active'`, installationID)
	return err
}
func (r *Repository) ListInstallations(ctx context.Context, userID string) ([]Installation, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,installation_key,COALESCE(user_id::text,''),status,created_at,updated_at,last_seen_at,usage_linked_at FROM installations WHERE user_id=$1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Installation
	for rows.Next() {
		i, err := scanInstallation(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, i)
	}
	return result, rows.Err()
}

func (r *Repository) ResolveEntitlementsForUser(ctx context.Context, userID string) (Entitlements, error) {
	return scanEntitlements(r.pool.QueryRow(ctx, `SELECT p.id,p.code,p.name,p.status,p.monthly_completion_limit,p.repository_context_enabled,p.streaming_enabled,p.premium_routing_enabled FROM user_entitlements e JOIN plans p ON p.id=e.plan_id WHERE e.user_id=$1 AND p.status='active'`, userID))
}
func (r *Repository) ResolveEntitlementsForInstallation(ctx context.Context, installationID string) (Entitlements, error) {
	return scanEntitlements(r.pool.QueryRow(ctx, `SELECT p.id,p.code,p.name,p.status,p.monthly_completion_limit,p.repository_context_enabled,p.streaming_enabled,p.premium_routing_enabled FROM installations i LEFT JOIN user_entitlements e ON e.user_id=i.user_id JOIN plans p ON p.id=COALESCE(e.plan_id,'00000000-0000-0000-0000-000000000001'::uuid) WHERE i.id=$1 AND i.status='active' AND p.status='active'`, installationID))
}
func scanEntitlements(row pgx.Row) (Entitlements, error) {
	var e Entitlements
	err := row.Scan(&e.ID, &e.Code, &e.Name, &e.Status, &e.MonthlyCompletions, &e.RepositoryContext, &e.Streaming, &e.PremiumRouting)
	if errors.Is(err, pgx.ErrNoRows) {
		return Entitlements{}, ErrNotFound
	}
	return e, err
}
func (r *Repository) SetUserPlan(ctx context.Context, userID, planCode string) error {
	tag, err := r.pool.Exec(ctx, `INSERT INTO user_entitlements(user_id,plan_id) SELECT $1,id FROM plans WHERE code=$2 AND status='active' ON CONFLICT (user_id) DO UPDATE SET plan_id=EXCLUDED.plan_id,updated_at=now()`, userID, planCode)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
