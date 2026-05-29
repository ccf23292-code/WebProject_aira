package routers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"warehouse-web/middlewares"
	"warehouse-web/models"
	"warehouse-web/services"
	"warehouse-web/utils"
)

// IngestController 处理"用户上传文件 → LLM 清洗 → 审核 → 入库"的全部 HTTP 端点。
type IngestController struct {
	service *services.IngestService
}

func NewIngestController(service *services.IngestService) *IngestController {
	return &IngestController{service: service}
}

// 允许上传的扩展名集合（与前端 accept 属性保持同步）。
var ingestAllowedExts = map[string]struct{}{
	".md": {}, ".markdown": {}, ".txt": {},
	".pdf":  {},
	".docx": {},
	".jpg":  {}, ".jpeg": {}, ".png": {},
}

// RegisterUserRoutes 挂载普通用户路由。
//
//	POST /api/ingest/upload   multipart
//	GET  /api/ingest/my
//	GET  /api/ingest/:id
func (ctl *IngestController) RegisterUserRoutes(group *gin.RouterGroup) {
	group.POST("/upload", ctl.Upload)
	group.GET("/my", ctl.ListMy)
	group.GET("/:id", ctl.GetOne)
}

// RegisterAdminRoutes 挂载管理员路由。
//
//	GET   /api/admin/ingest
//	GET   /api/admin/ingest/:id
//	PATCH /api/admin/ingest/:id
//	POST  /api/admin/ingest/:id/publish
//	POST  /api/admin/ingest/:id/reject
func (ctl *IngestController) RegisterAdminRoutes(group *gin.RouterGroup) {
	group.GET("/ingest", ctl.AdminList)
	group.GET("/ingest/:id", ctl.GetOne)
	group.PATCH("/ingest/:id", ctl.AdminUpdate)
	group.POST("/ingest/:id/publish", ctl.AdminPublish)
	group.POST("/ingest/:id/reject", ctl.AdminReject)
}

// ─────────────────────── handlers ───────────────────────

// Upload 接收 multipart 上传并创建一条 IngestJob。
//
// form 字段：
//
//	kind                question | explanation
//	course_id           已有课程 ID（与 new_course_name 二选一）
//	new_course_name     新课程名（与 course_id 二选一）
//	paper_name          题目流程必填
//	target_paper_id     题解流程必填
//	file                文件本体
func (ctl *IngestController) Upload(c *gin.Context) {
	cfg := uploadConfig{
		FormField:      "file",
		Folder:         "uploads/ingest",
		FilenamePrefix: "ingest",
		MaxSize:        20 * 1024 * 1024,
		AllowedExts:    ingestAllowedExts,
		StorageRoot:    defaultStorageRoot,
	}
	asset, err := saveUploadedAsset(c, cfg)
	if err != nil {
		ctl.handleUploadError(c, err)
		return
	}

	// 重组绝对路径方便后续 worker 读取（saveUploadedAsset 写到 storage/<folder>/<date>/<filename>）。
	// asset.PublicPath 形如 /static/uploads/ingest/20260528/ingest_xxx.pdf，去掉 /static 前缀就是磁盘相对路径。
	storagePath := strings.TrimPrefix(asset.PublicPath, "/static/")
	storagePath = "storage/" + storagePath

	kind := strings.TrimSpace(c.PostForm("kind"))
	courseID := strings.TrimSpace(c.PostForm("course_id"))
	newCourseName := strings.TrimSpace(c.PostForm("new_course_name"))
	paperName := strings.TrimSpace(c.PostForm("paper_name"))
	semester := strings.TrimSpace(c.PostForm("semester"))
	examType := strings.TrimSpace(c.PostForm("exam_type"))

	var year int
	if raw := strings.TrimSpace(c.PostForm("year")); raw != "" {
		v, parseErr := strconv.Atoi(raw)
		if parseErr != nil || v < 1900 || v > 2200 {
			utils.JSONError(c, http.StatusBadRequest, "invalid_request", "year 必须是合理的年份数字")
			return
		}
		year = v
	}

	var targetPaperID *uint64
	if raw := strings.TrimSpace(c.PostForm("target_paper_id")); raw != "" {
		v, parseErr := strconv.ParseUint(raw, 10, 64)
		if parseErr != nil || v == 0 {
			utils.JSONError(c, http.StatusBadRequest, "invalid_request", "target_paper_id 必须为正整数")
			return
		}
		targetPaperID = &v
	}

	userID := ctl.currentUserID(c)
	job, svcErr := ctl.service.CreateJob(services.CreateJobInput{
		UserID:        userID,
		Kind:          kind,
		CourseID:      courseID,
		NewCourseName: newCourseName,
		PaperName:     paperName,
		Year:          year,
		Semester:      semester,
		ExamType:      examType,
		TargetPaperID: targetPaperID,
		Filename:      asset.OriginalName,
		StoragePath:   storagePath,
		Mime:          asset.ContentType,
		Size:          asset.Size,
	})
	if svcErr != nil {
		ctl.handleError(c, svcErr)
		return
	}

	utils.JSONSuccess(c, http.StatusCreated, job)
}

func (ctl *IngestController) ListMy(c *gin.Context) {
	userID := ctl.currentUserID(c)
	status := strings.TrimSpace(c.Query("status"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	jobs, total, err := ctl.service.ListMyJobs(userID, status, page, size)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, gin.H{
		"total": total,
		"page":  page,
		"size":  size,
		"items": jobs,
	})
}

// GetOne 返回单条任务详情。
//
// 权限：本人可见；admin 可见所有。
// 该端点同时挂在 /api/ingest/:id 和 /api/admin/ingest/:id 两条路由上，
// 后者已被 AdminRequired 拦截，无需再判断角色；前者需要判断"是否本人或 admin"。
func (ctl *IngestController) GetOne(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "id 必须为正整数")
		return
	}
	job, svcErr := ctl.service.GetJob(id)
	if svcErr != nil {
		ctl.handleError(c, svcErr)
		return
	}

	// 鉴权：admin 一律可见；否则必须本人
	role, _ := c.Get(middlewares.CtxKeyRole)
	if models.Role(role.(string)) != models.RoleAdmin {
		if job.UserID != ctl.currentUserID(c) {
			utils.JSONError(c, http.StatusForbidden, "forbidden", "无权查看该任务")
			return
		}
	}

	utils.JSONSuccess(c, http.StatusOK, job)
}

func (ctl *IngestController) AdminList(c *gin.Context) {
	status := strings.TrimSpace(c.Query("status"))
	if status == "" {
		status = models.IngestStatusAwaitingReview
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	jobs, total, err := ctl.service.AdminListJobs(status, page, size)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, gin.H{
		"total": total,
		"page":  page,
		"size":  size,
		"items": jobs,
	})
}

// adminUpdateRequest 是 PATCH /api/admin/ingest/:id 的请求体。
// 字段都用指针，允许只更新部分字段；ParsedJSON 用 RawMessage 透传给 service。
type adminUpdateRequest struct {
	CourseID      *string          `json:"course_id"`
	NewCourseName *string          `json:"new_course_name"`
	PaperName     *string          `json:"paper_name"`
	Year          *int             `json:"year"`
	Semester      *string          `json:"semester"`
	ExamType      *string          `json:"exam_type"`
	TargetPaperID *uint64          `json:"target_paper_id"`
	ParsedJSON    *json.RawMessage `json:"parsed_json"`
}

func (ctl *IngestController) AdminUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "id 必须为正整数")
		return
	}

	var req adminUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	job, svcErr := ctl.service.AdminUpdateJob(id, services.AdminUpdateInput{
		CourseID:      req.CourseID,
		NewCourseName: req.NewCourseName,
		PaperName:     req.PaperName,
		Year:          req.Year,
		Semester:      req.Semester,
		ExamType:      req.ExamType,
		TargetPaperID: req.TargetPaperID,
		ParsedJSON:    req.ParsedJSON,
	})
	if svcErr != nil {
		ctl.handleError(c, svcErr)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, job)
}

func (ctl *IngestController) AdminPublish(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "id 必须为正整数")
		return
	}
	reviewerID := ctl.currentUserID(c)
	job, svcErr := ctl.service.PublishJob(id, reviewerID)
	if svcErr != nil {
		ctl.handleError(c, svcErr)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, job)
}

type adminRejectRequest struct {
	Reason string `json:"reason"`
}

func (ctl *IngestController) AdminReject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "id 必须为正整数")
		return
	}
	var req adminRejectRequest
	_ = c.ShouldBindJSON(&req) // reason 可选

	reviewerID := ctl.currentUserID(c)
	job, svcErr := ctl.service.RejectJob(id, reviewerID, req.Reason)
	if svcErr != nil {
		ctl.handleError(c, svcErr)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, job)
}

// ─────────────────────── helpers ───────────────────────

func (ctl *IngestController) currentUserID(c *gin.Context) models.PrimaryKey {
	val, _ := c.Get(middlewares.CtxKeyUserID)
	return val.(models.PrimaryKey)
}

func (ctl *IngestController) handleError(c *gin.Context, err error) {
	if svcErr, ok := err.(*services.ServiceError); ok {
		utils.JSONError(c, svcErr.Status, svcErr.Code, svcErr.Message)
		return
	}
	utils.JSONError(c, http.StatusInternalServerError, "internal_error", "服务器开小差了，请稍后重试")
}

func (ctl *IngestController) handleUploadError(c *gin.Context, err error) {
	if upErr, ok := err.(*uploadError); ok {
		utils.JSONError(c, upErr.status, upErr.code, upErr.message)
		return
	}
	utils.JSONError(c, http.StatusInternalServerError, "internal_error", "服务器开小差了，请稍后重试")
}
