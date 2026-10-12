package domain

import (
	"strings"
	"time"
)

const (
	DefaultAdminUsername = "admin"
	DefaultAdminPassword = "admin"
	MinPasswordLength    = 8
	MaxPasswordBytes     = 72 // bcrypt rejects longer input
)

type User struct {
	ID                 int64
	Username           string
	PasswordHash       string
	MustChangePassword bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Session struct {
	TokenHash string
	UserID    int64
	IP        string
	UserAgent string
	CreatedAt time.Time
	ExpiresAt time.Time
}

func (s Session) Expired(now time.Time) bool { return !now.Before(s.ExpiresAt) }

func ValidateNewPassword(username, password string) error {
	switch {
	case len(password) < MinPasswordLength:
		return invalid("newPassword", "must be at least 8 characters")
	case len(password) > MaxPasswordBytes:
		return invalid("newPassword", "must be at most 72 bytes")
	case password == DefaultAdminPassword, strings.EqualFold(password, username):
		return invalid("newPassword", "must not be the default password or the username")
	}
	return nil
}
