package routers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"warehouse-web/utils"
)

// FileController handles generic file uploads.
type FileController struct{}

func NewFileController() *FileController {
	return &FileController{}
}

// RegisterRoutes registers generic file upload routes.
//
//	POST /api/files/upload
func (ctl *FileController) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/upload", ctl.Upload)
}

// Upload stores a multipart file under storage and returns both public_path and url.
func (ctl *FileController) Upload(c *gin.Context) {
	uploadType := strings.ToLower(strings.TrimSpace(c.DefaultPostForm("type", "image")))

	cfg := uploadConfig{
		FormField:   "file",
		MaxSize:     20 * 1024 * 1024,
		StorageRoot: defaultStorageRoot,
	}

	switch uploadType {
	case "image", "images":
		uploadType = "image"
		cfg.Folder = "uploads/images"
		cfg.FilenamePrefix = "img"
		cfg.AllowedExts = imageFileExts
	case "file", "files":
		uploadType = "file"
		cfg.Folder = "uploads/files"
		cfg.FilenamePrefix = "file"
		cfg.AllowedExts = normalFileExts
	default:
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "type 仅支持 image 或 file")
		return
	}

	asset, err := saveUploadedAsset(c, cfg)
	if err != nil {
		ctl.handleUploadError(c, err)
		return
	}

	utils.JSONSuccess(c, http.StatusCreated, gin.H{
		"type":          uploadType,
		"original_name": asset.OriginalName,
		"filename":      asset.Filename,
		"size":          asset.Size,
		"content_type":  asset.ContentType,
		"public_path":   asset.PublicPath,
		"url":           asset.URL,
	})
}

func (ctl *FileController) handleUploadError(c *gin.Context, err error) {
	if upErr, ok := err.(*uploadError); ok {
		utils.JSONError(c, upErr.status, upErr.code, upErr.message)
		return
	}
	utils.JSONError(c, http.StatusInternalServerError, "internal_error", "服务器开小差了，请稍后重试")
}
