package models

import "time"

// TestPaper 对应数据库 testpapers 表，表示一份试卷。
type TestPaper struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	CourseID  string    `gorm:"size:64;index"            json:"course_id"`
	Name      string    `gorm:"size:256"                 json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
