package routers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"warehouse-web/services"
	"warehouse-web/utils"
)

// PaperController 处理课程 / 试卷 / 题目的浏览请求。
type PaperController struct {
	service     *services.PaperService
	courseSvc   *services.CourseService
}

// NewPaperController 创建 PaperController。
func NewPaperController(service *services.PaperService, courseSvc *services.CourseService) *PaperController {
	return &PaperController{service: service, courseSvc: courseSvc}
}

// RegisterRoutes 将浏览相关的路由绑定到指定的路由组。
//
//	GET /api/courses
//	GET /api/courses/:course_id
//	GET /api/courses/:course_id/papers
//	GET /api/papers/:paper_id/problems
func (ctl *PaperController) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/courses", ctl.ListCourses)
	group.GET("/courses/:course_id", ctl.GetCourse)
	group.GET("/courses/:course_id/papers", ctl.ListPapers)
	group.GET("/papers/:paper_id/problems", ctl.ListProblems)
}

// ListCourses 返回所有课程列表。
// GET /api/courses
func (ctl *PaperController) ListCourses(c *gin.Context) {
	query := c.Query("q")
	courses, err := ctl.courseSvc.ListCourses(query)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, courses)
}

// GetCourse 返回课程详情。
// GET /api/courses/:course_id
func (ctl *PaperController) GetCourse(c *gin.Context) {
	courseID := c.Param("course_id")
	course, err := ctl.courseSvc.GetCourse(courseID)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, course)
}

// ListPapers 返回指定课程下的试卷列表。
// GET /api/courses/:course_id/papers
func (ctl *PaperController) ListPapers(c *gin.Context) {
	courseID := c.Param("course_id")
	if courseID == "" {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "course_id 不能为空")
		return
	}

	papers := ctl.service.ListPapersByCourse(courseID)
	utils.JSONSuccess(c, http.StatusOK, papers)
}

// ListProblems 返回指定试卷下的题目列表。
// GET /api/papers/:paper_id/problems
func (ctl *PaperController) ListProblems(c *gin.Context) {
	paperID, err := strconv.ParseUint(c.Param("paper_id"), 10, 64)
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "paper_id 必须为正整数")
		return
	}

	problems := ctl.service.ListProblemsByPaper(paperID)
	utils.JSONSuccess(c, http.StatusOK, problems)
}

// handleError 统一处理服务层错误。
func (ctl *PaperController) handleError(c *gin.Context, err error) {
	if svcErr, ok := err.(*services.ServiceError); ok {
		utils.JSONError(c, svcErr.Status, svcErr.Code, svcErr.Message)
		return
	}
	utils.JSONError(c, http.StatusInternalServerError, "internal_error", "服务器开小差了，请稍后重试")
}
