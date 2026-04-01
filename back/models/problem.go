package models

// Option 表示选择题的一个选项。
type Option struct {
	Option string `json:"option"` // 选项字母，如 "A"
	Text   string `json:"text"`   // 选项内容
}

// Problem 对应数据库 problems 表，表示一道题目。
type Problem struct {
	ID          uint64   `gorm:"primaryKey;autoIncrement" json:"id"`
	TestpaperID uint64   `gorm:"index"                    json:"testpaper_id"`
	Order       int      `json:"order"`
	Test        string   `gorm:"type:text"                json:"test"`
	Options     []Option `gorm:"-"                        json:"options"`
	Answer      string   `gorm:"size:16"                  json:"answer"`
}
