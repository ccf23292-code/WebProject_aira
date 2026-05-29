package services

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"unicode"

	"gorm.io/gorm"
)

// 查重默认参数。这些不上配置，等真实数据反馈再调。
const (
	dedupNGramSize    = 3    // 3-gram，对中文友好
	dedupThreshold    = 0.70 // Jaccard 相似度阈值
	dedupMaxMatches   = 3    // 每道新题最多保留前 3 个疑似匹配，避免过载
	dedupSnippetChars = 80   // 展示用的题干片段长度（rune 数）
)

// DedupMatch 是单条疑似重复记录，最终序列化到 IngestJob.DedupWarnings。
type DedupMatch struct {
	NewSeq          int     `json:"seq"`               // 新题在 ingest 结果中的 sequence_id
	ProblemID       uint64  `json:"problem_id"`        // 命中的现有题目 ID
	PaperID         uint64  `json:"paper_id"`          // 命中题目所在试卷
	PaperName       string  `json:"paper_name"`        // 试卷名（方便 admin 在 UI 上识别）
	Similarity      float64 `json:"similarity"`        // Jaccard 相似度
	NewSnippet      string  `json:"new_snippet"`       // 新题题干前 N 字符
	ExistingSnippet string  `json:"existing_snippet"`  // 现有题题干前 N 字符
}

// existingDoc 是同课已有题目的预处理表示（rune 归一化 + n-gram 集合 + 展示片段）。
type existingDoc struct {
	id        uint64
	paperID   uint64
	paperName string
	ngrams    map[string]struct{}
	snippet   string
}

// FindSimilarProblems 在指定课程的所有现有题目里找与 newQuestions 相似的项。
//
// 设计要点：
//   - 一次性把课程下所有 problem 加载到内存（同课题量通常 < 几千，O(n) 可接受）
//   - 对每道现有题预先归一化 + 切 n-gram，避免内层循环重复计算
//   - 对每道新题：算它自己的 n-gram，遍历现有题集合算 Jaccard
//
// 没有 LLM 调用、零外部依赖，适合放进 worker 的 clean pipeline 同步执行。
func FindSimilarProblems(
	db *gorm.DB,
	courseID string,
	newQuestions []map[string]any,
) ([]DedupMatch, error) {
	if db == nil || courseID == "" || len(newQuestions) == 0 {
		return nil, nil
	}

	// 1) 加载课程下所有 problem + 关联试卷名 — 用 Raw SQL 避免 GORM Table+Joins+Scan 在不同版本的兼容坑
	type problemRow struct {
		ID        uint64
		Test      string
		PaperID   uint64
		PaperName string
	}
	var rows []problemRow
	err := db.Raw(`
		SELECT problems.id AS id,
		       problems.test AS test,
		       problems.testpaper_id AS paper_id,
		       test_papers.name AS paper_name
		FROM problems
		JOIN test_papers ON test_papers.id = problems.testpaper_id
		WHERE test_papers.course_id = ?
	`, courseID).Scan(&rows).Error
	if err != nil {
		log.Printf("dedup: load existing problems failed for course=%s: %v", courseID, err)
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "load existing problems failed: "+err.Error())
	}
	log.Printf("dedup: course=%s existing_problems=%d new_questions=%d", courseID, len(rows), len(newQuestions))
	if len(rows) == 0 {
		return nil, nil
	}

	// 2) 预处理：归一化 + 切 n-gram
	existing := make([]existingDoc, 0, len(rows))
	for _, r := range rows {
		grams := ngramSet(normalizeForDedup(r.Test), dedupNGramSize)
		if len(grams) == 0 {
			continue
		}
		existing = append(existing, existingDoc{
			id:        r.ID,
			paperID:   r.PaperID,
			paperName: r.PaperName,
			ngrams:    grams,
			snippet:   snippet(r.Test, dedupSnippetChars),
		})
	}

	// 3) 对每道新题查相似
	type candidate struct {
		sim float64
		doc *existingDoc
	}
	matches := make([]DedupMatch, 0)
	for _, q := range newQuestions {
		test, _ := q["test"].(string)
		if strings.TrimSpace(test) == "" {
			continue
		}
		newGrams := ngramSet(normalizeForDedup(test), dedupNGramSize)
		if len(newGrams) == 0 {
			continue
		}

		seq := 0
		if v, ok := numberValue(q["sequence_id"]); ok {
			seq = int(v)
		}

		// 在 existing 里找 >= 阈值的，按 sim 降序排，取前 K
		var cands []candidate
		for i := range existing {
			sim := jaccard(newGrams, existing[i].ngrams)
			if sim >= dedupThreshold {
				cands = append(cands, candidate{sim: sim, doc: &existing[i]})
			}
		}
		sort.Slice(cands, func(i, j int) bool { return cands[i].sim > cands[j].sim })
		if len(cands) > dedupMaxMatches {
			cands = cands[:dedupMaxMatches]
		}

		newSnip := snippet(test, dedupSnippetChars)
		for _, c := range cands {
			matches = append(matches, DedupMatch{
				NewSeq:          seq,
				ProblemID:       c.doc.id,
				PaperID:         c.doc.paperID,
				PaperName:       c.doc.paperName,
				Similarity:      roundTo3(c.sim),
				NewSnippet:      newSnip,
				ExistingSnippet: c.doc.snippet,
			})
		}
	}
	log.Printf("dedup: course=%s matched=%d", courseID, len(matches))
	return matches, nil
}

/* ─────────────────────────── 内部工具 ─────────────────────────── */

// normalizeForDedup 把题干变成"只保留有意义内容"的小写串：
//   - 删除空白
//   - 删除常见标点（中英文）
//   - 删除 LaTeX 包围符和 markdown 装饰，让公式骨架可比较
//   - 全角字母数字转半角
//   - 英文统一小写
func normalizeForDedup(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		if isJunkPunct(r) {
			continue
		}
		// 全角字母数字转半角
		switch {
		case r >= 0xFF21 && r <= 0xFF3A: // 全角 A-Z
			r -= 0xFEE0
		case r >= 0xFF41 && r <= 0xFF5A: // 全角 a-z
			r -= 0xFEE0
		case r >= 0xFF10 && r <= 0xFF19: // 全角 0-9
			r -= 0xFEE0
		}
		// 英文小写化
		if r >= 'A' && r <= 'Z' {
			r += 32
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isJunkPunct(r rune) bool {
	switch r {
	// LaTeX / Markdown / ASCII 标点
	case '$', '\\', '{', '}', '(', ')', '[', ']', '<', '>',
		',', '.', ';', ':', '?', '!', '"', '\'', '`', '~',
		'*', '_', '#', '|', '/', '-', '+', '=', '&', '^', '@':
		return true
	// 中文标点
	case '，', '。', '；', '：', '？', '！',
		'“', '”', '‘', '’', // 左/右 双引号 / 左/右 单引号
		'（', '）', '【', '】', '《', '》', '、', '·', '…':
		return true
	}
	return false
}

// ngramSet 返回字符串的 n-gram 集合（重复 gram 只保留一份）。
// 用 rune 切片避免在中文字符上切坏字节。
func ngramSet(s string, n int) map[string]struct{} {
	runes := []rune(s)
	if len(runes) < n {
		if len(runes) == 0 {
			return nil
		}
		// 短串直接整串当作一个 gram，避免完全没 gram
		return map[string]struct{}{string(runes): {}}
	}
	set := make(map[string]struct{}, len(runes)-n+1)
	for i := 0; i+n <= len(runes); i++ {
		set[string(runes[i:i+n])] = struct{}{}
	}
	return set
}

// jaccard 返回两个集合的 Jaccard 相似度 = |A∩B| / |A∪B|。
func jaccard(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	if len(a) > len(b) {
		a, b = b, a
	}
	inter := 0
	for k := range a {
		if _, ok := b[k]; ok {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// snippet 取字符串前 n 个 rune，超长加省略号。用于 UI 展示对比片段。
func snippet(s string, n int) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

func roundTo3(v float64) float64 {
	return float64(int(v*1000+0.5)) / 1000
}
