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

// AnswerController 处理做题记录。
type AnswerController struct {
	service *services.AnswerService
}

func NewAnswerController(service *services.AnswerService) *AnswerController {
	return &AnswerController{service: service}
}

// RegisterRoutes 注册做题记录接口。
// POST /api/answers
// POST /api/answers/batch
// GET  /api/answers
func (ctl *AnswerController) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("", ctl.Record)
	group.POST("/batch", ctl.RecordBatch)
	group.GET("", ctl.List)
}

// Record 写入做题记录。
func (ctl *AnswerController) Record(c *gin.Context) {
	var req services.AnswerRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}
	userID := ctl.currentUserID(c)
	if err := ctl.service.RecordAnswer(userID, req); err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccessMsg(c, http.StatusCreated, "recorded", nil)
}

// RecordBatch 批量写入做题记录。
func (ctl *AnswerController) RecordBatch(c *gin.Context) {
	var req services.AnswerBatchRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}
	userID := ctl.currentUserID(c)
	if err := ctl.service.RecordAnswersBatch(userID, req.Answers); err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccessMsg(c, http.StatusCreated, "recorded", nil)
}

// List 返回做题记录。
func (ctl *AnswerController) List(c *gin.Context) {
	userID := ctl.currentUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	records, total, err := ctl.service.ListAnswerRecords(userID, page, size)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, gin.H{
		"total": total,
		"page":  page,
		"size":  size,
		"items": records,
	})
}

func (ctl *AnswerController) currentUserID(c *gin.Context) models.PrimaryKey {
	val, _ := c.Get(middlewares.CtxKeyUserID)
	return val.(models.PrimaryKey)
}

func (ctl *AnswerController) handleError(c *gin.Context, err error) {
	if svcErr, ok := err.(*services.ServiceError); ok {
		utils.JSONError(c, svcErr.Status, svcErr.Code, svcErr.Message)
		return
	}
	utils.JSONError(c, http.StatusInternalServerError, "internal_error", "服务器开小差了，请稍后重试")
}
