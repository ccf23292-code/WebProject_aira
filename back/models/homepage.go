package models

import "time"

// HomepageMessage stores suggestions posted on the public homepage board.
type HomepageMessage struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint64    `gorm:"index" json:"user_id"`
	UserName  string    `gorm:"-" json:"user_name,omitempty"`
	AvatarURL string    `gorm:"-" json:"avatar_url,omitempty"`
	Content   string    `gorm:"type:text" json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
