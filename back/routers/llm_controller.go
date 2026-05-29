package routers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"warehouse-web/middlewares"
	"warehouse-web/models"
	"warehouse-web/services"
	"warehouse-web/utils"
)

// LLMController 处理 LLM 相关请求（流式 SSE + 缓存查询）。
type LLMController struct {
	service *services.LLMService
}

// NewLLMController 创建 LLMController。
func NewLLMController(service *services.LLMService) *LLMController {
	return &LLMController{service: service}
}

// RegisterRoutes 将 LLM 相关路由绑定到指定的路由组（该组应已挂载 AuthRequired 中间件）。
//
//	POST /api/llm/explain                          body: { "problem_id": 123 }
//	GET  /api/llm/explain/cached?problem_id=123    返回最近一条缓存（或 found:false）
func (ctl *LLMController) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/explain", ctl.Explain)
	group.GET("/explain/cached", ctl.GetCached)
}

type explainRequest struct {
	ProblemID uint64 `json:"problem_id" binding:"required"`
}

// cachedExplanationResponse 是 GET /explain/cached 的响应 data 结构。
// 用 found 字段而非 HTTP 404 表达"未命中"，让前端用一次 fetch 解决。
type cachedExplanationResponse struct {
	Found     bool   `json:"found"`
	Content   string `json:"content,omitempty"`
	Model     string `json:"model,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	Used      int64  `json:"used"`  // 当前用户对该题已生成次数
	Limit     int    `json:"limit"` // 每人每题生成上限
}

// GetCached 返回指定题目最近一条 LLM 缓存解析。
// GET /api/llm/explain/cached?problem_id=123
func (ctl *LLMController) GetCached(c *gin.Context) {
	raw := c.Query("problem_id")
	problemID, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || problemID == 0 {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "problem_id 必须为正整数")
		return
	}

	// 当前用户对该题已用次数 + 上限，供前端展示剩余/禁用按钮
	used, _ := ctl.service.CountUserExplanations(ctl.currentUserID(c), problemID)
	limit := services.MaxExplainPerUserPerProblem

	record, err := ctl.service.GetLatestExplanation(problemID)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	if record == nil {
		utils.JSONSuccess(c, http.StatusOK, cachedExplanationResponse{Found: false, Used: used, Limit: limit})
		return
	}
	utils.JSONSuccess(c, http.StatusOK, cachedExplanationResponse{
		Found:     true,
		Content:   record.Content,
		Model:     record.Model,
		CreatedAt: record.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Used:      used,
		Limit:     limit,
	})
}

// Explain 针对指定题目流式返回 AI 解析（Server-Sent Events）。
//
// 响应是 text/event-stream，事件协议：
//
//	event: token   data: {"text":"..."}
//	event: done    data: {"reason":"stop"}
//	event: error   data: {"message":"..."}
//
// 客户端断开会通过 c.Request.Context() 传播到 service 层，stream 优雅关闭。
// 正常结束时 service 会异步落库到 llm_explanations 表（前端无需额外保存请求）。
//
// TODO: add per-user rate limit（在此处或独立中间件实现）
func (ctl *LLMController) Explain(c *gin.Context) {
	var req explainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	userID := ctl.currentUserID(c)
	chunks, err := ctl.service.StreamExplain(c.Request.Context(), userID, req.ProblemID)
	if err != nil {
		ctl.handleError(c, err)
		return
	}

	// ── SSE 响应头 ────────────────────────
	// X-Accel-Buffering: no 关掉 Nginx 反向代理缓冲，否则 token 会被攒到流结束才一次性吐出
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	clientGone := c.Request.Context().Done()

	c.Stream(func(w io.Writer) bool {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				return false
			}
			switch chunk.Kind {
			case services.LLMChunkToken:
				writeSSE(c, "token", map[string]string{"text": chunk.Text})
				return true
			case services.LLMChunkDone:
				writeSSE(c, "done", map[string]string{"reason": chunk.Reason})
				return false
			case services.LLMChunkError:
				msg := "LLM 调用失败"
				if chunk.Err != nil && !errors.Is(chunk.Err, io.EOF) {
					msg = chunk.Err.Error()
				}
				writeSSE(c, "error", map[string]string{"message": msg})
				return false
			default:
				return true
			}

		case <-clientGone:
			return false
		}
	})
}

// writeSSE 将一条 SSE 事件写入响应。
func writeSSE(c *gin.Context, event string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_, _ = c.Writer.WriteString("event: ")
	_, _ = c.Writer.WriteString(event)
	_, _ = c.Writer.WriteString("\ndata: ")
	_, _ = c.Writer.Write(data)
	_, _ = c.Writer.WriteString("\n\n")
	c.Writer.Flush()
}

// currentUserID 从 gin.Context 中取出当前登录用户的 ID。
func (ctl *LLMController) currentUserID(c *gin.Context) models.PrimaryKey {
	val, _ := c.Get(middlewares.CtxKeyUserID)
	return val.(models.PrimaryKey)
}

// handleError 统一处理服务层错误（在 SSE 流开始前调用）。
func (ctl *LLMController) handleError(c *gin.Context, err error) {
	if svcErr, ok := err.(*services.ServiceError); ok {
		utils.JSONError(c, svcErr.Status, svcErr.Code, svcErr.Message)
		return
	}
	utils.JSONError(c, http.StatusInternalServerError, "internal_error", "服务器开小差了，请稍后重试")
}
