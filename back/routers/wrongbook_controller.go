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

// WrongBookController 处理错题本接口。
type WrongBookController struct {
	service *services.WrongBookService
}

func NewWrongBookController(service *services.WrongBookService) *WrongBookController {
	return &WrongBookController{service: service}
}

// RegisterRoutes 注册错题本路由。
// GET    /api/wrongbook
// PATCH  /api/wrongbook/:problem_id
// DELETE /api/wrongbook/:problem_id
// DELETE /api/wrongbook/trash
func (ctl *WrongBookController) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("", ctl.List)
	group.PATCH("/:problem_id", ctl.Update)
	group.DELETE("/:problem_id", ctl.Delete)
	group.DELETE("/trash", ctl.ClearTrash)
}

// List 返回错题本。
func (ctl *WrongBookController) List(c *gin.Context) {
	userID := ctl.currentUserID(c)
	status := c.Query("status")
	resp, err := ctl.service.ListWrongBook(userID, status)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, resp)
}

// Update 更新错题备注/状态。
func (ctl *WrongBookController) Update(c *gin.Context) {
	userID := ctl.currentUserID(c)
	problemID, err := strconv.ParseUint(c.Param("problem_id"), 10, 64)
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "problem_id 必须为正整数")
		return
	}
	var req struct {
		Note   *string `json:"note"`
		Status *string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	updated, err := ctl.service.UpdateWrongQuestion(userID, problemID, req.Note, req.Status)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, updated)
}

// Delete 删除单条错题。
func (ctl *WrongBookController) Delete(c *gin.Context) {
	userID := ctl.currentUserID(c)
	problemID, err := strconv.ParseUint(c.Param("problem_id"), 10, 64)
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "problem_id 必须为正整数")
		return
	}
	if err := ctl.service.RemoveWrongQuestion(userID, problemID); err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccessMsg(c, http.StatusOK, "deleted", nil)
}

// ClearTrash 清空垃圾篓。
func (ctl *WrongBookController) ClearTrash(c *gin.Context) {
	userID := ctl.currentUserID(c)
	if err := ctl.service.ClearTrash(userID); err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccessMsg(c, http.StatusOK, "cleared", nil)
}

func (ctl *WrongBookController) currentUserID(c *gin.Context) models.PrimaryKey {
	val, _ := c.Get(middlewares.CtxKeyUserID)
	return val.(models.PrimaryKey)
}

func (ctl *WrongBookController) handleError(c *gin.Context, err error) {
	if svcErr, ok := err.(*services.ServiceError); ok {
		utils.JSONError(c, svcErr.Status, svcErr.Code, svcErr.Message)
		return
	}
	utils.JSONError(c, http.StatusInternalServerError, "internal_error", "服务器开小差了，请稍后重试")
}
