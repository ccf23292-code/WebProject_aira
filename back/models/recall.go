package models

import (
	"time"

	"gorm.io/datatypes"
)

// RecallQuestionOption 是回忆题目的选项。
type RecallQuestionOption struct {
	Option string `json:"option"`
	Text   string `json:"text"`
}

// RecallPaper 对应回忆卷主表。
type RecallPaper struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	CourseID  string    `gorm:"size:64;index"            json:"course_id"`
	Title     string    `gorm:"size:256"                 json:"title"`
	CreatedBy uint64    `gorm:"index"                    json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RecallQuestion 对应回忆题目主表。
type RecallQuestion struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	PaperID      uint64    `gorm:"index"                    json:"paper_id"`
	QuestionType string    `gorm:"size:32;index"            json:"question_type"`
	Sequence     int       `gorm:"index"                    json:"sequence"`
	Content      string    `gorm:"type:text"                json:"content"`
	Answer       string    `gorm:"type:text"                json:"answer"`
	OptionsJSON  datatypes.JSON `gorm:"type:jsonb"               json:"-"`
	SourceUserID uint64    `gorm:"index"                    json:"source_user_id"`
	SupportCount int       `gorm:"default:0"                json:"support_count"`
	LastEditorID uint64    `gorm:"index"                    json:"last_editor_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// RecallQuestionSupport 记录用户对题目的支持。
type RecallQuestionSupport struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	QuestionID uint64    `gorm:"index:idx_question_user,unique" json:"question_id"`
	UserID     uint64    `gorm:"index:idx_question_user,unique" json:"user_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// RecallQuestionComment 记录题目的评论。
type RecallQuestionComment struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	QuestionID uint64    `gorm:"index"                    json:"question_id"`
	UserID     uint64    `gorm:"index"                    json:"user_id"`
	Content    string    `gorm:"type:text"                json:"content"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
