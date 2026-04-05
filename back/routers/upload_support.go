package routers

import (
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	defaultStorageRoot = "storage"
	staticURLPrefix    = "/static"
)

var (
	imageFileExts = map[string]struct{}{
		".jpg":  {},
		".jpeg": {},
		".png":  {},
		".gif":  {},
		".webp": {},
		".bmp":  {},
		".svg":  {},
	}
	normalFileExts = map[string]struct{}{
		".jpg":  {},
		".jpeg": {},
		".png":  {},
		".gif":  {},
		".webp": {},
		".bmp":  {},
		".svg":  {},
		".pdf":  {},
		".txt":  {},
		".md":   {},
		".doc":  {},
		".docx": {},
		".xls":  {},
		".xlsx": {},
		".ppt":  {},
		".pptx": {},
		".zip":  {},
		".rar":  {},
		".7z":   {},
	}
)

type uploadConfig struct {
	FormField      string
	Folder         string
	FilenamePrefix string
	MaxSize        int64
	AllowedExts    map[string]struct{}
	StorageRoot    string
}

type uploadedAsset struct {
	OriginalName string `json:"original_name"`
	Filename     string `json:"filename"`
	Size         int64  `json:"size"`
	ContentType  string `json:"content_type"`
	PublicPath   string `json:"public_path"`
	URL          string `json:"url"`
}

type uploadError struct {
	status  int
	code    string
	message string
}

func (e *uploadError) Error() string {
	return e.message
}

func saveUploadedAsset(c *gin.Context, cfg uploadConfig) (*uploadedAsset, error) {
	formField := strings.TrimSpace(cfg.FormField)
	if formField == "" {
		formField = "file"
	}

	file, err := c.FormFile(formField)
	if err != nil {
		return nil, &uploadError{
			status:  http.StatusBadRequest,
			code:    "invalid_request",
			message: fmt.Sprintf("缺少 %s 文件", formField),
		}
	}
	if cfg.MaxSize > 0 && file.Size > cfg.MaxSize {
		return nil, &uploadError{
			status:  http.StatusBadRequest,
			code:    "invalid_request",
			message: fmt.Sprintf("文件大小不能超过 %dMB", cfg.MaxSize/1024/1024),
		}
	}

	ext, err := detectUploadExt(file.Filename, file.Header.Get("Content-Type"))
	if err != nil {
		return nil, &uploadError{
			status:  http.StatusBadRequest,
			code:    "invalid_request",
			message: err.Error(),
		}
	}
	if len(cfg.AllowedExts) > 0 {
		if _, ok := cfg.AllowedExts[ext]; !ok {
			return nil, &uploadError{
				status:  http.StatusBadRequest,
				code:    "invalid_request",
				message: "当前文件类型不支持上传",
			}
		}
	}

	storageRoot := strings.TrimSpace(cfg.StorageRoot)
	if storageRoot == "" {
		storageRoot = defaultStorageRoot
	}
	folder := filepath.Clean(cfg.Folder)
	dateDir := time.Now().Format("20060102")
	targetDir := filepath.Join(storageRoot, folder, dateDir)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, &uploadError{
			status:  http.StatusInternalServerError,
			code:    "internal_error",
			message: "无法创建上传目录",
		}
	}

	filename := buildUploadFilename(cfg.FilenamePrefix, ext)
	targetPath := filepath.Join(targetDir, filename)
	if err := c.SaveUploadedFile(file, targetPath); err != nil {
		return nil, &uploadError{
			status:  http.StatusInternalServerError,
			code:    "internal_error",
			message: "保存上传文件失败",
		}
	}

	publicPath := path.Join(
		staticURLPrefix,
		filepath.ToSlash(folder),
		dateDir,
		filename,
	)

	return &uploadedAsset{
		OriginalName: file.Filename,
		Filename:     filename,
		Size:         file.Size,
		ContentType:  strings.TrimSpace(file.Header.Get("Content-Type")),
		PublicPath:   publicPath,
		URL:          buildPublicURL(c, publicPath),
	}, nil
}

func buildUploadFilename(prefix, ext string) string {
	namePrefix := strings.TrimSpace(prefix)
	if namePrefix == "" {
		namePrefix = "file"
	}
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	return strings.ToLower(fmt.Sprintf("%s_%d_%s%s", namePrefix, time.Now().UnixNano(), suffix, ext))
}

func detectUploadExt(filename, contentType string) (string, error) {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(filename)))
	if ext != "" {
		return ext, nil
	}

	contentType = strings.TrimSpace(contentType)
	if contentType != "" {
		if exts, err := mime.ExtensionsByType(contentType); err == nil && len(exts) > 0 {
			return strings.ToLower(exts[0]), nil
		}
	}
	return "", fmt.Errorf("文件缺少扩展名，请使用带后缀的文件名")
}

func buildPublicURL(c *gin.Context, publicPath string) string {
	publicPath = ensureLeadingSlash(strings.TrimSpace(publicPath))
	if publicPath == "" {
		return ""
	}
	if isAbsoluteURL(publicPath) {
		return publicPath
	}

	if baseURL := strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")); baseURL != "" {
		if joined := joinBaseURL(baseURL, publicPath); joined != "" {
			return joined
		}
	}

	host := firstForwardedValue(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(c.Request.Host)
	}
	if host == "" {
		return publicPath
	}

	scheme := "http"
	if proto := firstForwardedValue(c.GetHeader("X-Forwarded-Proto")); proto != "" {
		scheme = proto
	} else if c.Request.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, host, publicPath)
}

func toPublicURL(c *gin.Context, filePath string) string {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return ""
	}
	if isAbsoluteURL(filePath) {
		return filePath
	}
	return buildPublicURL(c, filePath)
}

func ensureLeadingSlash(value string) string {
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "/") {
		return value
	}
	return "/" + value
}

func isAbsoluteURL(value string) bool {
	lowerValue := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(lowerValue, "http://") || strings.HasPrefix(lowerValue, "https://")
}

func firstForwardedValue(value string) string {
	parts := strings.Split(value, ",")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func joinBaseURL(baseURL, publicPath string) string {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	parsed.Path = path.Join(parsed.Path, publicPath)
	return parsed.String()
}
