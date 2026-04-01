package routers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"warehouse-web/middlewares"
	"warehouse-web/models"
	"warehouse-web/services"
	"warehouse-web/utils"
)

// ProfileController 处理用户资料。
type ProfileController struct {
	service *services.ProfileService
}

func NewProfileController(service *services.ProfileService) *ProfileController {
	return &ProfileController{service: service}
}

// RegisterRoutes 注册用户资料路由。
// GET  /api/profile
// PUT  /api/profile
// POST /api/profile/avatar
func (ctl *ProfileController) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("", ctl.Get)
	group.PUT("", ctl.Update)
	group.POST("/avatar", ctl.UploadAvatar)
}

func (ctl *ProfileController) Get(c *gin.Context) {
	userID := ctl.currentUserID(c)
	profile, err := ctl.service.GetProfile(userID)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, profile)
}

func (ctl *ProfileController) Update(c *gin.Context) {
	userID := ctl.currentUserID(c)
	var req services.ProfileUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}
	profile, err := ctl.service.UpdateProfile(userID, req)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	if profile.AvatarURL != "" {
		c.SetCookie("avatarUrl", profile.AvatarURL, 3600*24*30, "/", "", false, false)
	}
	utils.JSONSuccess(c, http.StatusOK, profile)
}

func (ctl *ProfileController) UploadAvatar(c *gin.Context) {
	userID := ctl.currentUserID(c)

	file, err := c.FormFile("avatar")
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "缺少头像文件")
		return
	}
	if file.Size > 5*1024*1024 {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "头像大小不能超过 5MB")
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		ext = ".png"
	}

	baseDir := filepath.Join("storage", "avatars")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		utils.JSONError(c, http.StatusInternalServerError, "internal_error", "无法创建头像目录")
		return
	}

	filename := fmtAvatarFilename(userID, ext)
	path := filepath.Join(baseDir, filename)
	if err := c.SaveUploadedFile(file, path); err != nil {
		utils.JSONError(c, http.StatusInternalServerError, "internal_error", "保存头像失败")
		return
	}

	url := "/static/avatars/" + filename
	profile, err := ctl.service.UpdateAvatar(userID, url)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	c.SetCookie("avatarUrl", url, 3600*24*30, "/", "", false, false)
	utils.JSONSuccess(c, http.StatusOK, profile)
}

func fmtAvatarFilename(userID uint64, ext string) string {
	stamp := time.Now().Unix()
	return strings.ToLower(
		fmt.Sprintf("u%d_%d%s", userID, stamp, ext),
	)
}

func (ctl *ProfileController) currentUserID(c *gin.Context) models.PrimaryKey {
	val, _ := c.Get(middlewares.CtxKeyUserID)
	return val.(models.PrimaryKey)
}

func (ctl *ProfileController) handleError(c *gin.Context, err error) {
	if svcErr, ok := err.(*services.ServiceError); ok {
		utils.JSONError(c, svcErr.Status, svcErr.Code, svcErr.Message)
		return
	}
	utils.JSONError(c, http.StatusInternalServerError, "internal_error", "服务器开小差了，请稍后重试")
}
