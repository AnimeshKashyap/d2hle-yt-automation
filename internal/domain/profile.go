package domain

import "time"

type SessionStatus string

const (
	SessionUnconfigured    SessionStatus = "unconfigured"
	SessionActive          SessionStatus = "active"
	SessionExpiringSoon    SessionStatus = "expiring_soon"
	SessionNeedsReauth     SessionStatus = "needs_reauth"
	SessionNeedsUserAction SessionStatus = "needs_user_action"
	SessionLoginInProgress SessionStatus = "login_in_progress"
	SessionCheckFailed     SessionStatus = "check_failed"
)

type Profile struct {
	ID            string        `json:"profile_id"`
	DisplayName   string        `json:"display_name"`
	UserDataDir   string        `json:"user_data_dir"`
	SessionStatus SessionStatus `json:"session_status"`
	AccountHint   string        `json:"account_hint,omitempty"`
	LastCheckedAt   *time.Time `json:"last_checked_at,omitempty"`
	LastRefreshedAt *time.Time `json:"last_refreshed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	Tokens        []APIToken    `json:"tokens,omitempty"`
}

type APIToken struct {
	ID        string    `json:"token_id"`
	Label     string    `json:"label,omitempty"`
	Hash      string    `json:"hash"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateProfileRequest struct {
	ProfileID   string `json:"profile_id"`
	DisplayName string `json:"display_name"`
}

type IssueTokenRequest struct {
	Label string `json:"label,omitempty"`
}

type IssueTokenResponse struct {
	TokenID string `json:"token_id"`
	Token   string `json:"token"`
}
