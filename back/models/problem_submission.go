package models

import "time"

// ProblemSubmission 是某次 PaperAttempt 下，对单道题目的提交记录。
//
// 核心约束：(attempt_id, problem_id) 复合唯一索引，由数据库层面保证
// "一次尝试里同一道题只能提交一次"。重复提交时 INSERT 会因唯一约束失败，
// 服务层据此返回 409 already_submitted。
type ProblemSubmission struct {
	ID          uint64     `gorm:"primaryKey;autoIncrement"                           json:"id"`
	AttemptID   uint64     `gorm:"uniqueIndex:idx_attempt_problem;index;not null"     json:"attempt_id"`
	ProblemID   uint64     `gorm:"uniqueIndex:idx_attempt_problem;index;not null"     json:"problem_id"`
	UserID      PrimaryKey `gorm:"index;not null"                                     json:"user_id"`
	UserAnswer  string     `gorm:"type:text"                                          json:"user_answer"`
	IsCorrect   bool       `gorm:"index"                                              json:"is_correct"`
	Score       float64    `gorm:"type:numeric(8,2);default:0"                        json:"score"`
	SubmittedAt time.Time  `gorm:"index"                                              json:"submitted_at"`
}
