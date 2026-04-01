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

// FavoriteController 处理收藏相关请求（需登录）。
type FavoriteController struct {
	service *services.FavoriteService
}

// NewFavoriteController 创建 FavoriteController。
func NewFavoriteController(service *services.FavoriteService) *FavoriteController {
	return &FavoriteController{service: service}
}

// RegisterRoutes 将收藏相关的路由绑定到指定的路由组（该组应已挂载 AuthRequired 中间件）。
//
//	GET    /api/favorites
//	GET    /api/favorites/ids
//	POST   /api/favorites
//	DELETE /api/favorites/:problem_id
func (ctl *FavoriteController) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("", ctl.List)
	group.GET("/ids", ctl.ListIDs)
	group.POST("", ctl.Add)
	group.DELETE("/:problem_id", ctl.Remove)
}

// addFavoriteRequest 添加收藏请求体。
type addFavoriteRequest struct {
	ProblemID uint64 `json:"problem_id" binding:"required"`
}

// List 返回当前用户的收藏列表（支持分页）。
// GET /api/favorites?page=1&size=10
func (ctl *FavoriteController) List(c *gin.Context) {
	userID := ctl.currentUserID(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}

	result, err := ctl.service.ListFavorites(userID, page, size)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, result)
}

// ListIDs 返回收藏题目 ID 列表。
// GET /api/favorites/ids
func (ctl *FavoriteController) ListIDs(c *gin.Context) {
	userID := ctl.currentUserID(c)
	ids, err := ctl.service.ListFavoriteIDs(userID)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, ids)
}

// Add 添加一条收藏。
// POST /api/favorites  { "problem_id": 1001 }
func (ctl *FavoriteController) Add(c *gin.Context) {
	var req addFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	userID := ctl.currentUserID(c)
	if err := ctl.service.AddFavorite(userID, req.ProblemID); err != nil {
		ctl.handleError(c, err)
		return
	}

	utils.JSONSuccessMsg(c, http.StatusOK, "收藏成功", nil)
}

// Remove 取消收藏。
// DELETE /api/favorites/:problem_id
func (ctl *FavoriteController) Remove(c *gin.Context) {
	problemID, err := strconv.ParseUint(c.Param("problem_id"), 10, 64)
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "problem_id 必须为正整数")
		return
	}

	userID := ctl.currentUserID(c)
	if err := ctl.service.RemoveFavorite(userID, problemID); err != nil {
		ctl.handleError(c, err)
		return
	}

	utils.JSONSuccessMsg(c, http.StatusOK, "取消收藏成功", nil)
}

// currentUserID 从 gin.Context 中取出当前登录用户的 ID。
func (ctl *FavoriteController) currentUserID(c *gin.Context) models.PrimaryKey {
	val, _ := c.Get(middlewares.CtxKeyUserID)
	return val.(models.PrimaryKey)
}

// handleError 统一处理服务层错误。
func (ctl *FavoriteController) handleError(c *gin.Context, err error) {
	if svcErr, ok := err.(*services.ServiceError); ok {
		utils.JSONError(c, svcErr.Status, svcErr.Code, svcErr.Message)
		return
	}
	utils.JSONError(c, http.StatusInternalServerError, "internal_error", "服务器开小差了，请稍后重试")
}
