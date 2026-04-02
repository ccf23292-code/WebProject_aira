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

// GetCourseComments 获取某门课程的所有课程评价。
func (s *CourseService) GetCourseComments(courseID string) ([]models.CourseComment, error) {
	courseID = strings.TrimSpace(courseID)
	if courseID == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "course_id 不能为空")
	}

	db := s.db.Model(&models.CourseComment{}).Where("course_id = ?", courseID)
	var comments []models.CourseComment
	if err := db.Order("id DESC").Find(&comments).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load course comments")
	}
	return comments, nil
}

// GetTeacherComments 获取某门课程某教师的所有教师评价。
func (s *CourseService) GetTeacherComments(courseID, teacherID string) ([]models.TeacherComment, error) {
	courseID = strings.TrimSpace(courseID)
	teacherID = strings.TrimSpace(teacherID)
	if courseID == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "course_id 不能为空")
	}
	if teacherID == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "teacher_id 不能为空")
	}

	db := s.db.Model(&models.TeacherComment{}).Where("course_id = ? AND teacher_id = ?", courseID, teacherID)
	var comments []models.TeacherComment
	if err := db.Order("id DESC").Find(&comments).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load teacher comments")
	}
	return comments, nil
}

// GetGradingStandards 获取某门课程某教师的评分标准。
func (s *CourseService) GetGradingStandards(courseID, teacherID string) ([]models.GradingStandard, error) {
	courseID = strings.TrimSpace(courseID)
	teacherID = strings.TrimSpace(teacherID)
	if courseID == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "course_id 不能为空")
	}
	if teacherID == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "teacher_id 不能为空")
	}

	db := s.db.Model(&models.GradingStandard{}).Where("course_id = ? AND teacher_id = ?", courseID, teacherID)
	var standards []models.GradingStandard
	if err := db.Order("id DESC").Find(&standards).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load grading standards")
	}
	return standards, nil
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
