package models

import "time"

// Favorite 对应数据库 favorites 表，表示用户收藏的一道题目。
type Favorite struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"favorite_id"`
	UserID    uint64    `gorm:"index"                    json:"user_id"`
	ProblemID uint64    `gorm:"index"                    json:"problem_id"`
	AddedAt   time.Time `json:"added_at"`
}

// FavoriteItem 是收藏列表接口返回给前端的聚合结构。
type FavoriteItem struct {
	FavoriteID     uint64         `json:"favorite_id"`
	ProblemID      uint64         `json:"problem_id"`
	CourseID       string         `json:"course_id"`
	CourseName     string         `json:"course_name"`
	AddedAt        time.Time      `json:"added_at"`
	ProblemDetails ProblemDetails `json:"problem_details"`
}

// FavoriteCourseGroup 表示按课程聚合的收藏结果。
type FavoriteCourseGroup struct {
	CourseID   string         `json:"course_id"`
	CourseName string         `json:"course_name"`
	Items      []FavoriteItem `json:"items"`
}

// ProblemDetails 是嵌套在收藏项中的题目摘要。
type ProblemDetails struct {
	TestpaperName string `json:"testpaper_name"`
	Order         int    `json:"order"`
	Test          string `json:"test"`
}

// FavoritePage 是收藏列表的分页响应。
type FavoritePage struct {
	Total  int                 `json:"total"`
	Page   int                 `json:"page"`
	Size   int                 `json:"size"`
	Items  []FavoriteItem      `json:"items"`
	Groups []FavoriteCourseGroup `json:"groups"`
}
