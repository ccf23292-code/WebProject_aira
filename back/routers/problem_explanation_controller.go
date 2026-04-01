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

// ProblemExplanationController 处理题解相关请求。
type ProblemExplanationController struct {
	service *services.ProblemExplanationService
}

func NewProblemExplanationController(service *services.ProblemExplanationService) *ProblemExplanationController {
	return &ProblemExplanationController{service: service}
}

func (ctl *ProblemExplanationController) RegisterPublicRoutes(group *gin.RouterGroup) {
	group.GET("/problems/:problem_id/explanations", ctl.List)
}

func (ctl *ProblemExplanationController) RegisterProtectedRoutes(group *gin.RouterGroup) {
	group.POST("/problems/:problem_id/explanations", ctl.Upsert)
	group.PATCH("/problems/:problem_id/explanations/:explanation_id", ctl.Update)
	group.POST("/problems/:problem_id/explanations/:explanation_id/vote", ctl.Vote)
}

func (ctl *ProblemExplanationController) List(c *gin.Context) {
	problemID, ok := ctl.parseUintParam(c, "problem_id")
	if !ok {
		return
	}

	userID := ctl.currentOptionalUserID(c)
	result, err := ctl.service.ListProblemExplanations(problemID, userID)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, result)
}

func (ctl *ProblemExplanationController) Upsert(c *gin.Context) {
	problemID, ok := ctl.parseUintParam(c, "problem_id")
	if !ok {
		return
	}

	var req services.UpsertProblemExplanationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	result, err := ctl.service.UpsertProblemExplanation(problemID, ctl.currentUserID(c), req)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, result)
}

func (ctl *ProblemExplanationController) Update(c *gin.Context) {
	problemID, ok := ctl.parseUintParam(c, "problem_id")
	if !ok {
		return
	}
	explanationID, ok := ctl.parseUintParam(c, "explanation_id")
	if !ok {
		return
	}

	var req services.UpsertProblemExplanationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	result, err := ctl.service.UpdateProblemExplanation(problemID, explanationID, ctl.currentUserID(c), req)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, result)
}

func (ctl *ProblemExplanationController) Vote(c *gin.Context) {
	problemID, ok := ctl.parseUintParam(c, "problem_id")
	if !ok {
		return
	}
	explanationID, ok := ctl.parseUintParam(c, "explanation_id")
	if !ok {
		return
	}

	var req services.VoteProblemExplanationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	result, err := ctl.service.VoteProblemExplanation(problemID, explanationID, ctl.currentUserID(c), req.Value)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, result)
}

func (ctl *ProblemExplanationController) parseUintParam(c *gin.Context, name string) (uint64, bool) {
	value, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", name+" 必须为正整数")
		return 0, false
	}
	return value, true
}

func (ctl *ProblemExplanationController) currentUserID(c *gin.Context) models.PrimaryKey {
	val, _ := c.Get(middlewares.CtxKeyUserID)
	return val.(models.PrimaryKey)
}

func (ctl *ProblemExplanationController) currentOptionalUserID(c *gin.Context) *models.PrimaryKey {
	val, exists := c.Get(middlewares.CtxKeyUserID)
	if !exists {
		return nil
	}
	userID := val.(models.PrimaryKey)
	return &userID
}

func (ctl *ProblemExplanationController) handleError(c *gin.Context, err error) {
	if svcErr, ok := err.(*services.ServiceError); ok {
		utils.JSONError(c, svcErr.Status, svcErr.Code, svcErr.Message)
		return
	}
	utils.JSONError(c, http.StatusInternalServerError, "internal_error", "服务器开小差了，请稍后重试")
}
