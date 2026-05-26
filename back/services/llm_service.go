package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"

	"warehouse-web/models"
)

// LLMChunkKind 标识一次流式输出事件的类型。
type LLMChunkKind string

const (
	LLMChunkToken LLMChunkKind = "token" // 一段 token 文本
	LLMChunkDone  LLMChunkKind = "done"  // 流正常结束
	LLMChunkError LLMChunkKind = "error" // 流中发生错误
)

// LLMChunk 是 LLMService 向调用方推送的最小事件单元。
type LLMChunk struct {
	Kind   LLMChunkKind
	Text   string // Kind=token 时填实际内容
	Reason string // Kind=done 时填 finish_reason
	Err    error  // Kind=error 时填底层错误
}

// LLMConfig 收敛 LLM 调用所需的运行时配置。
type LLMConfig struct {
	APIKey  string
	BaseURL string
	Model   string
	Timeout time.Duration
}

// Enabled 报告是否已配置可用的 API Key。
func (c LLMConfig) Enabled() bool {
	return strings.TrimSpace(c.APIKey) != ""
}

// LoadLLMConfigFromEnv 从环境变量加载 LLM 配置，未配置项落到默认值。
//   - LLM_API_KEY          必填；为空时 Enabled()==false，接口返回 503
//   - LLM_BASE_URL         默认 https://api.deepseek.com/v1
//   - LLM_MODEL            默认 deepseek-chat
//   - LLM_TIMEOUT_SECONDS  默认 60
func LoadLLMConfigFromEnv() LLMConfig {
	timeout := 60 * time.Second
	if raw := strings.TrimSpace(os.Getenv("LLM_TIMEOUT_SECONDS")); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			timeout = time.Duration(seconds) * time.Second
		}
	}

	baseURL := strings.TrimSpace(os.Getenv("LLM_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}

	model := strings.TrimSpace(os.Getenv("LLM_MODEL"))
	if model == "" {
		model = "deepseek-chat"
	}

	return LLMConfig{
		APIKey:  strings.TrimSpace(os.Getenv("LLM_API_KEY")),
		BaseURL: baseURL,
		Model:   model,
		Timeout: timeout,
	}
}

// LLMService 封装与 OpenAI 兼容大模型的交互（DeepSeek/Kimi/智谱 等）。
type LLMService struct {
	cfg    LLMConfig
	client *openai.Client
	paper  *PaperService
}

// NewLLMService 构造 LLMService。若 cfg.Enabled() 为 false，则不创建底层 client，
// 所有流式接口在运行时返回 503，避免空 key 也能起服。
func NewLLMService(cfg LLMConfig, paper *PaperService) *LLMService {
	var client *openai.Client
	if cfg.Enabled() {
		oc := openai.DefaultConfig(cfg.APIKey)
		oc.BaseURL = cfg.BaseURL
		client = openai.NewClientWithConfig(oc)
	}
	return &LLMService{cfg: cfg, client: client, paper: paper}
}

// Enabled 反映服务是否已就绪（API Key 已配置）。
func (s *LLMService) Enabled() bool {
	return s.client != nil
}

// StreamExplain 针对给定题目向 LLM 请求 AI 解析，并通过 channel 实时推送 token。
// 返回的 channel 在流结束（成功 / 错误 / 取消）时会被关闭。
// 调用方应 range 该 channel 直到关闭，并通过 ctx 控制取消（例如客户端断开）。
//
// TODO: add per-user rate limit（建议在此处用 token bucket 限制每分钟次数）
func (s *LLMService) StreamExplain(ctx context.Context, problemID uint64) (<-chan LLMChunk, error) {
	if !s.Enabled() {
		return nil, newServiceError("llm_disabled", http.StatusServiceUnavailable, "LLM 服务未配置")
	}

	problem, err := s.paper.GetProblem(problemID)
	if err != nil {
		return nil, err
	}
	if problem == nil {
		return nil, newServiceError("not_found", http.StatusNotFound, "problem not found")
	}

	prompt := buildExplainPrompt(problem)
	req := openai.ChatCompletionRequest{
		Model: s.cfg.Model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: explainSystemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
		Stream: true,
	}

	// 用一个挂在父 ctx 下的子 ctx，附加单次请求总超时
	streamCtx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	stream, err := s.client.CreateChatCompletionStream(streamCtx, req)
	if err != nil {
		cancel()
		return nil, newServiceError("llm_upstream_error", http.StatusBadGateway, fmt.Sprintf("LLM 请求失败: %v", err))
	}

	out := make(chan LLMChunk, 16)

	go func() {
		defer close(out)
		defer cancel()
		defer stream.Close()

		for {
			select {
			case <-streamCtx.Done():
				if errors.Is(streamCtx.Err(), context.Canceled) {
					// 客户端主动断开，静默退出
					return
				}
				out <- LLMChunk{Kind: LLMChunkError, Err: streamCtx.Err()}
				return
			default:
			}

			resp, recvErr := stream.Recv()
			if errors.Is(recvErr, io.EOF) {
				out <- LLMChunk{Kind: LLMChunkDone, Reason: "stop"}
				return
			}
			if recvErr != nil {
				out <- LLMChunk{Kind: LLMChunkError, Err: recvErr}
				return
			}
			if len(resp.Choices) == 0 {
				continue
			}
			choice := resp.Choices[0]
			if delta := choice.Delta.Content; delta != "" {
				select {
				case out <- LLMChunk{Kind: LLMChunkToken, Text: delta}:
				case <-streamCtx.Done():
					return
				}
			}
			if choice.FinishReason != "" {
				out <- LLMChunk{Kind: LLMChunkDone, Reason: string(choice.FinishReason)}
				return
			}
		}
	}()

	return out, nil
}

/* ════════════ 提示词 ════════════ */

const explainSystemPrompt = `你是一名严谨、耐心的大学课程助教。请基于用户提供的题目，给出参考解析：
1. 先用 2-4 句话概述解题思路。
2. 再给出最终答案（与题目"正确答案"对齐；如果你对答案有异议请明确指出）。
3. 最后用 1-2 条要点总结相关知识点，便于复习。
全程使用 Markdown 排版，公式用 LaTeX，简洁、不要废话。`

// buildExplainPrompt 根据 Problem 拼一段 user prompt，包含题干、选项、正确答案。
func buildExplainPrompt(p *models.Problem) string {
	var b strings.Builder
	b.WriteString("题目内容：\n")
	b.WriteString(strings.TrimSpace(p.Test))
	b.WriteString("\n")

	if len(p.Options) > 0 {
		b.WriteString("\n选项：\n")
		for _, opt := range p.Options {
			b.WriteString("- ")
			b.WriteString(opt.Option)
			b.WriteString(". ")
			b.WriteString(opt.Text)
			b.WriteString("\n")
		}
	}

	if answer := strings.TrimSpace(p.Answer); answer != "" {
		b.WriteString("\n正确答案：")
		b.WriteString(answer)
		b.WriteString("\n")
	}

	b.WriteString("\n请按系统要求给出参考解析。")
	return b.String()
}
