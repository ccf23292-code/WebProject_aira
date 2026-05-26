package routers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"warehouse-web/middlewares"
	"warehouse-web/models"
	"warehouse-web/services"
	"warehouse-web/utils"
)

// CheckinController 处理每日签到相关请求（需登录）。
type CheckinController struct {
	service *services.CheckinService
}

// NewCheckinController 创建 CheckinController。
func NewCheckinController(service *services.CheckinService) *CheckinController {
	return &CheckinController{service: service}
}

// RegisterRoutes 将签到相关的路由绑定到指定的路由组（该组应已挂载 AuthRequired 中间件）。
//
//	GET  /api/checkin/today
//	POST /api/checkin
func (ctl *CheckinController) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/today", ctl.GetToday)
	group.POST("", ctl.CheckIn)
}

// GetToday 返回当前用户的签到状态。
// GET /api/checkin/today
func (ctl *CheckinController) GetToday(c *gin.Context) {
	userID := ctl.currentUserID(c)
	status, err := ctl.service.GetTodayStatus(userID)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, status)
}

// CheckIn 提交一次签到。
// POST /api/checkin
// 重复签到时返回 409，前端据此做"今日已签到"灰态展示。
func (ctl *CheckinController) CheckIn(c *gin.Context) {
	userID := ctl.currentUserID(c)
	status, err := ctl.service.CheckIn(userID)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, status)
}

// currentUserID 从 gin.Context 中取出当前登录用户的 ID。
func (ctl *CheckinController) currentUserID(c *gin.Context) models.PrimaryKey {
	val, _ := c.Get(middlewares.CtxKeyUserID)
	return val.(models.PrimaryKey)
}

// handleError 统一处理服务层错误。
func (ctl *CheckinController) handleError(c *gin.Context, err error) {
	if svcErr, ok := err.(*services.ServiceError); ok {
		utils.JSONError(c, svcErr.Status, svcErr.Code, svcErr.Message)
		return
	}
	utils.JSONError(c, http.StatusInternalServerError, "internal_error", "服务器开小差了，请稍后重试")
}
