package routers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"warehouse-web/services"
	"warehouse-web/utils"
)

// AdminController 处理管理员的试卷/题目上传与修改请求。
type AdminController struct {
	service *services.PaperService
}

// NewAdminController 创建 AdminController。
func NewAdminController(service *services.PaperService) *AdminController {
	return &AdminController{service: service}
}

// RegisterRoutes 将管理员相关的路由绑定到指定的路由组（该组应已挂载 AuthRequired + AdminRequired 中间件）。
//
//	POST   /api/admin/papers
//	PUT    /api/admin/papers/:paper_id
//	DELETE /api/admin/papers/:paper_id
//	PUT    /api/admin/problems/:problem_id
func (ctl *AdminController) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/papers", ctl.CreatePaper)
	group.PUT("/papers/:paper_id", ctl.UpdatePaper)
	group.DELETE("/papers/:paper_id", ctl.DeletePaper)
	group.PUT("/problems/:problem_id", ctl.UpdateProblem)
}

// CreatePaper 创建新试卷。
// POST /api/admin/papers
func (ctl *AdminController) CreatePaper(c *gin.Context) {
	var req services.CreatePaperRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	paper, err := ctl.service.CreatePaper(req)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusCreated, paper)
}

// UpdatePaper 更新指定试卷。
// PUT /api/admin/papers/:paper_id
func (ctl *AdminController) UpdatePaper(c *gin.Context) {
	paperID, err := strconv.ParseUint(c.Param("paper_id"), 10, 64)
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "paper_id 必须为正整数")
		return
	}

	var req services.UpdatePaperRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	paper, svcErr := ctl.service.UpdatePaper(paperID, req)
	if svcErr != nil {
		ctl.handleError(c, svcErr)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, paper)
}

// DeletePaper 删除指定试卷及其题目。
// DELETE /api/admin/papers/:paper_id
func (ctl *AdminController) DeletePaper(c *gin.Context) {
	paperID, err := strconv.ParseUint(c.Param("paper_id"), 10, 64)
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "paper_id 必须为正整数")
		return
	}

	if svcErr := ctl.service.DeletePaper(paperID); svcErr != nil {
		ctl.handleError(c, svcErr)
		return
	}
	utils.JSONSuccessMsg(c, http.StatusOK, "删除成功", nil)
}

// UpdateProblem 更新指定题目。
// PUT /api/admin/problems/:problem_id
func (ctl *AdminController) UpdateProblem(c *gin.Context) {
	problemID, err := strconv.ParseUint(c.Param("problem_id"), 10, 64)
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "problem_id 必须为正整数")
		return
	}

	var req services.UpdateProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	problem, svcErr := ctl.service.UpdateProblem(problemID, req)
	if svcErr != nil {
		ctl.handleError(c, svcErr)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, problem)
}

// handleError 统一处理服务层错误。
func (ctl *AdminController) handleError(c *gin.Context, err error) {
	if svcErr, ok := err.(*services.ServiceError); ok {
		utils.JSONError(c, svcErr.Status, svcErr.Code, svcErr.Message)
		return
	}
	utils.JSONError(c, http.StatusInternalServerError, "internal_error", "服务器开小差了，请稍后重试")
}
