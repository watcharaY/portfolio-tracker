package domain

import "time"

const (
	DefaultAdminUsername = "admin"
	DefaultAdminPassword = "admin"
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
