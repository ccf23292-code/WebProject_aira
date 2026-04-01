package services

import (
	"encoding/json"
	"net/http"
	"time"

	"gorm.io/gorm"

	"warehouse-web/models"
)

// PaperService 提供课程、试卷、题目的查询能力（数据库版本）。
type PaperService struct {
	db *gorm.DB
}

// CreatePaperRequest 是创建试卷请求体。
type CreatePaperRequest struct {
	CourseID string `json:"course_id"`
	Name     string `json:"name"`
}

// UpdatePaperRequest 是更新试卷请求体。
type UpdatePaperRequest struct {
	Name string `json:"name"`
}

// UpdateProblemRequest 是更新题目请求体。
type UpdateProblemRequest struct {
	Test        string          `json:"test"`
	Answer      string          `json:"answer"`
	Options     []models.Option `json:"options"`
	Explanation string          `json:"explanation"`
	Score       *float64        `json:"score"`
}

// NewPaperService 创建 PaperService。
func NewPaperService(db *gorm.DB) *PaperService {
	return &PaperService{db: db}
}

// ListPapers 根据 courseID 返回该课程下的试卷列表。
func (s *PaperService) ListPapers(courseID string) ([]models.TestPaper, error) {
	var result []models.TestPaper
	if err := s.db.Where("course_id = ?", courseID).Order("id ASC").Find(&result).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "paper list failed")
	}
	return result, nil
}

// ListPapersByCourse 与控制器约定保持一致，返回指定课程下的试卷。
func (s *PaperService) ListPapersByCourse(courseID string) []models.TestPaper {
	papers, _ := s.ListPapers(courseID)
	return papers
}

// ListProblems 根据 paperID 返回该试卷下的题目列表。
func (s *PaperService) ListProblems(paperID uint64) ([]models.Problem, error) {
	var result []models.Problem
	if err := s.db.Where("testpaper_id = ?", paperID).Order("\"order\" ASC").Find(&result).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "problem list failed")
	}
	for i := range result {
		s.inflateOptions(&result[i])
	}
	return result, nil
}

// ListProblemsByPaper 与控制器约定保持一致，返回指定试卷下的题目。
func (s *PaperService) ListProblemsByPaper(paperID uint64) []models.Problem {
	problems, _ := s.ListProblems(paperID)
	return problems
}

// GetProblem 根据 problemID 返回单道题目。
func (s *PaperService) GetProblem(problemID uint64) (*models.Problem, error) {
	var p models.Problem
	if err := s.db.Where("id = ?", problemID).First(&p).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, newServiceError("not_found", http.StatusNotFound, "problem not found")
		}
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "problem load failed")
	}
	s.inflateOptions(&p)
	return &p, nil
}

// GetPaper 根据 paperID 返回试卷信息。
func (s *PaperService) GetPaper(paperID uint64) (*models.TestPaper, error) {
	var p models.TestPaper
	if err := s.db.Where("id = ?", paperID).First(&p).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, newServiceError("not_found", http.StatusNotFound, "paper not found")
		}
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "paper load failed")
	}
	return &p, nil
}

// ────────────────────────── 管理方法（admin） ──────────────────────────

// CreatePaper 创建一份新试卷。
func (s *PaperService) CreatePaper(req CreatePaperRequest) (models.TestPaper, error) {
	if req.CourseID == "" {
		return models.TestPaper{}, newServiceError("invalid_request", http.StatusBadRequest, "course_id 不能为空")
	}
	if req.Name == "" {
		return models.TestPaper{}, newServiceError("invalid_request", http.StatusBadRequest, "name 不能为空")
	}

	paper := models.TestPaper{
		CourseID: req.CourseID,
		Name:     req.Name,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.db.Create(&paper).Error; err != nil {
		return models.TestPaper{}, newServiceError("internal_error", http.StatusInternalServerError, "create paper failed")
	}
	return paper, nil
}

// UpdatePaper 更新试卷基本信息。
func (s *PaperService) UpdatePaper(id uint64, req UpdatePaperRequest) (*models.TestPaper, error) {
	if req.Name == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "name 不能为空")
	}

	var paper models.TestPaper
	if err := s.db.Where("id = ?", id).First(&paper).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, newServiceError("not_found", http.StatusNotFound, "paper not found")
		}
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "paper load failed")
	}
	paper.Name = req.Name
	if err := s.db.Save(&paper).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "paper update failed")
	}
	return &paper, nil
}

// DeletePaper 删除试卷及其题目。
func (s *PaperService) DeletePaper(id uint64) error {
	if err := s.db.Where("testpaper_id = ?", id).Delete(&models.Problem{}).Error; err != nil {
		return newServiceError("internal_error", http.StatusInternalServerError, "problem delete failed")
	}
	res := s.db.Where("id = ?", id).Delete(&models.TestPaper{})
	if res.Error != nil {
		return newServiceError("internal_error", http.StatusInternalServerError, "paper delete failed")
	}
	if res.RowsAffected == 0 {
		return newServiceError("not_found", http.StatusNotFound, "paper not found")
	}
	return nil
}

// UpdateProblem 更新题目内容。
func (s *PaperService) UpdateProblem(id uint64, req UpdateProblemRequest) (*models.Problem, error) {
	var problem models.Problem
	if err := s.db.Where("id = ?", id).First(&problem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, newServiceError("not_found", http.StatusNotFound, "problem not found")
		}
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "problem load failed")
	}

	if req.Test != "" {
		problem.Test = req.Test
	}
	if req.Answer != "" {
		problem.Answer = req.Answer
	}
	if req.Explanation != "" {
		problem.Explanation = req.Explanation
	}
	if req.Score != nil {
		problem.Score = *req.Score
	}
	if req.Options != nil {
		problem.Options = req.Options
		payload, _ := json.Marshal(req.Options)
		problem.OptionsJSON = payload
	}

	if err := s.db.Save(&problem).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "problem update failed")
	}
	s.inflateOptions(&problem)
	return &problem, nil
}

func (s *PaperService) inflateOptions(problem *models.Problem) {
	if len(problem.OptionsJSON) == 0 {
		return
	}
	var options []models.Option
	if err := json.Unmarshal(problem.OptionsJSON, &options); err == nil {
		problem.Options = options
	}
}
