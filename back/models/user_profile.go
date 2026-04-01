package models

import "time"

// UserProfile 存储用户可编辑的个人信息。
type UserProfile struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint64    `gorm:"uniqueIndex"             json:"user_id"`
	Nickname  string    `gorm:"size:128"                json:"nickname"`
	AvatarURL string    `gorm:"size:512"                json:"avatar_url"`
	Level     int       `gorm:"default:1"               json:"level"`
	Username  string    `gorm:"-"                       json:"username"`
	Email     string    `gorm:"-"                       json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
