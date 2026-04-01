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

// RecallController 处理回忆卷相关接口。
type RecallController struct {
	service *services.RecallService
}

// NewRecallController 创建 RecallController。
func NewRecallController(service *services.RecallService) *RecallController {
	return &RecallController{service: service}
}

// RegisterRoutes 将回忆卷相关路由绑定到指定的路由组。
func (ctl *RecallController) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/courses/:course_id/papers", ctl.ListRecallPapers)
	group.POST("/courses/:course_id/papers", ctl.CreateRecallPaper)
	group.GET("/papers/:paper_id/question-types", ctl.ListQuestionTypes)
	group.GET("/papers/:paper_id/questions/top", ctl.ListTopQuestions)
	group.GET("/papers/:paper_id/questions", ctl.ListQuestionsBySequence)
	group.POST("/papers/:paper_id/questions", ctl.CreateQuestion)
	group.PATCH("/questions/:question_id", ctl.UpdateQuestion)
	group.POST("/questions/:question_id/support", ctl.SupportQuestion)
	group.GET("/questions/:question_id/comments", ctl.ListComments)
	group.POST("/questions/:question_id/comments", ctl.AddComment)
}

// ListRecallPapers 返回课程下的回忆卷。
func (ctl *RecallController) ListRecallPapers(c *gin.Context) {
	courseID := c.Param("course_id")
	papers, err := ctl.service.ListRecallPapers(courseID)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, papers)
}

// CreateRecallPaper 创建回忆卷。
func (ctl *RecallController) CreateRecallPaper(c *gin.Context) {
	courseID := c.Param("course_id")

	var req services.CreateRecallPaperRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	userID := ctl.currentUserID(c)
	paper, err := ctl.service.CreateRecallPaper(courseID, userID, req)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusCreated, paper)
}

// ListQuestionTypes 返回题型汇总。
func (ctl *RecallController) ListQuestionTypes(c *gin.Context) {
	paperID, err := strconv.ParseUint(c.Param("paper_id"), 10, 64)
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "paper_id 必须为正整数")
		return
	}

	result, svcErr := ctl.service.ListQuestionTypeSummary(paperID)
	if svcErr != nil {
		ctl.handleError(c, svcErr)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, result)
}

// ListTopQuestions 返回每个题号支持度最高的题目。
func (ctl *RecallController) ListTopQuestions(c *gin.Context) {
	paperID, err := strconv.ParseUint(c.Param("paper_id"), 10, 64)
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "paper_id 必须为正整数")
		return
	}

	questionType := c.Query("question_type")
	result, svcErr := ctl.service.ListTopQuestions(paperID, questionType)
	if svcErr != nil {
		ctl.handleError(c, svcErr)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, result)
}

// ListQuestionsBySequence 返回指定题号下的全部题目版本。
func (ctl *RecallController) ListQuestionsBySequence(c *gin.Context) {
	paperID, err := strconv.ParseUint(c.Param("paper_id"), 10, 64)
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "paper_id 必须为正整数")
		return
	}

	questionType := c.Query("question_type")
	sequence, err := strconv.Atoi(c.Query("sequence"))
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "sequence 必须为正整数")
		return
	}

	result, svcErr := ctl.service.ListQuestionsBySequence(paperID, questionType, sequence)
	if svcErr != nil {
		ctl.handleError(c, svcErr)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, result)
}

// CreateQuestion 创建题目。
func (ctl *RecallController) CreateQuestion(c *gin.Context) {
	paperID, err := strconv.ParseUint(c.Param("paper_id"), 10, 64)
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "paper_id 必须为正整数")
		return
	}

	var req services.CreateRecallQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	userID := ctl.currentUserID(c)
	question, svcErr := ctl.service.CreateRecallQuestion(paperID, userID, req)
	if svcErr != nil {
		ctl.handleError(c, svcErr)
		return
	}
	utils.JSONSuccess(c, http.StatusCreated, question)
}

// UpdateQuestion 更新题目内容。
func (ctl *RecallController) UpdateQuestion(c *gin.Context) {
	questionID, err := strconv.ParseUint(c.Param("question_id"), 10, 64)
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "question_id 必须为正整数")
		return
	}

	var req services.UpdateRecallQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	userID := ctl.currentUserID(c)
	question, svcErr := ctl.service.UpdateRecallQuestion(questionID, userID, req)
	if svcErr != nil {
		ctl.handleError(c, svcErr)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, question)
}

// SupportQuestion 为题目加一票支持。
func (ctl *RecallController) SupportQuestion(c *gin.Context) {
	questionID, err := strconv.ParseUint(c.Param("question_id"), 10, 64)
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "question_id 必须为正整数")
		return
	}

	userID := ctl.currentUserID(c)
	question, svcErr := ctl.service.SupportQuestion(questionID, userID)
	if svcErr != nil {
		ctl.handleError(c, svcErr)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, question)
}

// ListComments 返回评论列表。
func (ctl *RecallController) ListComments(c *gin.Context) {
	questionID, err := strconv.ParseUint(c.Param("question_id"), 10, 64)
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "question_id 必须为正整数")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	result, svcErr := ctl.service.ListComments(questionID, page, size)
	if svcErr != nil {
		ctl.handleError(c, svcErr)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, result)
}

// AddComment 新增评论。
func (ctl *RecallController) AddComment(c *gin.Context) {
	questionID, err := strconv.ParseUint(c.Param("question_id"), 10, 64)
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "question_id 必须为正整数")
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	userID := ctl.currentUserID(c)
	comment, svcErr := ctl.service.AddComment(questionID, userID, req.Content)
	if svcErr != nil {
		ctl.handleError(c, svcErr)
		return
	}
	utils.JSONSuccess(c, http.StatusCreated, comment)
}

// currentUserID 从 gin.Context 中取出当前登录用户的 ID。
func (ctl *RecallController) currentUserID(c *gin.Context) models.PrimaryKey {
	val, _ := c.Get(middlewares.CtxKeyUserID)
	return val.(models.PrimaryKey)
}

// handleError 统一处理服务层错误。
func (ctl *RecallController) handleError(c *gin.Context, err error) {
	if svcErr, ok := err.(*services.ServiceError); ok {
		utils.JSONError(c, svcErr.Status, svcErr.Code, svcErr.Message)
		return
	}
	utils.JSONError(c, http.StatusInternalServerError, "internal_error", "服务器开小差了，请稍后重试")
}
