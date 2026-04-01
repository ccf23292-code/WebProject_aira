package services

import (
	"net/http"
	"strings"

	"gorm.io/gorm"

	"warehouse-web/models"
)

// CourseService 提供课程查询能力。
type CourseService struct {
	db *gorm.DB
}

// NewCourseService 创建 CourseService。
func NewCourseService(db *gorm.DB) *CourseService {
	return &CourseService{db: db}
}

// ListCourses 支持按课程名称或课程代码搜索。
func (s *CourseService) ListCourses(query string) ([]models.Course, error) {
	db := s.db.Model(&models.Course{})
	q := strings.TrimSpace(query)
	if q != "" {
		like := "%" + q + "%"
		db = db.Where("name ILIKE ? OR code ILIKE ? OR id ILIKE ?", like, like, like)
	}

	var courses []models.Course
	if err := db.Order("name ASC").Find(&courses).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load courses")
	}
	return courses, nil
}

// GetCourse 返回课程详情。
func (s *CourseService) GetCourse(courseID string) (*models.Course, error) {
	courseID = strings.TrimSpace(courseID)
	if courseID == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "course_id 不能为空")
	}

	var course models.Course
	if err := s.db.Where("id = ? OR code = ?", courseID, courseID).First(&course).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, newServiceError("not_found", http.StatusNotFound, "course not found")
		}
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load course")
	}
	return &course, nil
}
