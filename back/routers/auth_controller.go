package routers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"warehouse-web/services"
	"warehouse-web/utils"
)

// AuthController 处理登录 / 注册 / 登出请求。
type AuthController struct {
	service *services.AuthService
}

// NewAuthController 创建 AuthController。
func NewAuthController(service *services.AuthService) *AuthController {
	return &AuthController{service: service}
}

// RegisterRoutes 将认证相关的路由绑定到指定的路由组。
func (ctl *AuthController) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/register", ctl.Register)
	group.POST("/login", ctl.Login)
	group.POST("/logout", ctl.Logout)
}

// Register 处理用户注册请求。
// POST /api/auth/register
func (ctl *AuthController) Register(c *gin.Context) {
	var req services.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}
	resp, err := ctl.service.Register(req)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusCreated, resp)
}

// Login 处理用户登录请求。
// POST /api/auth/login
func (ctl *AuthController) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}
	resp, err := ctl.service.Login(req)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, resp)
}

// Logout 处理用户登出请求。
// POST /api/auth/logout
func (ctl *AuthController) Logout(c *gin.Context) {
	var req services.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}
	resp, err := ctl.service.Logout(req)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, resp)
}

// handleError 统一处理服务层错误，区分业务错误和内部错误。
func (ctl *AuthController) handleError(c *gin.Context, err error) {
	if svcErr, ok := err.(*services.ServiceError); ok {
		utils.JSONError(c, svcErr.Status, svcErr.Code, svcErr.Message)
		return
	}
	utils.JSONError(c, http.StatusInternalServerError, "internal_error", "服务器开小差了，请稍后重试")
}
