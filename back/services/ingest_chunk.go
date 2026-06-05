package services

import (
	"context"
	"encoding/json"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// 分块阈值与目标块大小（rune 数）。
//
//   - 小于阈值的文本走原 CleanQuestionText 单次调用，省事
//   - 超过阈值才切块，每块目标 chunkTargetRunes 上下浮动
//
// 经验值（基于 DeepSeek chat 8192 tokens 输出上限，每题 ~200 输出 tokens）：
//   - 阈值 ~ 6000 rune（中文场景下约 30 题源文本）
//   - 目标 4000 rune 一块 = 20 题左右，给 LLM 留余量
const (
	chunkTriggerRunes = 6000
	chunkTargetRunes  = 4000
	chunkMaxParallel  = 3 // 并发数；DeepSeek 速率限制宽松，3 路足够
)

// 题号边界识别正则（按命中优先级排）。
//
//   - 行首阿拉伯数字 + 中文/英文标点：`1.` `1、` `1)` `1）`
//   - "第N题" / "第 N 题"
//   - Markdown header 含数字：`## 1.` `### 1)`
//   - 题型小节标题（不算严格边界，但适合作 fallback 切点）
var questionBoundaryRe = regexp.MustCompile(
	`(?m)^[ \t]*(?:\d+[\.\)、）]\s|第\s*\d+\s*题|#{1,4}\s*\d+[\.\)、）]?|【\s*\d+\s*】)`,
)

// 题型小节标题，作为 fallback 边界
var sectionHeaderRe = regexp.MustCompile(
	`(?m)^[ \t]*(?:#{1,4}\s*)?(?:一|二|三|四|五|六|七|八|九|十|判断题|单选题|单项选择题|多选题|多项选择题|填空题|简答题|名词解释|论述题|计算题|材料题)\b`,
)

// chunkRawTextByQuestionBoundary 按题号边界把长文本切成多个块。
//
// 算法：
//  1. 找所有题号边界的字符（rune）位置
//  2. 从头开始扫，累计 rune 数；每超过 chunkTargetRunes 就在下一个边界处断开
//  3. 没找到边界时回退到题型小节标题；仍找不到回退到滑窗切割
//
// 保证：每块以一个题号开头（除了第一块可能含小节前导）。
// 不保证：单题超过 chunkTargetRunes 时一个块装一题（认了）。
func chunkRawTextByQuestionBoundary(text string) []string {
	runes := []rune(text)
	if len(runes) <= chunkTriggerRunes {
		return []string{text}
	}

	// 1) 收集所有边界 rune 偏移
	boundaries := collectBoundariesRuneIdx(text, runes)
	if len(boundaries) < 2 {
		// 没题号边界 → 走滑窗
		return chunkBySliding(runes)
	}

	// 2) 沿边界贪心累积
	chunks := make([]string, 0, len(runes)/chunkTargetRunes+1)
	start := 0
	lastBoundary := boundaries[0]
	for _, b := range boundaries {
		// 已累积超过目标 + 距上次断点合理距离 → 在当前边界断开
		if b-start >= chunkTargetRunes && b > lastBoundary {
			chunks = append(chunks, strings.TrimSpace(string(runes[start:b])))
			start = b
			lastBoundary = b
		}
	}
	// 尾巴
	if start < len(runes) {
		tail := strings.TrimSpace(string(runes[start:]))
		if tail != "" {
			chunks = append(chunks, tail)
		}
	}
	return chunks
}

// collectBoundariesRuneIdx 把正则匹配的字节偏移换算成 rune 偏移。
// 题号边界优先；找不到任何题号边界时降级到题型小节标题。
func collectBoundariesRuneIdx(text string, runes []rune) []int {
	byteToRune := buildByteToRune(text, len(runes))
	matches := questionBoundaryRe.FindAllStringIndex(text, -1)
	if len(matches) < 2 {
		matches = sectionHeaderRe.FindAllStringIndex(text, -1)
	}
	if len(matches) == 0 {
		return nil
	}
	out := make([]int, 0, len(matches))
	seen := make(map[int]struct{}, len(matches))
	for _, m := range matches {
		ri := byteToRune[m[0]]
		if _, dup := seen[ri]; dup {
			continue
		}
		seen[ri] = struct{}{}
		out = append(out, ri)
	}
	sort.Ints(out)
	return out
}

// buildByteToRune 给定原文与 rune 切片，建立 "字节偏移 → rune 偏移" 映射表。
// 长度为 len(text)+1，超尾位置映射到 len(runes)。
func buildByteToRune(text string, runesLen int) []int {
	out := make([]int, len(text)+1)
	ri := 0
	for bi := range text {
		out[bi] = ri
		ri++
	}
	out[len(text)] = runesLen
	return out
}

// chunkBySliding 没有可识别边界时按 rune 数硬切，相邻块加 200 rune 重叠
// 防止题被切两半（重叠部分由 LLM 在每块里重复识别，merge 时按题干指纹去重）。
func chunkBySliding(runes []rune) []string {
	const overlap = 200
	var chunks []string
	for start := 0; start < len(runes); {
		end := start + chunkTargetRunes
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, strings.TrimSpace(string(runes[start:end])))
		if end >= len(runes) {
			break
		}
		start = end - overlap
		if start < 0 {
			start = 0
		}
	}
	return chunks
}

// CleanQuestionTextSmart 是大试卷自适应清洗入口：
//
//   - 文本量小 → 直接调 CleanQuestionText 一次过
//   - 文本量大 → 切块、并发调 LLM、合并 items
//
// 合并时 sequence_id 全局重编号（1..N），保 source_id 唯一约束不撞。
// LLM 失败的块跳过（仅日志），整体仍返回已识别的部分。
func CleanQuestionTextSmart(
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

	chunks := chunkRawTextByQuestionBoundary(rawText)
	if len(chunks) == 1 {
		return CleanQuestionText(ctx, llm, chunks[0])
	}

	log.Printf("ingest: smart clean splitting into %d chunks (total %d runes)", len(chunks), len([]rune(rawText)))

	results := parallelCleanChunks(ctx, llm, chunks, CleanQuestionText)
	if len(results.merged) == 0 {
		return nil, newServiceError("llm_no_items", 502, "分块清洗后未识别到任何题目（所有块均失败或为空）")
	}

	// 重新编号 sequence_id 为全局连续整数，避免跨块撞 seq
	for i := range results.merged {
		results.merged[i]["sequence_id"] = i + 1
	}

	// 重新序列化 RawJSON 让 admin 编辑时看到一致结构
	envBytes := mustMarshalEnvelope(results.merged)

	return &IngestCleanResult{
		Items:     results.merged,
		RawJSON:   string(envBytes),
		Model:     llm.cfg.Model,
		Truncated: results.anyTruncated,
	}, nil
}

// chunkCleanResults 是并发清洗后聚合的中间结构。
type chunkCleanResults struct {
	merged       []map[string]any
	anyTruncated bool
}

// parallelCleanChunks 并发调用单块清洗，按 chunkMaxParallel 限并发。
// cleanFn 通常是 CleanQuestionText / CleanExplanationText，便于复用给题解流程。
func parallelCleanChunks(
	ctx context.Context,
	llm *LLMService,
	chunks []string,
	cleanFn func(context.Context, *LLMService, string) (*IngestCleanResult, error),
) chunkCleanResults {
	type chunkOut struct {
		idx    int
		result *IngestCleanResult
		err    error
	}
	resCh := make(chan chunkOut, len(chunks))
	sem := make(chan struct{}, chunkMaxParallel)
	var wg sync.WaitGroup

	for i, c := range chunks {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, chunk string) {
			defer wg.Done()
			defer func() { <-sem }()
			r, err := cleanFn(ctx, llm, chunk)
			resCh <- chunkOut{idx: idx, result: r, err: err}
		}(i, c)
	}
	wg.Wait()
	close(resCh)

	// 按 idx 排序，让合并后的题目大致保持原文顺序
	all := make([]chunkOut, 0, len(chunks))
	for o := range resCh {
		all = append(all, o)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].idx < all[j].idx })

	out := chunkCleanResults{merged: make([]map[string]any, 0, 64)}
	for _, o := range all {
		if o.err != nil {
			log.Printf("ingest: chunk %d clean failed: %v", o.idx, o.err)
			continue
		}
		if o.result == nil {
			continue
		}
		if o.result.Truncated {
			out.anyTruncated = true
		}
		out.merged = append(out.merged, o.result.Items...)
	}
	return out
}

// mustMarshalEnvelope 把 items 数组包成 {"items":[...]} 序列化。
// 失败时返回最小可解析的 {"items":[]}。
func mustMarshalEnvelope(items []map[string]any) []byte {
	b, err := json.Marshal(map[string]any{"items": items})
	if err != nil {
		return []byte(`{"items":[]}`)
	}
	return b
}
