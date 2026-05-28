package models

import "time"

// LLMExplanation 保存一次 LLM 生成的题目解析，按生成时间 append-only。
// 同一题目可有多条记录（用户多次"重新生成"会逐条追加），按 CreatedAt DESC 取最新即可。
type LLMExplanation struct {
	ID        uint64     `gorm:"primaryKey;autoIncrement"  json:"id"`
	ProblemID uint64     `gorm:"index"                     json:"problem_id"`
	UserID    PrimaryKey `gorm:"index"                     json:"user_id"`
	Content   string     `gorm:"type:text"                 json:"content"`
	Model     string     `gorm:"size:64"                   json:"model"`
	CreatedAt time.Time  `                                 json:"created_at"`
}
