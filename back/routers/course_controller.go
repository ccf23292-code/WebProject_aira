package routers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"warehouse-web/middlewares"
	"warehouse-web/models"
	"warehouse-web/services"
	"warehouse-web/utils"
)

// CourseController handles course comment related endpoints.
type CourseController struct {
	service *services.CourseService
}

// NewCourseController creates a CourseController.
func NewCourseController(service *services.CourseService) *CourseController {
	return &CourseController{service: service}
}

// RegisterRoutes registers course comment endpoints (auth required).
//
//	POST /api/courses/:course_id/comments
//	POST /api/courses/:course_id/description-submissions
//	GET /api/courses/:course_id/description-submissions/mine
//	POST /api/courses/:course_id/teachers
//	POST /api/courses/:course_id/teachers/:teacher_id/comments
//	POST /api/courses/:course_id/teachers/:teacher_id/grading-standards
func (ctl *CourseController) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/courses/:course_id/description-submissions", ctl.SubmitDescriptionSubmission)
	group.GET("/courses/:course_id/description-submissions/mine", ctl.ListMyDescriptionSubmissions)
	group.POST("/courses/:course_id/comments", ctl.AddCourseComment)
	group.POST("/courses/:course_id/teachers", ctl.AddTeacher)
	group.POST("/courses/:course_id/teachers/:teacher_id/comments", ctl.AddTeacherComment)
	group.POST("/courses/:course_id/teachers/:teacher_id/grading-standards", ctl.AddGradingStandard)
}

// SubmitDescriptionSubmission creates a pending course description proposal.
func (ctl *CourseController) SubmitDescriptionSubmission(c *gin.Context) {
	courseID := c.Param("course_id")

	var req services.SubmitCourseDescriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	item, err := ctl.service.SubmitCourseDescription(courseID, ctl.currentUserID(c), req.Content)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusCreated, item)
}

// ListMyDescriptionSubmissions returns the current user's submissions for a course.
func (ctl *CourseController) ListMyDescriptionSubmissions(c *gin.Context) {
	courseID := c.Param("course_id")
	items, err := ctl.service.ListMyCourseDescriptionSubmissions(courseID, ctl.currentUserID(c))
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, items)
}

// AddCourseComment creates a course comment.
func (ctl *CourseController) AddCourseComment(c *gin.Context) {
	courseID := c.Param("course_id")

	var req services.AddCourseCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	comment, err := ctl.service.AddCourseComment(courseID, ctl.currentUserID(c), req.Comment)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	comment.AvatarURL = toPublicURL(c, comment.AvatarURL)
	utils.JSONSuccess(c, http.StatusCreated, comment)
}

// AddTeacher creates a teacher entry.
func (ctl *CourseController) AddTeacher(c *gin.Context) {
	courseID := c.Param("course_id")

	var req services.AddTeacherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	teacher, err := ctl.service.AddTeacher(courseID, ctl.currentUserID(c), req)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusCreated, teacher)
}

// AddTeacherComment creates a teacher comment.
func (ctl *CourseController) AddTeacherComment(c *gin.Context) {
	courseID := c.Param("course_id")
	teacherID := c.Param("teacher_id")

	var req services.AddTeacherCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	comment, err := ctl.service.AddTeacherComment(courseID, teacherID, ctl.currentUserID(c), req.Comment)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	comment.AvatarURL = toPublicURL(c, comment.AvatarURL)
	utils.JSONSuccess(c, http.StatusCreated, comment)
}

// AddGradingStandard creates a grading standard.
func (ctl *CourseController) AddGradingStandard(c *gin.Context) {
	courseID := c.Param("course_id")
	teacherID := c.Param("teacher_id")

	var req services.AddGradingStandardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	standard, err := ctl.service.AddGradingStandard(courseID, teacherID, ctl.currentUserID(c), req)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusCreated, standard)
}

// currentUserID gets current user ID from context.
func (ctl *CourseController) currentUserID(c *gin.Context) models.PrimaryKey {
	val, _ := c.Get(middlewares.CtxKeyUserID)
	return val.(models.PrimaryKey)
}

// handleError handles service errors.
func (ctl *CourseController) handleError(c *gin.Context, err error) {
	if svcErr, ok := err.(*services.ServiceError); ok {
		utils.JSONError(c, svcErr.Status, svcErr.Code, svcErr.Message)
		return
	}
	utils.JSONError(c, http.StatusInternalServerError, "internal_error", "服务器开小差了，请稍后重试")
}
