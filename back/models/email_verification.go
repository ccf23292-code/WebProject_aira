package models

import "time"

// EmailVerification 记录邮箱验证码与发送节流信息。
type EmailVerification struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Email      string    `gorm:"size:128;uniqueIndex"     json:"email"`
	Code       string    `gorm:"size:16"                  json:"code"`
	ExpiresAt  time.Time `gorm:"index"                    json:"expires_at"`
	LastSentAt time.Time `json:"last_sent_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
