package account

import "context"

// DevelopmentMailer is intentionally opt-in and only suitable for tests/local development.
// Production deployments must supply a real EmailSender implementation.
type DevelopmentMailer struct{ LastCode string }

func (m *DevelopmentMailer) SendMagicCode(_ context.Context, _ string, code string) error {
	m.LastCode = code
	return nil
}
