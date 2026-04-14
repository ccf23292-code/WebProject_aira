package models

import "time"

// AuthSession 持久化 access / refresh token，保证重启后登录态仍可校验。
type AuthSession struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID           uint64    `gorm:"index;not null"           json:"user_id"`
	AccessTokenHash  string    `gorm:"size:64;uniqueIndex"      json:"-"`
	RefreshTokenHash string    `gorm:"size:64;uniqueIndex"      json:"-"`
	AccessExpiresAt  time.Time `gorm:"index"                    json:"access_expires_at"`
	RefreshExpiresAt time.Time `gorm:"index"                   json:"refresh_expires_at"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
