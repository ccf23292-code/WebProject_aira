package models

import (
	"time"

	"gorm.io/datatypes"
)

// IngestJob 表示一次"用户上传文件 → LLM 清洗 → 管理员审核 → 入库"流程中的任务记录。
//
// 状态机：
//
//	pending          ── worker 拉起 ──▶ processing
//	processing       ── 清洗成功 ──▶ awaiting_review
//	processing       ── 清洗失败 ──▶ failed
//	awaiting_review  ── admin 发布 ──▶ published
//	awaiting_review  ── admin 拒绝 ──▶ rejected
//
// 仅当 status=awaiting_review 时 ParsedJSON 可被 admin 编辑。
type IngestJob struct {
	ID     uint64     `gorm:"primaryKey;autoIncrement"   json:"id"`
	UserID PrimaryKey `gorm:"index;not null"              json:"user_id"`

	// Kind 标识本次上传是"题目"还是"题解"。
	Kind string `gorm:"size:16;index"               json:"kind"` // "question" / "explanation"

	// CourseID 必填于已有课程。NewCourseName 仅当用户选择"新增课程"时填，由 admin 审核时决定是否落 courses 表。
	CourseID      string `gorm:"size:64;index"          json:"course_id"`
	NewCourseName string `gorm:"size:128"               json:"new_course_name"`

	// PaperName 是兼容旧前端的"试卷名"自由字段，新前端走结构化三段（Year/Semester/ExamType）。
	// 题解流程则使用 TargetPaperID 指向已有试卷。
	PaperName     string  `gorm:"size:256"               json:"paper_name"`
	TargetPaperID *uint64 `gorm:"index"                  json:"target_paper_id"`

	// 结构化命名（题目流程优先用这三个；为空时回落到 PaperName）。
	// admin 审核时也能改这三段，决定与哪份既有试卷合并。
	Year     int    `                              json:"year,omitempty"`
	Semester string `gorm:"size:16"                json:"semester,omitempty"`
	ExamType string `gorm:"size:32"                json:"exam_type,omitempty"`

	// 原始文件信息。
	Filename    string `gorm:"size:256"                  json:"filename"`
	StoragePath string `gorm:"size:512"                  json:"storage_path"`
	Mime        string `gorm:"size:64"                   json:"mime"`
	Size        int64  `                                 json:"size"`

	Status       string `gorm:"size:32;index"             json:"status"`
	ErrorMessage string `gorm:"type:text"                 json:"error_message"`

	// RawText 预处理后的中间文本（Markdown）。
	// ParsedJSON LLM 结构化后的结果（jsonb），格式取决于 Kind。
	// DedupWarnings worker 跑完清洗后，对每道题用 n-gram + Jaccard 在同课程已有题里找相似项的结果。
	// 非阻塞，仅用于提醒用户和 admin。
	RawText       string         `gorm:"type:text"        json:"raw_text"`
	ParsedJSON    datatypes.JSON `gorm:"type:jsonb"       json:"parsed_json"`
	DedupWarnings datatypes.JSON `gorm:"type:jsonb"       json:"dedup_warnings,omitempty"`

	LLMModel string `gorm:"size:64"                    json:"llm_model"`

	ReviewerID  *PrimaryKey `gorm:"index"                  json:"reviewer_id"`
	ReviewedAt  *time.Time  `                              json:"reviewed_at"`
	PublishedAt *time.Time  `                              json:"published_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IngestJob 状态常量。
const (
	IngestStatusPending         = "pending"
	IngestStatusProcessing      = "processing"
	IngestStatusAwaitingReview  = "awaiting_review"
	IngestStatusPublished       = "published"
	IngestStatusRejected        = "rejected"
	IngestStatusFailed          = "failed"
)

// IngestJob 类型常量。
const (
	IngestKindQuestion    = "question"
	IngestKindExplanation = "explanation"
)
