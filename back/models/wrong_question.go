package models

import "time"

const (
	WrongStatusUnmastered = "unmastered"
	WrongStatusMastered   = "mastered"
	WrongStatusTrash      = "trash"
)

// WrongQuestion 记录用户错题。
type WrongQuestion struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      uint64    `gorm:"index;not null"            json:"user_id"`
	ProblemID   uint64    `gorm:"index;not null"            json:"problem_id"`
	CourseID    string    `gorm:"size:128;index"            json:"course_id"`
	PaperID     uint64    `gorm:"index"                     json:"paper_id"`
	Note        string    `gorm:"type:text"                 json:"note"`
	Status      string    `gorm:"size:32;index"             json:"status"`
	WrongCount  int       `gorm:"default:1"                 json:"wrong_count"`
	LastWrongAt time.Time `json:"last_wrong_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
