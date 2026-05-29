package routers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"warehouse-web/middlewares"
	"warehouse-web/models"
	"warehouse-web/services"
	"warehouse-web/utils"
)

// AttemptController 处理"严格练习"模式下的 attempt / submission 接口。
type AttemptController struct {
	service *services.AttemptService
}

// NewAttemptController 创建 AttemptController。
func NewAttemptController(service *services.AttemptService) *AttemptController {
	return &AttemptController{service: service}
}

// RegisterRoutes 将 attempt 相关路由绑定到指定的路由组（该组应已挂载 AuthRequired 中间件）。
//
//	POST /api/papers/:paper_id/attempts
//	GET  /api/attempts/:attempt_id
//	POST /api/attempts/:attempt_id/problems/:problem_id/submit
func (ctl *AttemptController) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/papers/:paper_id/attempts", ctl.CreateAttempt)
	group.GET("/attempts/:attempt_id", ctl.GetAttempt)
	group.POST("/attempts/:attempt_id/problems/:problem_id/submit", ctl.SubmitProblem)
}

// createAttemptRequest 是 CreateAttempt 接口的请求体。
// body 可省略；缺省时 force_reset=false，即"恢复进度"语义。
type createAttemptRequest struct {
	ForceReset bool `json:"force_reset"`
}

// CreateAttempt 获取或创建一次尝试（get-or-create）。
//
//	默认：复用现有 in_progress（恢复进度）；若无则创建新的
//	force_reset=true：abandon 旧 in_progress + 创建新的
func (ctl *AttemptController) CreateAttempt(c *gin.Context) {
	paperID, err := strconv.ParseUint(c.Param("paper_id"), 10, 64)
	if err != nil || paperID == 0 {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "paper_id 必须为正整数")
		return
	}

	var req createAttemptRequest
	// body 可选：空 / 非 JSON 都视为 ForceReset=false，不阻塞调用
	_ = c.ShouldBindJSON(&req)

	userID := ctl.currentUserID(c)
	result, err := ctl.service.CreateAttempt(userID, paperID, req.ForceReset)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusCreated, result)
}

// GetAttempt 返回某次尝试的聚合状态 + 所有已提交记录。
// 用于前端"恢复现场"。
func (ctl *AttemptController) GetAttempt(c *gin.Context) {
	attemptID, err := strconv.ParseUint(c.Param("attempt_id"), 10, 64)
	if err != nil || attemptID == 0 {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "attempt_id 必须为正整数")
		return
	}

	userID := ctl.currentUserID(c)
	detail, err := ctl.service.GetAttempt(userID, attemptID)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, detail)
}

type submitProblemRequest struct {
	UserAnswer string `json:"user_answer"`
}

// SubmitProblem 提交单题答案。
// 重复提交（违反唯一索引）返回 409 already_submitted。
func (ctl *AttemptController) SubmitProblem(c *gin.Context) {
	attemptID, err := strconv.ParseUint(c.Param("attempt_id"), 10, 64)
	if err != nil || attemptID == 0 {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "attempt_id 必须为正整数")
		return
	}
	problemID, err := strconv.ParseUint(c.Param("problem_id"), 10, 64)
	if err != nil || problemID == 0 {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "problem_id 必须为正整数")
		return
	}

	var req submitProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	userID := ctl.currentUserID(c)
	result, err := ctl.service.SubmitProblem(userID, attemptID, problemID, req.UserAnswer)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, result)
}

// currentUserID 从 gin.Context 中取出当前登录用户的 ID。
func (ctl *AttemptController) currentUserID(c *gin.Context) models.PrimaryKey {
	val, _ := c.Get(middlewares.CtxKeyUserID)
	return val.(models.PrimaryKey)
}

// handleError 统一处理服务层错误。
func (ctl *AttemptController) handleError(c *gin.Context, err error) {
	if svcErr, ok := err.(*services.ServiceError); ok {
		utils.JSONError(c, svcErr.Status, svcErr.Code, svcErr.Message)
		return
	}
	utils.JSONError(c, http.StatusInternalServerError, "internal_error", "服务器开小差了，请稍后重试")
}
