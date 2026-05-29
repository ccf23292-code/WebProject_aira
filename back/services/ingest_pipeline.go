package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// IngestCleanResult 是 LLM 清洗管道对外返回的结果。
//
//   - Items 是原始结构化数据（题目或题解列表）。
//   - RawJSON 是 LLM 原始 JSON 字符串，方便落库后 admin 编辑 / 排错。
//   - Model 是实际使用的模型名，便于审计。
type IngestCleanResult struct {
	Items   []map[string]any
	RawJSON string
	Model   string
}

// CleanQuestionText 把预处理后的 Markdown 全文丢给 LLM，要求结构化为题目数组。
//
// 强制 JSON 输出（DeepSeek 兼容 OpenAI 的 response_format=json_object）。
// 提示词要求顶层包装为 {"items":[...]}，避免裸数组导致部分模型拒绝 JSON 模式。
func CleanQuestionText(
	ctx context.Context,
	llm *LLMService,
	rawText string,
) (*IngestCleanResult, error) {
	if llm == nil || !llm.Enabled() {
		return nil, newServiceError("llm_disabled", 503, "LLM 服务未配置")
	}
	rawText = strings.TrimSpace(rawText)
	if rawText == "" {
		return nil, newServiceError("invalid_request", 400, "待清洗内容为空")
	}

	callCtx, cancel := context.WithTimeout(ctx, llm.cfg.Timeout)
	defer cancel()

	resp, err := llm.client.CreateChatCompletion(callCtx, openai.ChatCompletionRequest{
		Model: llm.cfg.Model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: questionCleanSystemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: rawText},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	})
	if err != nil {
		return nil, newServiceError("llm_upstream_error", 502, fmt.Sprintf("LLM 调用失败: %v", err))
	}
	if len(resp.Choices) == 0 {
		return nil, newServiceError("llm_empty_response", 502, "LLM 未返回内容")
	}

	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	items, err := parseItemsEnvelope(content)
	if err != nil {
		return nil, newServiceError("llm_invalid_json", 502, "LLM 返回不符合 JSON 结构: "+err.Error())
	}
	if len(items) == 0 {
		return nil, newServiceError("llm_no_items", 502, "LLM 未识别到任何题目")
	}

	return &IngestCleanResult{
		Items:   items,
		RawJSON: content,
		Model:   llm.cfg.Model,
	}, nil
}

// CleanExplanationText 把题解原文结构化为 {sequence_id, content_md} 列表。
func CleanExplanationText(
	ctx context.Context,
	llm *LLMService,
	rawText string,
) (*IngestCleanResult, error) {
	if llm == nil || !llm.Enabled() {
		return nil, newServiceError("llm_disabled", 503, "LLM 服务未配置")
	}
	rawText = strings.TrimSpace(rawText)
	if rawText == "" {
		return nil, newServiceError("invalid_request", 400, "待清洗内容为空")
	}

	callCtx, cancel := context.WithTimeout(ctx, llm.cfg.Timeout)
	defer cancel()

	resp, err := llm.client.CreateChatCompletion(callCtx, openai.ChatCompletionRequest{
		Model: llm.cfg.Model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: explanationCleanSystemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: rawText},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	})
	if err != nil {
		return nil, newServiceError("llm_upstream_error", 502, fmt.Sprintf("LLM 调用失败: %v", err))
	}
	if len(resp.Choices) == 0 {
		return nil, newServiceError("llm_empty_response", 502, "LLM 未返回内容")
	}

	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	items, err := parseItemsEnvelope(content)
	if err != nil {
		return nil, newServiceError("llm_invalid_json", 502, "LLM 返回不符合 JSON 结构: "+err.Error())
	}
	if len(items) == 0 {
		return nil, newServiceError("llm_no_items", 502, "LLM 未识别到任何题解")
	}

	return &IngestCleanResult{
		Items:   items,
		RawJSON: content,
		Model:   llm.cfg.Model,
	}, nil
}

// parseItemsEnvelope 解析 {"items":[...]} 信封。
// 若 LLM 误返回了顶层数组，也兜底解析一次。
func parseItemsEnvelope(content string) ([]map[string]any, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("empty content")
	}

	// 优先尝试 {"items":[...]}
	var envelope struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal([]byte(content), &envelope); err == nil && envelope.Items != nil {
		return envelope.Items, nil
	}

	// 兜底：顶层就是数组
	var arr []map[string]any
	if err := json.Unmarshal([]byte(content), &arr); err == nil {
		return arr, nil
	}

	return nil, errors.New(`expected JSON like {"items":[...]}`)
}

// ValidateQuestionItems 对 LLM 输出的题目做最小化字段校验，避免错误数据扩散到入库阶段。
// 不强制 difficulty/tags 等可选字段，但要求 test 非空、有合法 question_type。
func ValidateQuestionItems(items []map[string]any) error {
	allowedType := map[string]bool{
		"singleChoice":   true,
		"multipleChoice": true,
		"trueOrFalse":    true,
		"fillBlank":      true,
		"shortAnswer":    true,
	}
	for i, item := range items {
		test, _ := item["test"].(string)
		if strings.TrimSpace(test) == "" {
			return fmt.Errorf("第 %d 题缺少 test 字段", i+1)
		}
		qt, _ := item["question_type"].(string)
		if !allowedType[qt] {
			return fmt.Errorf("第 %d 题 question_type 非法: %q", i+1, qt)
		}
	}
	return nil
}

// ValidateExplanationItems 校验题解输出格式：要求 sequence_id 与 content_md 都存在。
func ValidateExplanationItems(items []map[string]any) error {
	for i, item := range items {
		if _, ok := item["sequence_id"]; !ok {
			return fmt.Errorf("第 %d 条题解缺少 sequence_id", i+1)
		}
		content, _ := item["content_md"].(string)
		if strings.TrimSpace(content) == "" {
			return fmt.Errorf("第 %d 条题解 content_md 为空", i+1)
		}
	}
	return nil
}

const questionCleanSystemPrompt = `你是一名课程题库整理助手。请把用户提供的题目原文转换为 JSON。

═══ 输出 schema ═══
1. 输出形如 {"items":[ ... ]} 的 JSON 对象，不要任何 Markdown 代码块包裹。
2. items 中每个对象遵守如下 schema（不要添加额外字段，缺失字段填空字符串或空数组）：
   {
     "sequence_id": 整数,
     "question_type": "singleChoice" | "multipleChoice" | "trueOrFalse" | "fillBlank" | "shortAnswer",
     "test": "题干（保留 LaTeX：行内 $...$、块级 $$...$$）",
     "options": [ { "option": "A", "text": "..." }, ... ],
     "answer": "标准答案（选择题填字母如 A 或 AB；判断题填 True/False；其他题填答案文本）",
     "explanation": "原文若给出解析就抄录，否则填空字符串",
     "difficulty": "easy" | "medium" | "hard"（不确定时填 "medium"）,
     "tags": ["知识点1", "知识点2"]
   }
3. 非选择题的 options 一律是 []。
4. 题号从用户原文里识别，无法识别则按出现顺序从 1 开始编号。

═══ 自检与修复规则（在输出前自己过一遍）═══

[A] 排版修复
  - 数学公式必须被 $...$（行内）或 $$...$$（块级）包围。看到裸的 "O(n log n)" "x^2" "\\sqrt{2}" 这种就补上 $...$。
  - 选项里只列字母不带文字的（如 "A. " 后面空），按原文该有的内容补；实在没法判断就保留空字符串。
  - 中英文标点混用时统一成与上下文一致的风格（中文题干用中文标点，英文题干用英文标点）。

[B] OCR 常见字符混淆 — 按上下文判断真实字符
  - 数字 0 vs 字母 O：在变量/标识符上下文中倾向字母 O；在数值上下文中倾向数字 0
  - 数字 1 vs 字母 l vs 字母 I：在编程/字母列表中倾向 l/I，在数学计数中倾向 1
  - 数字 5 vs 字母 S：题号位置倾向 5
  - 中文 "○" / "〇" 常被识为大写 "O"，反之亦然
  - 全角 "（）" vs 半角 "()"、全角 "，" vs 半角 "," — 公式里统一用半角

[C] 答案与题干自洽性检查
  - 单选题：answer 必须是 options 里实际存在的某个字母（A/B/C/D…）。若 OCR 识别出"答案 E"但选项只到 D，倾向认为是 OCR 错位，重新审视答案字符。
  - 多选题：answer 每个字母都必须出现在 options 里。
  - 判断题：answer 只能是 "True" 或 "False"（其它如"对/错/√/×/T/F"统一映射）。
  - 填空题：若题干没有空位标记（____、{}、（）等），但 answer 是单值，考虑是否其实是简答题。
  - 矛盾不可调和时，answer 留空字符串，而不是硬塞错的。

[D] 严禁脑补缺失内容
  - 原文截断、模糊、有缺漏 → 字段留空字符串/空数组，不要凭知识补全。
  - 没有官方解析 → explanation 留空，不要自己写。
  - 没有难度标注 → difficulty 填 "medium"（保守默认），不要根据题目"看起来"判断。
  - 没有 tags → tags 填 [] 或最多 1 个最确定的，不要罗列一堆。`

const explanationCleanSystemPrompt = `你是一名课程题解整理助手。请把用户提供的题解原文按题号拆分为 JSON。

═══ 输出 schema ═══
1. 输出形如 {"items":[ ... ]} 的 JSON 对象，不要任何 Markdown 代码块包裹。
2. items 中每个对象遵守如下 schema：
   {
     "sequence_id": 整数,                // 对应原文里"第几题"
     "content_md": "Markdown 题解正文，保留 LaTeX；不要用三个反引号代码块包裹整段"
   }
3. 若某题无题解则不输出该项。

═══ 自检与修复规则 ═══

[A] 排版：数学公式必须包在 $...$ / $$...$$ 里；保留原解析的代码块（如果有，单独用三反引号围起，但不要包整段）。

[B] OCR 字符纠正：参考题目清洗的常见混淆（0/O、1/l、5/S、全半角标点）按上下文修正。

[D] 严禁脑补：原文残缺就只输出残缺的内容，不要补全完整解题过程；某题原文没有解析就直接跳过该题（不输出该 item）。`
