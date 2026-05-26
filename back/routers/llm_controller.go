package routers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"warehouse-web/services"
	"warehouse-web/utils"
)

// LLMController 处理 LLM 相关请求（流式 SSE）。
type LLMController struct {
	service *services.LLMService
}

// NewLLMController 创建 LLMController。
func NewLLMController(service *services.LLMService) *LLMController {
	return &LLMController{service: service}
}

// RegisterRoutes 将 LLM 相关路由绑定到指定的路由组（该组应已挂载 AuthRequired 中间件）。
//
//	POST /api/llm/explain   body: { "problem_id": 123 }
func (ctl *LLMController) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/explain", ctl.Explain)
}

type explainRequest struct {
	ProblemID uint64 `json:"problem_id" binding:"required"`
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
//
// TODO: add per-user rate limit（在此处或独立中间件实现）
func (ctl *LLMController) Explain(c *gin.Context) {
	var req explainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}

	chunks, err := ctl.service.StreamExplain(c.Request.Context(), req.ProblemID)
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
				// service 那边正常关闭了 channel
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
			// 客户端断开：service 端 ctx 也会随之取消，goroutine 会自行退出
			return false
		}
	})
}

// writeSSE 将一条 SSE 事件写入响应。失败时只能尽力——客户端可能已断开。
func writeSSE(c *gin.Context, event string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	// 手写 SSE 格式而非 c.SSEvent，是为了精确控制 event/data 字段顺序及空行
	_, _ = c.Writer.WriteString("event: ")
	_, _ = c.Writer.WriteString(event)
	_, _ = c.Writer.WriteString("\ndata: ")
	_, _ = c.Writer.Write(data)
	_, _ = c.Writer.WriteString("\n\n")
	c.Writer.Flush()
}

// handleError 统一处理服务层错误（在 SSE 流开始前调用）。
func (ctl *LLMController) handleError(c *gin.Context, err error) {
	if svcErr, ok := err.(*services.ServiceError); ok {
		utils.JSONError(c, svcErr.Status, svcErr.Code, svcErr.Message)
		return
	}
	utils.JSONError(c, http.StatusInternalServerError, "internal_error", "服务器开小差了，请稍后重试")
}
