package services

import (
	"net/http"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"warehouse-web/models"
)

// CourseService 提供课程查询能力。
type CourseService struct {
	db *gorm.DB
}

// AddCourseCommentRequest is the payload for adding a course comment.
type AddCourseCommentRequest struct {
	Comment string `json:"comment" binding:"required"`
}

// AddTeacherCommentRequest is the payload for adding a teacher comment.
type AddTeacherCommentRequest struct {
	Comment string `json:"comment" binding:"required"`
}

// AddTeacherRequest is the payload for adding a teacher entry.
type AddTeacherRequest struct {
	ID    string `json:"id"`
	Name  string `json:"name" binding:"required"`
	Title string `json:"title"`
}

// AddGradingStandardRequest is the payload for adding a grading standard.
type AddGradingStandardRequest struct {
	Description string `json:"description"`
	Standard    string `json:"standard"`
	StandardImg string `json:"standard_img"`
}

// SubmitCourseDescriptionRequest is the payload for a user suggestion.
type SubmitCourseDescriptionRequest struct {
	Content string `json:"content" binding:"required"`
}

// ReviewCourseDescriptionRequest is the payload for admin review.
type ReviewCourseDescriptionRequest struct {
	Action     string `json:"action" binding:"required"`
	ReviewNote string `json:"review_note"`
}

// NewCourseService 创建 CourseService。
func NewCourseService(db *gorm.DB) *CourseService {
	return &CourseService{db: db}
}

// SubmitCourseDescription stores a pending description proposal.
func (s *CourseService) SubmitCourseDescription(courseID string, userID models.PrimaryKey, content string) (*models.CourseDescriptionSubmission, error) {
	courseID = strings.TrimSpace(courseID)
	content = strings.TrimSpace(content)
	if courseID == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "course_id 不能为空")
	}
	if content == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "content 不能为空")
	}
	if _, err := s.GetCourse(courseID); err != nil {
		return nil, err
	}

	item := models.CourseDescriptionSubmission{
		CourseID: courseID,
		UserID:   strconv.FormatUint(uint64(userID), 10),
		Content:  content,
		Status:   models.CourseDescriptionSubmissionPending,
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to create description submission")
	}
	return &item, nil
}

// ListMyCourseDescriptionSubmissions returns the current user's submissions for one course.
func (s *CourseService) ListMyCourseDescriptionSubmissions(courseID string, userID models.PrimaryKey) ([]models.CourseDescriptionSubmission, error) {
	courseID = strings.TrimSpace(courseID)
	if courseID == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "course_id 不能为空")
	}

	var items []models.CourseDescriptionSubmission
	if err := s.db.
		Where("course_id = ? AND user_id = ?", courseID, strconv.FormatUint(uint64(userID), 10)).
		Order("id DESC").
		Find(&items).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load description submissions")
	}
	return items, nil
}

// ListCourseDescriptionSubmissions returns submissions filtered by status for admins.
func (s *CourseService) ListCourseDescriptionSubmissions(status string) ([]models.CourseDescriptionSubmission, error) {
	query := s.db.Model(&models.CourseDescriptionSubmission{})
	status = strings.TrimSpace(status)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var items []models.CourseDescriptionSubmission
	if err := query.Order("id DESC").Find(&items).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load description submissions")
	}
	return items, nil
}

// ReviewCourseDescriptionSubmission approves or rejects a pending proposal.
func (s *CourseService) ReviewCourseDescriptionSubmission(submissionID uint64, reviewerID models.PrimaryKey, req ReviewCourseDescriptionRequest) (*models.CourseDescriptionSubmission, error) {
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != "approve" && action != "reject" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "action 必须为 approve 或 reject")
	}

	var item models.CourseDescriptionSubmission
	if err := s.db.Where("id = ?", submissionID).First(&item).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, newServiceError("not_found", http.StatusNotFound, "description submission not found")
		}
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load description submission")
	}

	status := models.CourseDescriptionSubmissionRejected
	if action == "approve" {
		status = models.CourseDescriptionSubmissionApproved
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		item.Status = status
		item.ReviewedBy = strconv.FormatUint(uint64(reviewerID), 10)
		item.ReviewNote = strings.TrimSpace(req.ReviewNote)
		if err := tx.Save(&item).Error; err != nil {
			return err
		}
		if action == "approve" {
			if err := tx.Model(&models.Course{}).
				Where("id = ?", item.CourseID).
				Update("description", item.Content).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to review description submission")
	}
	return &item, nil
}

// ListTeachers returns all teachers for one course.
func (s *CourseService) ListTeachers(courseID string) ([]models.Teacher, error) {
	courseID = strings.TrimSpace(courseID)
	if courseID == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "course_id 不能为空")
	}

	var teachers []models.Teacher
	if err := s.db.Where("course_id = ?", courseID).Order("name ASC").Find(&teachers).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load teachers")
	}
	return teachers, nil
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
	s.hydrateCourseCommentUsers(comments)
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
	s.hydrateTeacherCommentDisplayFields(comments)
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
	s.hydrateGradingTeacherNames(standards)
	return standards, nil
}

// ListCourses 支持按课程名称或课程代码搜索。
func (s *CourseService) ListCourses(query string) ([]models.Course, error) {
	db := s.db.Model(&models.Course{})
	q := strings.TrimSpace(query)
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		db = db.Where(
			"LOWER(name) LIKE ? OR LOWER(code) LIKE ? OR LOWER(id) LIKE ?",
			like,
			like,
			like,
		)
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

// AddCourseComment creates a course comment.
func (s *CourseService) AddCourseComment(courseID string, userID models.PrimaryKey, comment string) (*models.CourseComment, error) {
	courseID = strings.TrimSpace(courseID)
	comment = strings.TrimSpace(comment)
	if courseID == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "course_id 不能为空")
	}
	if comment == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "comment 不能为空")
	}

	item := models.CourseComment{
		CourseID: courseID,
		UserID:   strconv.FormatUint(uint64(userID), 10),
		Comment:  comment,
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to create course comment")
	}
	if name, ok := s.lookupUserDisplayNames([]uint64{uint64(userID)})[strconv.FormatUint(uint64(userID), 10)]; ok {
		item.UserName = name
	}
	return &item, nil
}

// AddTeacher creates a teacher entry for a course.
func (s *CourseService) AddTeacher(courseID string, req AddTeacherRequest) (*models.Teacher, error) {
	courseID = strings.TrimSpace(courseID)
	name := strings.TrimSpace(req.Name)
	title := strings.TrimSpace(req.Title)
	if courseID == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "course_id 不能为空")
	}
	if name == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "teacher name 不能为空")
	}

	teacherID := strings.TrimSpace(req.ID)
	if teacherID == "" {
		teacherID = strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	}

	item := models.Teacher{
		ID:       teacherID,
		CourseID: courseID,
		Name:     name,
		Title:    title,
	}
	if err := s.db.Where("id = ? AND course_id = ?", teacherID, courseID).FirstOrCreate(&item).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to create teacher")
	}
	return &item, nil
}

// AddTeacherComment creates a teacher comment.
func (s *CourseService) AddTeacherComment(courseID, teacherID string, userID models.PrimaryKey, comment string) (*models.TeacherComment, error) {
	courseID = strings.TrimSpace(courseID)
	teacherID = strings.TrimSpace(teacherID)
	comment = strings.TrimSpace(comment)
	if courseID == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "course_id 不能为空")
	}
	if teacherID == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "teacher_id 不能为空")
	}
	if comment == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "comment 不能为空")
	}

	item := models.TeacherComment{
		CourseID:  courseID,
		TeacherID: teacherID,
		UserID:    strconv.FormatUint(uint64(userID), 10),
		Comment:   comment,
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to create teacher comment")
	}
	item.UserName = s.lookupUserDisplayNames([]uint64{uint64(userID)})[strconv.FormatUint(uint64(userID), 10)]
	item.TeacherName = s.lookupTeacherNamesByKey([]string{courseID + "::" + teacherID})[courseID+"::"+teacherID]
	return &item, nil
}

// AddGradingStandard creates a grading standard.
func (s *CourseService) AddGradingStandard(courseID, teacherID string, req AddGradingStandardRequest) (*models.GradingStandard, error) {
	courseID = strings.TrimSpace(courseID)
	teacherID = strings.TrimSpace(teacherID)
	description := strings.TrimSpace(req.Description)
	standard := strings.TrimSpace(req.Standard)
	standardImg := strings.TrimSpace(req.StandardImg)
	if courseID == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "course_id 不能为空")
	}
	if teacherID == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "teacher_id 不能为空")
	}
	if description == "" && standard == "" && standardImg == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "评分标准不能为空")
	}

	item := models.GradingStandard{
		CourseID:    courseID,
		TeacherID:   teacherID,
		Description: description,
		Standard:    standard,
		StandardImg: standardImg,
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to create grading standard")
	}
	item.TeacherName = s.lookupTeacherNamesByKey([]string{courseID + "::" + teacherID})[courseID+"::"+teacherID]
	return &item, nil
}

func (s *CourseService) hydrateCourseCommentUsers(comments []models.CourseComment) {
	if len(comments) == 0 {
		return
	}

	userIDs := make([]uint64, 0, len(comments))
	seen := make(map[string]struct{}, len(comments))
	for _, item := range comments {
		if _, ok := seen[item.UserID]; ok {
			continue
		}
		seen[item.UserID] = struct{}{}
		if id, err := strconv.ParseUint(item.UserID, 10, 64); err == nil {
			userIDs = append(userIDs, id)
		}
	}

	displayNames := s.lookupUserDisplayNames(userIDs)
	for idx := range comments {
		if display, ok := displayNames[comments[idx].UserID]; ok {
			comments[idx].UserName = display
		}
	}
}

func (s *CourseService) hydrateTeacherCommentDisplayFields(comments []models.TeacherComment) {
	if len(comments) == 0 {
		return
	}

	courseComments := make([]models.CourseComment, 0, len(comments))
	teacherKeys := make([]string, 0, len(comments))
	seenTeachers := make(map[string]struct{}, len(comments))
	for _, item := range comments {
		courseComments = append(courseComments, models.CourseComment{UserID: item.UserID})
		key := item.CourseID + "::" + item.TeacherID
		if _, ok := seenTeachers[key]; !ok {
			seenTeachers[key] = struct{}{}
			teacherKeys = append(teacherKeys, key)
		}
	}

	s.hydrateCourseCommentUsers(courseComments)
	teacherNames := s.lookupTeacherNamesByKey(teacherKeys)
	for idx := range comments {
		comments[idx].UserName = courseComments[idx].UserName
		comments[idx].TeacherName = teacherNames[comments[idx].CourseID+"::"+comments[idx].TeacherID]
	}
}

func (s *CourseService) hydrateGradingTeacherNames(standards []models.GradingStandard) {
	if len(standards) == 0 {
		return
	}

	keys := make([]string, 0, len(standards))
	seen := make(map[string]struct{}, len(standards))
	for _, item := range standards {
		key := item.CourseID + "::" + item.TeacherID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}

	teacherNames := s.lookupTeacherNamesByKey(keys)
	for idx := range standards {
		standards[idx].TeacherName = teacherNames[standards[idx].CourseID+"::"+standards[idx].TeacherID]
	}
}

func (s *CourseService) lookupUserDisplayNames(userIDs []uint64) map[string]string {
	if len(userIDs) == 0 {
		return map[string]string{}
	}

	var users []models.User
	if err := s.db.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return map[string]string{}
	}

	var profiles []models.UserProfile
	_ = s.db.Where("user_id IN ?", userIDs).Find(&profiles).Error

	profileNames := make(map[uint64]string, len(profiles))
	for _, profile := range profiles {
		if nickname := strings.TrimSpace(profile.Nickname); nickname != "" {
			profileNames[profile.UserID] = nickname
		}
	}

	displayNames := make(map[string]string, len(users))
	for _, user := range users {
		name := strings.TrimSpace(profileNames[user.ID])
		if name == "" {
			name = user.Username
		}
		displayNames[strconv.FormatUint(user.ID, 10)] = name
	}
	return displayNames
}

func (s *CourseService) lookupTeacherNamesByKey(keys []string) map[string]string {
	if len(keys) == 0 {
		return map[string]string{}
	}

	var teachers []models.Teacher
	if err := s.db.Find(&teachers).Error; err != nil {
		return map[string]string{}
	}

	keySet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		keySet[key] = struct{}{}
	}

	names := make(map[string]string, len(keys))
	for _, teacher := range teachers {
		key := teacher.CourseID + "::" + teacher.ID
		if _, ok := keySet[key]; ok {
			names[key] = teacher.Name
		}
	}
	return names
}
