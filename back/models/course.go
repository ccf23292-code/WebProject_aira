package models

// Course 对应数据库 courses 表，表示一门课程。
type Course struct {
	ID          string `gorm:"primaryKey;size:64" json:"id"`
	Name        string `gorm:"size:128"           json:"name"`
	Description string `gorm:"size:512"           json:"description"`
}
