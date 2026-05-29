package models

import "time"

// PaperAttempt 状态常量。
const (
	AttemptStatusInProgress = "in_progress"
	AttemptStatusCompleted  = "completed"
	AttemptStatusAbandoned  = "abandoned"
)

// PaperAttempt 表示某用户对某张试卷的一次"严格练习"尝试。
// 业务规则：
//   - 同一用户对同一张试卷在新建 attempt 时，已有的 in_progress 会被自动置为 abandoned。
//   - 提交单题后聚合字段（Score / Correct / Submitted）会同步更新。
//   - 当 Submitted == Total 时，Status 自动从 in_progress 转为 completed，并写入 CompletedAt。
type PaperAttempt struct {
	ID          uint64     `gorm:"primaryKey;autoIncrement"            json:"id"`
	UserID      PrimaryKey `gorm:"index;not null"                      json:"user_id"`
	PaperID     uint64     `gorm:"index;not null"                      json:"paper_id"`
	CourseID    string     `gorm:"size:128;index"                      json:"course_id"`
	Status      string     `gorm:"size:16;index;default:'in_progress'" json:"status"`
	Score       float64    `gorm:"type:numeric(8,2);default:0"         json:"score"`
	MaxScore    float64    `gorm:"type:numeric(8,2);default:0"         json:"max_score"`
	Correct     int        `gorm:"default:0"                           json:"correct"`
	Submitted   int        `gorm:"default:0"                           json:"submitted"`
	Total       int        `gorm:"default:0"                           json:"total"`
	StartedAt   time.Time  `                                           json:"started_at"`
	UpdatedAt   time.Time  `                                           json:"updated_at"`
	CompletedAt *time.Time `                                           json:"completed_at,omitempty"`
}
