package models

// Course 对应数据库 courses 表，表示一门课程。
type Course struct {
	ID          string  `gorm:"primaryKey;size:128" json:"id"` // xskcdm
	Code        string  `gorm:"size:128;index"      json:"code"` // xskcdm
	Name        string  `gorm:"size:256"           json:"name"`
	College     string  `gorm:"size:256"           json:"college"`
	Category    string  `gorm:"size:128"           json:"category"`
	Credits     float64 `gorm:"type:numeric(4,1)"  json:"credits"`
	Description string  `gorm:"type:text"          json:"description"`
}
