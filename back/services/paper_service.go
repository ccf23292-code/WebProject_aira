package services

import (
	"net/http"
	"sync"
	"time"

	"warehouse-web/models"
)

// PaperService 提供课程、试卷、题目的查询能力（内存版本）。
type PaperService struct {
	mu       sync.RWMutex
	courses  []models.Course
	papers   []models.TestPaper
	problems []models.Problem
	favorites map[uint64][]models.Favorite
	favIDSeq  uint64
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
	Test    string          `json:"test"`
	Answer  string          `json:"answer"`
	Options []models.Option `json:"options"`
}

// NewPaperService 创建并填充示例种子数据的 PaperService。
func NewPaperService() *PaperService {
	svc := &PaperService{
		favorites: make(map[uint64][]models.Favorite),
	}
	svc.seed()
	return svc
}

// ────────────────────────── 查询方法 ──────────────────────────

// ListCourses 返回全部课程列表。
func (s *PaperService) ListCourses() []models.Course {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.Course, len(s.courses))
	copy(out, s.courses)
	return out
}

// ListPapers 根据 courseID 返回该课程下的试卷列表。
func (s *PaperService) ListPapers(courseID string) ([]models.TestPaper, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []models.TestPaper
	for _, p := range s.papers {
		if p.CourseID == courseID {
			result = append(result, p)
		}
	}
	if result == nil {
		result = []models.TestPaper{}
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []models.Problem
	for _, p := range s.problems {
		if p.TestpaperID == paperID {
			result = append(result, p)
		}
	}
	if result == nil {
		result = []models.Problem{}
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.problems {
		if p.ID == problemID {
			return &p, nil
		}
	}
	return nil, newServiceError("not_found", http.StatusNotFound, "problem not found")
}

// GetPaper 根据 paperID 返回试卷信息。
func (s *PaperService) GetPaper(paperID uint64) (*models.TestPaper, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.papers {
		if p.ID == paperID {
			return &p, nil
		}
	}
	return nil, newServiceError("not_found", http.StatusNotFound, "paper not found")
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

	s.mu.Lock()
	defer s.mu.Unlock()

	paper := models.TestPaper{
		CourseID: req.CourseID,
		Name:     req.Name,
	}
	paper.ID = uint64(len(s.papers) + 1)
	if paper.CreatedAt.IsZero() {
		paper.CreatedAt = time.Now().UTC()
	}
	s.papers = append(s.papers, paper)
	return paper, nil
}

// UpdatePaper 更新试卷基本信息。
func (s *PaperService) UpdatePaper(id uint64, req UpdatePaperRequest) (*models.TestPaper, error) {
	if req.Name == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "name 不能为空")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.papers {
		if s.papers[i].ID == id {
			s.papers[i].Name = req.Name
			return &s.papers[i], nil
		}
	}
	return nil, newServiceError("not_found", http.StatusNotFound, "paper not found")
}

// DeletePaper 删除试卷及其下属题目。
func (s *PaperService) DeletePaper(id uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i := range s.papers {
		if s.papers[i].ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return newServiceError("not_found", http.StatusNotFound, "paper not found")
	}
	s.papers = append(s.papers[:idx], s.papers[idx+1:]...)

	// 级联删除题目
	filtered := s.problems[:0]
	for _, p := range s.problems {
		if p.TestpaperID != id {
			filtered = append(filtered, p)
		}
	}
	s.problems = filtered
	return nil
}

// UpdateProblem 更新题目内容。
func (s *PaperService) UpdateProblem(id uint64, req UpdateProblemRequest) (*models.Problem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.problems {
		if s.problems[i].ID == id {
			if req.Test != "" {
				s.problems[i].Test = req.Test
			}
			if req.Answer != "" {
				s.problems[i].Answer = req.Answer
			}
			if req.Options != nil {
				s.problems[i].Options = req.Options
			}
			return &s.problems[i], nil
		}
	}
	return nil, newServiceError("not_found", http.StatusNotFound, "problem not found")
}

// ListFavorites 返回用户收藏列表（分页）。
func (s *PaperService) ListFavorites(userID models.PrimaryKey, page, size int) models.FavoritePage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := s.favorites[userID]
	total := len(items)
	start := (page - 1) * size
	if start < 0 {
		start = 0
	}
	if start > total {
		start = total
	}
	end := start + size
	if end > total {
		end = total
	}

	pageItems := make([]models.FavoriteItem, 0, end-start)
	for _, fav := range items[start:end] {
		problem, paperName := s.problemSnapshotLocked(fav.ProblemID)
		if problem == nil {
			continue
		}
		pageItems = append(pageItems, models.FavoriteItem{
			FavoriteID: fav.ID,
			ProblemID:  fav.ProblemID,
			AddedAt:    fav.AddedAt,
			ProblemDetails: models.ProblemDetails{
				TestpaperName: paperName,
				Order:         problem.Order,
				Test:          problem.Test,
			},
		})
	}

	return models.FavoritePage{
		Total: total,
		Page:  page,
		Size:  size,
		Items: pageItems,
	}
}

// AddFavorite 为用户添加一条收藏。
func (s *PaperService) AddFavorite(userID models.PrimaryKey, problemID uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.findProblemLocked(problemID) == nil {
		return newServiceError("not_found", http.StatusNotFound, "problem not found")
	}
	for _, fav := range s.favorites[userID] {
		if fav.ProblemID == problemID {
			return nil
		}
	}

	s.favIDSeq++
	s.favorites[userID] = append(s.favorites[userID], models.Favorite{
		ID:        s.favIDSeq,
		UserID:    userID,
		ProblemID: problemID,
		AddedAt:   time.Now().UTC(),
	})
	return nil
}

// RemoveFavorite 删除用户的一条收藏。
func (s *PaperService) RemoveFavorite(userID models.PrimaryKey, problemID uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := s.favorites[userID]
	idx := -1
	for i := range items {
		if items[i].ProblemID == problemID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return newServiceError("not_found", http.StatusNotFound, "favorite not found")
	}
	s.favorites[userID] = append(items[:idx], items[idx+1:]...)
	return nil
}

func (s *PaperService) findProblemLocked(problemID uint64) *models.Problem {
	for i := range s.problems {
		if s.problems[i].ID == problemID {
			return &s.problems[i]
		}
	}
	return nil
}

func (s *PaperService) problemSnapshotLocked(problemID uint64) (*models.Problem, string) {
	var found *models.Problem
	for i := range s.problems {
		if s.problems[i].ID == problemID {
			p := s.problems[i]
			found = &p
			break
		}
	}
	if found == nil {
		return nil, ""
	}
	paperName := ""
	for i := range s.papers {
		if s.papers[i].ID == found.TestpaperID {
			paperName = s.papers[i].Name
			break
		}
	}
	return found, paperName
}

// ────────────────────────── 种子数据 ──────────────────────────

func (s *PaperService) seed() {
	s.courses = []models.Course{
		{ID: "course-101", Name: "高等数学", Description: "2026春夏学期高等数学课程资料"},
		{ID: "course-102", Name: "线性代数", Description: "2026春夏学期线性代数课程资料"},
		{ID: "course-103", Name: "概率论与数理统计", Description: "2026春夏学期概率论课程资料"},
	}

	s.papers = []models.TestPaper{
		{ID: 1, CourseID: "course-101", Name: "2026-spring-summer 期末卷", CreatedAt: time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)},
		{ID: 2, CourseID: "course-101", Name: "2025-autumn-winter 期末卷", CreatedAt: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)},
		{ID: 3, CourseID: "course-102", Name: "2026-spring-summer 期末卷", CreatedAt: time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)},
	}

	s.problems = []models.Problem{
		{
			ID: 1001, TestpaperID: 1, Order: 1,
			Test: "函数 f(x) = x² 的导数是？",
			Options: []models.Option{
				{Option: "A", Text: "x"},
				{Option: "B", Text: "2x"},
				{Option: "C", Text: "x²"},
				{Option: "D", Text: "2"},
			},
			Answer: "B",
		},
		{
			ID: 1002, TestpaperID: 1, Order: 2,
			Test: "∫ 2x dx = ?",
			Options: []models.Option{
				{Option: "A", Text: "x² + C"},
				{Option: "B", Text: "2x² + C"},
				{Option: "C", Text: "x + C"},
				{Option: "D", Text: "2 + C"},
			},
			Answer: "A",
		},
		{
			ID: 1003, TestpaperID: 1, Order: 3,
			Test: "lim(x→0) sin(x)/x = ?",
			Options: []models.Option{
				{Option: "A", Text: "0"},
				{Option: "B", Text: "1"},
				{Option: "C", Text: "∞"},
				{Option: "D", Text: "不存在"},
			},
			Answer: "B",
		},
		{
			ID: 2001, TestpaperID: 2, Order: 1,
			Test: "泰勒公式中 e^x 在 x=0 处的展开式首项是？",
			Options: []models.Option{
				{Option: "A", Text: "0"},
				{Option: "B", Text: "1"},
				{Option: "C", Text: "x"},
				{Option: "D", Text: "e"},
			},
			Answer: "B",
		},
		{
			ID: 3001, TestpaperID: 3, Order: 1,
			Test: "单位矩阵的行列式值为？",
			Options: []models.Option{
				{Option: "A", Text: "0"},
				{Option: "B", Text: "1"},
				{Option: "C", Text: "-1"},
				{Option: "D", Text: "n"},
			},
			Answer: "B",
		},
	}
}
