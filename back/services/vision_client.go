package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

// VisionConfig 收敛图片 OCR 调用所需的配置。
// 默认目标是阿里云通义千问 VL（OpenAI 兼容协议）。
//
//	LLM_VISION_API_KEY        必填；为空时 Enabled()==false
//	LLM_VISION_BASE_URL       默认 https://dashscope.aliyuncs.com/compatible-mode/v1
//	LLM_VISION_MODEL          默认 qwen-vl-max
//	LLM_VISION_TIMEOUT_SECONDS 默认 90
type VisionConfig struct {
	APIKey  string
	BaseURL string
	Model   string
	Timeout time.Duration
}

func (c VisionConfig) Enabled() bool {
	return strings.TrimSpace(c.APIKey) != ""
}

func LoadVisionConfigFromEnv() VisionConfig {
	timeout := 90 * time.Second
	if raw := strings.TrimSpace(os.Getenv("LLM_VISION_TIMEOUT_SECONDS")); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
			timeout = time.Duration(seconds) * time.Second
		}
	}

	baseURL := strings.TrimSpace(os.Getenv("LLM_VISION_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	model := strings.TrimSpace(os.Getenv("LLM_VISION_MODEL"))
	if model == "" {
		model = "qwen-vl-max"
	}

	return VisionConfig{
		APIKey:  strings.TrimSpace(os.Getenv("LLM_VISION_API_KEY")),
		BaseURL: baseURL,
		Model:   model,
		Timeout: timeout,
	}
}

// VisionClient 提供从单张图片提取题目原文的能力。
// 当 APIKey 未配置时 Enabled()=false，调用 ExtractFromImage 会拿到明确错误。
type VisionClient struct {
	cfg    VisionConfig
	client *openai.Client
}

func NewVisionClient(cfg VisionConfig) *VisionClient {
	if !cfg.Enabled() {
		return &VisionClient{cfg: cfg}
	}
	oc := openai.DefaultConfig(cfg.APIKey)
	oc.BaseURL = cfg.BaseURL
	return &VisionClient{cfg: cfg, client: openai.NewClientWithConfig(oc)}
}

func (v *VisionClient) Enabled() bool {
	return v != nil && v.client != nil
}

// ExtractFromImage 把指定图片转 base64，调用 vision 模型，回传识别出的 Markdown 原文。
//
// system prompt 强调：仅做"原样录入"，不要总结、不要回答题目本身。
func (v *VisionClient) ExtractFromImage(ctx context.Context, path string) (string, error) {
	if !v.Enabled() {
		return "", newExtractError(
			"vision_disabled",
			"图片识别未启用：请联系管理员配置 LLM_VISION_API_KEY",
		)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", newExtractError("read_failed", "读取图片失败: "+err.Error())
	}
	if len(data) == 0 {
		return "", newExtractError("empty_content", "图片内容为空")
	}

	mt := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
	if mt == "" {
		mt = "image/jpeg"
	}
	dataURL := fmt.Sprintf("data:%s;base64,%s", mt, base64.StdEncoding.EncodeToString(data))

	callCtx, cancel := context.WithTimeout(ctx, v.cfg.Timeout)
	defer cancel()

	resp, err := v.client.CreateChatCompletion(callCtx, openai.ChatCompletionRequest{
		Model: v.cfg.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: visionExtractSystemPrompt,
			},
			{
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{
							URL:    dataURL,
							Detail: openai.ImageURLDetailHigh,
						},
					},
					{
						Type: openai.ChatMessagePartTypeText,
						Text: "请把图片中的全部题目内容（题干、选项、答案、题号）按原样录入为 Markdown 文本。",
					},
				},
			},
		},
	})
	if err != nil {
		return "", newExtractError("vision_upstream_error", "Vision 模型调用失败: "+err.Error())
	}
	if len(resp.Choices) == 0 {
		return "", newExtractError("vision_empty_response", "Vision 模型未返回内容")
	}

	text := strings.TrimSpace(resp.Choices[0].Message.Content)
	if text == "" {
		return "", newExtractError("vision_empty_response", "Vision 模型未识别到文本")
	}
	return text, nil
}

const visionExtractSystemPrompt = `你是 OCR 助手。把用户提供的图片中"题目原文"忠实录入为 Markdown 文本：
- 保留题号、题干、选项（A/B/C/D…）、答案。
- 数学公式使用 LaTeX，行内用 $...$，块级用 $$...$$。
- 不要总结、不要解释、不要回答题目，只做原样录入。
- 若图片含多道题，按出现顺序逐题输出，题与题之间空行隔开。`
