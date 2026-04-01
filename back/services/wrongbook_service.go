package services

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"warehouse-web/models"
)

// WrongBookService 提供错题本相关能力。
type WrongBookService struct {
	db    *gorm.DB
	paper *PaperService
}

func NewWrongBookService(db *gorm.DB, paper *PaperService) *WrongBookService {
	return &WrongBookService{db: db, paper: paper}
}

type WrongBookItem struct {
	ProblemID   uint64    `json:"problem_id"`
	PaperID     uint64    `json:"paper_id"`
	Order       int       `json:"order"`
	Test        string    `json:"test"`
	Status      string    `json:"status"`
	Note        string    `json:"note"`
	WrongCount  int       `json:"wrong_count"`
	LastWrongAt time.Time `json:"last_wrong_at"`
}

type WrongBookCourseGroup struct {
	CourseID       string          `json:"course_id"`
	CourseName     string          `json:"course_name"`
	LastPracticeAt *time.Time      `json:"last_practice_at"`
	Items          []WrongBookItem `json:"items"`
}

type WrongBookResponse struct {
	Courses []WrongBookCourseGroup `json:"courses"`
}

// ListWrongBook 按课程分组返回错题本。
func (s *WrongBookService) ListWrongBook(userID uint64, status string) (*WrongBookResponse, error) {
	courseNameMap := make(map[string]string)
	var courseList []models.Course
	if err := s.db.Find(&courseList).Error; err == nil {
		for _, c := range courseList {
			courseNameMap[c.ID] = c.Name
		}
	}

	var rows []models.WrongQuestion
	query := s.db.Where("user_id = ?", userID)
	status = strings.TrimSpace(status)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("last_wrong_at DESC").Find(&rows).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load wrong book")
	}

	lastMap, err := s.lastPracticeByCourse(userID)
	if err != nil {
		return nil, err
	}

	courseMap := make(map[string]*WrongBookCourseGroup)
	for _, row := range rows {
		problem, err := s.paper.GetProblem(row.ProblemID)
		if err != nil || problem == nil {
			continue
		}
		paper, _ := s.paper.GetPaper(problem.TestpaperID)
		courseID := row.CourseID
		courseName := courseNameMap[courseID]
		if paper != nil {
			courseID = paper.CourseID
		}

		group, ok := courseMap[courseID]
		if !ok {
			var lastPtr *time.Time
			if last, exists := lastMap[courseID]; exists {
				lastPtr = &last
			}
			courseMap[courseID] = &WrongBookCourseGroup{
				CourseID:       courseID,
				CourseName:     courseName,
				LastPracticeAt: lastPtr,
			}
			group = courseMap[courseID]
		}
		// CourseName 暂用 courseID，后续可由课程表补全

		group.Items = append(group.Items, WrongBookItem{
			ProblemID:   row.ProblemID,
			PaperID:     row.PaperID,
			Order:       problem.Order,
			Test:        problem.Test,
			Status:      row.Status,
			Note:        row.Note,
			WrongCount:  row.WrongCount,
			LastWrongAt: row.LastWrongAt,
		})
	}

	groups := make([]WrongBookCourseGroup, 0, len(courseMap))
	for _, group := range courseMap {
		groups = append(groups, *group)
	}
	// 排序：按 lastPracticeAt DESC
	sort.SliceStable(groups, func(i, j int) bool {
		li := groups[i].LastPracticeAt
		lj := groups[j].LastPracticeAt
		if li == nil && lj == nil {
			return groups[i].CourseID < groups[j].CourseID
		}
		if li == nil {
			return false
		}
		if lj == nil {
			return true
		}
		return li.After(*lj)
	})

	return &WrongBookResponse{Courses: groups}, nil
}

func (s *WrongBookService) lastPracticeByCourse(userID uint64) (map[string]time.Time, error) {
	type row struct {
		CourseID string
		LastAt   time.Time
	}
	var rows []row
	if err := s.db.Model(&models.AnswerRecord{}).
		Select("course_id, MAX(answered_at) as last_at").
		Where("user_id = ?", userID).
		Group("course_id").
		Scan(&rows).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load practice time")
	}
	result := make(map[string]time.Time)
	for _, r := range rows {
		result[r.CourseID] = r.LastAt
	}
	return result, nil
}

// UpdateWrongQuestion 更新备注/状态。
func (s *WrongBookService) UpdateWrongQuestion(userID uint64, problemID uint64, note *string, status *string) (*models.WrongQuestion, error) {
	var wrong models.WrongQuestion
	if err := s.db.Where("user_id = ? AND problem_id = ?", userID, problemID).First(&wrong).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, newServiceError("not_found", http.StatusNotFound, "wrong question not found")
		}
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to update wrong question")
	}

	if note != nil {
		wrong.Note = strings.TrimSpace(*note)
	}
	if status != nil {
		trimmed := strings.TrimSpace(*status)
		if trimmed != "" {
			wrong.Status = trimmed
		}
	}
	if err := s.db.Save(&wrong).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to update wrong question")
	}
	return &wrong, nil
}

// RemoveWrongQuestion 删除单条错题。
func (s *WrongBookService) RemoveWrongQuestion(userID uint64, problemID uint64) error {
	res := s.db.Where("user_id = ? AND problem_id = ?", userID, problemID).Delete(&models.WrongQuestion{})
	if res.Error != nil {
		return newServiceError("internal_error", http.StatusInternalServerError, "failed to delete wrong question")
	}
	if res.RowsAffected == 0 {
		return newServiceError("not_found", http.StatusNotFound, "wrong question not found")
	}
	return nil
}

// ClearTrash 清空垃圾篓。
func (s *WrongBookService) ClearTrash(userID uint64) error {
	if err := s.db.Where("user_id = ? AND status = ?", userID, models.WrongStatusTrash).
		Delete(&models.WrongQuestion{}).Error; err != nil {
		return newServiceError("internal_error", http.StatusInternalServerError, "failed to clear trash")
	}
	return nil
}
