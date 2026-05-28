package routers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"warehouse-web/middlewares"
	"warehouse-web/services"
	"warehouse-web/utils"
)

type HomepageController struct {
	service *services.HomepageService
}

type AddHomepageMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

func NewHomepageController(service *services.HomepageService) *HomepageController {
	return &HomepageController{service: service}
}

func (ctl *HomepageController) RegisterPublicRoutes(group *gin.RouterGroup) {
	group.GET("/homepage/messages", ctl.ListMessages)
}

func (ctl *HomepageController) RegisterProtectedRoutes(group *gin.RouterGroup) {
	group.POST("/homepage/messages", ctl.AddMessage)
	group.PUT("/homepage/messages/:id", ctl.UpdateMessage)
	group.DELETE("/homepage/messages/:id", ctl.DeleteMessage)
}

func (ctl *HomepageController) ListMessages(c *gin.Context) {
	items, err := ctl.service.ListMessages()
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	for i := range items {
		items[i].AvatarURL = toPublicURL(c, items[i].AvatarURL)
	}
	utils.JSONSuccess(c, http.StatusOK, items)
}

func (ctl *HomepageController) AddMessage(c *gin.Context) {
	var req AddHomepageMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	item, err := ctl.service.AddMessage(c.GetUint64(middlewares.CtxKeyUserID), req.Content)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	item.AvatarURL = toPublicURL(c, item.AvatarURL)
	utils.JSONSuccess(c, http.StatusCreated, item)
}

func (ctl *HomepageController) UpdateMessage(c *gin.Context) {
	messageID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "id 必须为正整数")
		return
	}

	var req AddHomepageMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	item, svcErr := ctl.service.UpdateMessage(c.GetUint64(middlewares.CtxKeyUserID), messageID, req.Content)
	if svcErr != nil {
		ctl.handleError(c, svcErr)
		return
	}
	item.AvatarURL = toPublicURL(c, item.AvatarURL)
	utils.JSONSuccess(c, http.StatusOK, item)
}

func (ctl *HomepageController) DeleteMessage(c *gin.Context) {
	messageID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "id 必须为正整数")
		return
	}

	if svcErr := ctl.service.DeleteMessage(c.GetUint64(middlewares.CtxKeyUserID), messageID); svcErr != nil {
		ctl.handleError(c, svcErr)
		return
	}
	utils.JSONSuccessMsg(c, http.StatusOK, "删除成功", nil)
}

func (ctl *HomepageController) handleError(c *gin.Context, err error) {
	if svcErr, ok := err.(*services.ServiceError); ok {
		utils.JSONError(c, svcErr.Status, svcErr.Code, svcErr.Message)
		return
	}
	utils.JSONError(c, http.StatusInternalServerError, "internal_error", "服务器开小差了，请稍后重试")
}
