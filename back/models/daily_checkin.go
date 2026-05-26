package models

import "time"

// UserCheckin 对应数据库 user_checkins 表，记录每个用户的签到聚合状态。
// 每位用户对应一行，UserID 作为主键。
type UserCheckin struct {
	UserID          PrimaryKey `gorm:"primaryKey"             json:"user_id"`
	LastCheckinDate string     `gorm:"size:10;index"          json:"last_checkin_date"` // YYYY-MM-DD
	ContinuousDays  int        `gorm:"default:0"              json:"continuous_days"`
	MaxContinuous   int        `gorm:"default:0"              json:"max_continuous"`
	TotalDays       int        `gorm:"default:0"              json:"total_days"`
	UpdatedAt       time.Time  `                              json:"updated_at"`
}

// CheckinStatus 是签到接口的统一响应结构。
type CheckinStatus struct {
	CheckedToday    bool   `json:"checked_today"`
	LastCheckinDate string `json:"last_checkin_date"`
	ContinuousDays  int    `json:"continuous_days"`
	MaxContinuous   int    `json:"max_continuous"`
	TotalDays       int    `json:"total_days"`
}
