package models

import "gorm.io/datatypes"

// Option 表示选择题的一个选项。
type Option struct {
	Option string `json:"option"` // 选项字母，如 "A"
	Text   string `json:"text"`   // 选项内容
}

// Problem 对应数据库 problems 表，表示一道题目。
// Status 字段表示题目的状态，如 "pending", "processing", "processed", "error" 等。
type Problem struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement"                 json:"id"`
	TestpaperID  uint64         `gorm:"index;uniqueIndex:idx_source_paper"       json:"testpaper_id"`
	SourceID     string         `gorm:"size:64;uniqueIndex:idx_source_paper"     json:"source_id"`
	Order        int            `json:"order"`
	SequenceID   int            `json:"sequence_id"`
	QuestionType string         `gorm:"size:32;index"                            json:"question_type"`
	Category     string         `gorm:"size:128;index"                           json:"category"`
	SourceURL    string         `gorm:"size:512"                                 json:"source_url"`
	Test         string         `gorm:"type:text"                                json:"test"`
	Answer       string         `gorm:"type:text"                                json:"answer"`
	Score        float64        `gorm:"type:numeric(6,2)"                        json:"score"`
	Explanation  string         `gorm:"type:text"                                json:"explanation"`
	Difficulty   string         `gorm:"size:32"                                  json:"difficulty"`
	OptionsJSON  datatypes.JSON `gorm:"type:jsonb"                               json:"-"`
	TagsJSON     datatypes.JSON `gorm:"type:jsonb"                               json:"-"`
	Options      []Option       `gorm:"-"                                        json:"options"`
	Status       string         `gorm:"size:32;index"                            json:"status"`
	LLM          string         `gorm:"type:text"                                json:"llm"`
	LLMAnswer    string         `gorm:"type:text"                                json:"llm_answer"`
}
