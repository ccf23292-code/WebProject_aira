package routers

import (
	"net/http"

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
}

func (ctl *HomepageController) ListMessages(c *gin.Context) {
	items, err := ctl.service.ListMessages()
	if err != nil {
		ctl.handleError(c, err)
		return
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
	utils.JSONSuccess(c, http.StatusCreated, item)
}

func (ctl *HomepageController) handleError(c *gin.Context, err error) {
	if svcErr, ok := err.(*services.ServiceError); ok {
		utils.JSONError(c, svcErr.Status, svcErr.Code, svcErr.Message)
		return
	}
	utils.JSONError(c, http.StatusInternalServerError, "internal_error", "服务器开小差了，请稍后重试")
}
