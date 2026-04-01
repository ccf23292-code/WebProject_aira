package models

import "time"

// ProblemExplanation 是用户对题目的题解投稿。
// 约束：同一用户对同一道题最多保留一条解析，通过更新实现编辑。
type ProblemExplanation struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"                     json:"id"`
	ProblemID uint64    `gorm:"index;uniqueIndex:idx_problem_explanation"    json:"problem_id"`
	UserID    uint64    `gorm:"index;uniqueIndex:idx_problem_explanation"    json:"user_id"`
	ContentMD string    `gorm:"type:text"                                    json:"content_md"`
	UpVotes   int       `gorm:"default:0"                                    json:"up_votes"`
	DownVotes int       `gorm:"default:0"                                    json:"down_votes"`
	Score     int       `gorm:"default:0;index"                              json:"score"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProblemExplanationVote 记录用户对题解的点赞 / 踩。
type ProblemExplanationVote struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement"              json:"id"`
	ExplanationID uint64    `gorm:"index;uniqueIndex:idx_explanation_vote" json:"explanation_id"`
	UserID        uint64    `gorm:"index;uniqueIndex:idx_explanation_vote" json:"user_id"`
	Value         int       `json:"value"` // 1 = upvote, -1 = downvote
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ProblemExplanationItem 是题解响应视图。
type ProblemExplanationItem struct {
	ID         uint64    `json:"id"`
	ProblemID  uint64    `json:"problem_id"`
	AuthorID   uint64    `json:"author_id"`
	AuthorName string    `json:"author_name"`
	ContentMD  string    `json:"content_md"`
	UpVotes    int       `json:"up_votes"`
	DownVotes  int       `json:"down_votes"`
	MyVote     int       `json:"my_vote"`
	CanEdit    bool      `json:"can_edit"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ProblemExplanationListData 是题解列表响应。
type ProblemExplanationListData struct {
	OfficialExplanation string                   `json:"official_explanation"`
	Items               []ProblemExplanationItem `json:"items"`
	MyItem              *ProblemExplanationItem  `json:"my_item,omitempty"`
}
