package models

import "time"

// AnswerRecord 记录用户做题行为。
type AnswerRecord struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         uint64    `gorm:"index;not null"            json:"user_id"`
	CourseID       string    `gorm:"size:128;index"            json:"course_id"`
	PaperID        uint64    `gorm:"index"                     json:"paper_id"`
	ProblemID      uint64    `gorm:"index"                     json:"problem_id"`
	SelectedOption string    `gorm:"size:16"                   json:"selected_option"`
	IsCorrect      bool      `gorm:"index"                     json:"is_correct"`
	Mode           string    `gorm:"size:32;index"             json:"mode"`
	AnsweredAt     time.Time `gorm:"index"                     json:"answered_at"`
}
