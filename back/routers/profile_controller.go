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

// ProfileController 处理用户资料接口。
type ProfileController struct {
	service *services.ProfileService
}

func NewProfileController(service *services.ProfileService) *ProfileController {
	return &ProfileController{service: service}
}

// RegisterRoutes 注册用户资料相关路由。
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
	profile.AvatarURL = toPublicURL(c, profile.AvatarURL)
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

	profile.AvatarURL = toPublicURL(c, profile.AvatarURL)
	if profile.AvatarURL != "" {
		c.SetCookie("avatarUrl", profile.AvatarURL, 3600*24*30, "/", "", false, false)
	}
	utils.JSONSuccess(c, http.StatusOK, profile)
}

func (ctl *ProfileController) UploadAvatar(c *gin.Context) {
	userID := ctl.currentUserID(c)

	asset, err := saveUploadedAsset(c, uploadConfig{
		FormField:      "avatar",
		Folder:         "avatars",
		FilenamePrefix: "u" + strconv.FormatUint(uint64(userID), 10),
		MaxSize:        5 * 1024 * 1024,
		AllowedExts:    imageFileExts,
		StorageRoot:    defaultStorageRoot,
	})
	if err != nil {
		if upErr, ok := err.(*uploadError); ok {
			utils.JSONError(c, upErr.status, upErr.code, upErr.message)
			return
		}
		utils.JSONError(c, http.StatusInternalServerError, "internal_error", "保存头像失败")
		return
	}

	profile, err := ctl.service.UpdateAvatar(userID, asset.PublicPath)
	if err != nil {
		ctl.handleError(c, err)
		return
	}

	profile.AvatarURL = asset.URL
	c.SetCookie("avatarUrl", asset.URL, 3600*24*30, "/", "", false, false)
	utils.JSONSuccess(c, http.StatusOK, profile)
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
