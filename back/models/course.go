package models

import "time"

// Course 对应数据库 courses 表，表示一门课程。
type Course struct {
	ID          string  `gorm:"primaryKey;size:128" json:"id"`   // xskcdm
	Code        string  `gorm:"size:128;index"      json:"code"` // xskcdm
	Name        string  `gorm:"size:256"           json:"name"`
	College     string  `gorm:"size:256"           json:"college"`
	Category    string  `gorm:"size:128"           json:"category"`
	Credits     float64 `gorm:"type:numeric(4,1)"  json:"credits"`
	Description string  `gorm:"type:text"          json:"description"`
}

const (
	CourseDescriptionSubmissionPending  = "pending"
	CourseDescriptionSubmissionApproved = "approved"
	CourseDescriptionSubmissionRejected = "rejected"
)

// CourseDescriptionSubmission stores user-submitted description proposals.
type CourseDescriptionSubmission struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	CourseID   string    `gorm:"size:128;index" json:"course_id"`
	UserID     string    `gorm:"size:128;index" json:"user_id"`
	Content    string    `gorm:"type:text" json:"content"`
	Status     string    `gorm:"size:32;index" json:"status"`
	ReviewedBy string    `gorm:"size:128" json:"reviewed_by,omitempty"`
	ReviewNote string    `gorm:"type:text" json:"review_note,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TeacherSubmission stores user-submitted teacher metadata proposals.
type TeacherSubmission struct {
	ID                 uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	CourseID           string    `gorm:"size:128;index" json:"course_id"`
	UserID             string    `gorm:"size:128;index" json:"user_id"`
	Name               string    `gorm:"size:256" json:"name"`
	Title              string    `gorm:"size:256" json:"title,omitempty"`
	Status             string    `gorm:"size:32;index" json:"status"`
	ReviewedBy         string    `gorm:"size:128" json:"reviewed_by,omitempty"`
	ReviewNote         string    `gorm:"type:text" json:"review_note,omitempty"`
	PublishedTeacherID string    `gorm:"size:128" json:"published_teacher_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// GradingStandardSubmission stores user-submitted grading standard proposals.
type GradingStandardSubmission struct {
	ID                  uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	CourseID            string    `gorm:"size:128;index" json:"course_id"`
	TeacherID           string    `gorm:"size:128;index" json:"teacher_id"`
	UserID              string    `gorm:"size:128;index" json:"user_id"`
	Description         string    `gorm:"type:text" json:"description,omitempty"`
	Standard            string    `gorm:"type:text" json:"standard,omitempty"`
	StandardImg         string    `gorm:"type:text" json:"standard_img,omitempty"`
	Status              string    `gorm:"size:32;index" json:"status"`
	ReviewedBy          string    `gorm:"size:128" json:"reviewed_by,omitempty"`
	ReviewNote          string    `gorm:"type:text" json:"review_note,omitempty"`
	PublishedStandardID uint      `json:"published_standard_id,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// Teacher 对应数据库 teachers 表，表示一位教师。
type Teacher struct {
	ID        string    `gorm:"primaryKey;size:128" json:"id"`        // jsdm
	CourseID  string    `gorm:"size:128;index"      json:"course_id"` // xskcdm
	Name      string    `gorm:"size:256"           json:"name"`
	Title     string    `gorm:"size:256"           json:"title,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GradingStandard 对应数据库 grading_standards 表，表示某门课程某教师的评分标准。
// Img存储在 ./back/course-center/grading/{CourseID}-{TeacherID}/filename.{ext}，URL存储在数据库中，访问时通过后端接口提供访问链接。
type GradingStandard struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	CourseID    string    `gorm:"size:128;index" json:"course_id"`  // xskcdm
	TeacherID   string    `gorm:"size:128;index" json:"teacher_id"` // jsdm
	TeacherName string    `gorm:"-" json:"teacher_name,omitempty"`
	Description string    `gorm:"type:text" json:"description,omitempty"`  // 评分标准描述文本内容
	Standard    string    `gorm:"type:text" json:"standard,omitempty"`     // 评分标准文本内容
	StandardImg string    `gorm:"type:text" json:"standard_img,omitempty"` // 存储评分标准图片的 URL
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TeacherComment 对应数据库 teacher_comments 表，表示对某门课程某教师的评价。
type TeacherComment struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CourseID    string    `gorm:"size:128;index" json:"course_id"` // xskcdm
	UserID      string    `gorm:"size:128;index" json:"user_id"`   // 用户 ID
	UserName    string    `gorm:"-" json:"user_name,omitempty"`
	AvatarURL   string    `gorm:"-" json:"avatar_url,omitempty"`
	TeacherID   string    `gorm:"size:128;index" json:"teacher_id"` // jsdm
	TeacherName string    `gorm:"-" json:"teacher_name,omitempty"`
	Comment     string    `gorm:"type:text" json:"comment"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CourseComment 对应数据库 course_comments 表，表示对某门课程的评价。
type CourseComment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CourseID  string    `gorm:"size:128;index" json:"course_id"` // xskcdm
	UserID    string    `gorm:"size:128;index" json:"user_id"`   // 用户 ID
	UserName  string    `gorm:"-" json:"user_name,omitempty"`
	AvatarURL string    `gorm:"-" json:"avatar_url,omitempty"`
	Comment   string    `gorm:"type:text" json:"comment"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
