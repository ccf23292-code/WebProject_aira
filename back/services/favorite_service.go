package services

import (
	"net/http"
	"sort"
	"time"

	"gorm.io/gorm"

	"warehouse-web/models"
)

// FavoriteService 提供收藏相关的持久化操作。
type FavoriteService struct {
	db      *gorm.DB
	paper   *PaperService
}

func NewFavoriteService(db *gorm.DB, paper *PaperService) *FavoriteService {
	return &FavoriteService{db: db, paper: paper}
}

// ListFavorites 返回用户收藏列表（分页）。
func (s *FavoriteService) ListFavorites(userID models.PrimaryKey, page, size int) (models.FavoritePage, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}

	var total int64
	if err := s.db.Model(&models.Favorite{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return models.FavoritePage{}, newServiceError("internal_error", http.StatusInternalServerError, "failed to load favorites")
	}

	var favs []models.Favorite
	if err := s.db.Where("user_id = ?", userID).
		Order("added_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&favs).Error; err != nil {
		return models.FavoritePage{}, newServiceError("internal_error", http.StatusInternalServerError, "failed to load favorites")
	}

	items := make([]models.FavoriteItem, 0, len(favs))
	courseNameMap := s.loadCourseNames()
	for _, fav := range favs {
		problem, paper, paperName := s.paperSnapshot(fav.ProblemID)
		if problem == nil {
			continue
		}
		courseID := ""
		courseName := ""
		if paper != nil {
			courseID = paper.CourseID
			courseName = courseNameMap[paper.CourseID]
		}
		items = append(items, models.FavoriteItem{
			FavoriteID: fav.ID,
			ProblemID:  fav.ProblemID,
			CourseID:   courseID,
			CourseName: courseName,
			AddedAt:    fav.AddedAt,
			ProblemDetails: models.ProblemDetails{
				TestpaperName: paperName,
				Order:         problem.Order,
				Test:          problem.Test,
			},
		})
	}

	return models.FavoritePage{
		Total:  int(total),
		Page:   page,
		Size:   size,
		Items:  items,
		Groups: groupFavorites(items),
	}, nil
}

// ListFavoriteIDs 返回收藏题目 ID 列表。
func (s *FavoriteService) ListFavoriteIDs(userID models.PrimaryKey) ([]uint64, error) {
	var ids []uint64
	if err := s.db.Model(&models.Favorite{}).
		Where("user_id = ?", userID).
		Pluck("problem_id", &ids).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load favorites")
	}
	return ids, nil
}

// AddFavorite 添加收藏。
func (s *FavoriteService) AddFavorite(userID models.PrimaryKey, problemID uint64) error {
	if _, err := s.paper.GetProblem(problemID); err != nil {
		return newServiceError("not_found", http.StatusNotFound, "problem not found")
	}

	var existing models.Favorite
	if err := s.db.Where("user_id = ? AND problem_id = ?", userID, problemID).First(&existing).Error; err == nil {
		return nil
	}

	fav := models.Favorite{
		UserID:    userID,
		ProblemID: problemID,
		AddedAt:   time.Now().UTC(),
	}
	if err := s.db.Create(&fav).Error; err != nil {
		return newServiceError("internal_error", http.StatusInternalServerError, "failed to add favorite")
	}
	return nil
}

// RemoveFavorite 删除收藏。
func (s *FavoriteService) RemoveFavorite(userID models.PrimaryKey, problemID uint64) error {
	res := s.db.Where("user_id = ? AND problem_id = ?", userID, problemID).Delete(&models.Favorite{})
	if res.Error != nil {
		return newServiceError("internal_error", http.StatusInternalServerError, "failed to remove favorite")
	}
	if res.RowsAffected == 0 {
		return newServiceError("not_found", http.StatusNotFound, "favorite not found")
	}
	return nil
}

func (s *FavoriteService) paperSnapshot(problemID uint64) (*models.Problem, *models.TestPaper, string) {
	problem, err := s.paper.GetProblem(problemID)
	if err != nil || problem == nil {
		return nil, nil, ""
	}
	paper, _ := s.paper.GetPaper(problem.TestpaperID)
	paperName := ""
	if paper != nil {
		paperName = paper.Name
	}
	return problem, paper, paperName
}

func (s *FavoriteService) loadCourseNames() map[string]string {
	var courses []models.Course
	if err := s.db.Find(&courses).Error; err != nil {
		return map[string]string{}
	}
	result := make(map[string]string, len(courses))
	for _, course := range courses {
		result[course.ID] = course.Name
	}
	return result
}

func groupFavorites(items []models.FavoriteItem) []models.FavoriteCourseGroup {
	groupMap := make(map[string]*models.FavoriteCourseGroup)
	order := make([]string, 0)
	for _, item := range items {
		key := item.CourseID
		if key == "" {
			key = "unknown"
		}
		group, exists := groupMap[key]
		if !exists {
			group = &models.FavoriteCourseGroup{
				CourseID:   item.CourseID,
				CourseName: item.CourseName,
				Items:      []models.FavoriteItem{},
			}
			groupMap[key] = group
			order = append(order, key)
		}
		group.Items = append(group.Items, item)
	}

	groups := make([]models.FavoriteCourseGroup, 0, len(order))
	for _, key := range order {
		group := groupMap[key]
		sort.SliceStable(group.Items, func(i, j int) bool {
			return group.Items[i].AddedAt.After(group.Items[j].AddedAt)
		})
		groups = append(groups, *group)
	}
	return groups
}
