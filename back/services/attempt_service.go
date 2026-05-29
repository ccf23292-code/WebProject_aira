package services

import (
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"warehouse-web/models"
)

// AttemptService 提供"严格练习模式"下的尝试与提交相关业务。
type AttemptService struct {
	db    *gorm.DB
	paper *PaperService
}

// NewAttemptService 创建 AttemptService。
func NewAttemptService(db *gorm.DB, paper *PaperService) *AttemptService {
	return &AttemptService{db: db, paper: paper}
}

// CreateAttemptResult 是 CreateAttempt 的返回结构。
// Created 字段告诉调用方本次是新建还是复用：
//   - true  → 新建 attempt（旧 in_progress 被 abandon，或本来没有）
//   - false → 复用已有的 in_progress（默认行为下"恢复进度"）
type CreateAttemptResult struct {
	Attempt *models.PaperAttempt `json:"attempt"`
	Total   int                  `json:"total"`
	Created bool                 `json:"created"`
}

// AttemptDetail 是 GetAttempt 的返回结构。
type AttemptDetail struct {
	Attempt     *models.PaperAttempt        `json:"attempt"`
	Submissions []models.ProblemSubmission  `json:"submissions"`
}

// SubmitProblemResult 是 SubmitProblem 的返回结构。
type SubmitProblemResult struct {
	Submission    *models.ProblemSubmission `json:"submission"`
	CorrectAnswer string                    `json:"correct_answer"`
	Attempt       *models.PaperAttempt      `json:"attempt"`
}

/* ════════════════════ CreateAttempt ════════════════════ */

// CreateAttempt 在指定试卷上获取或创建一次尝试。
//
// 默认行为（forceReset=false）：
//   - 若同 (user_id, paper_id) 下有 in_progress，直接复用最新一条（"恢复上次进度"）
//   - 否则新建一条
//
// 强制重置（forceReset=true）：
//   - 把旧 in_progress 全部 abandon，并新建一条
//
// 并发说明：MVP 阶段未加 PostgreSQL 部分唯一索引；极少数情况下两个并发
// 请求都进入 create 分支可能产生两条 in_progress。下次 get 时会按
// started_at DESC 取最近的一条，业务侧无明显影响。
func (s *AttemptService) CreateAttempt(userID models.PrimaryKey, paperID uint64, forceReset bool) (*CreateAttemptResult, error) {
	// ── 1) 默认模式下，先尝试复用现有 in_progress ─────────────
	if !forceReset {
		var existing models.PaperAttempt
		err := s.db.Where("user_id = ? AND paper_id = ? AND status = ?",
			userID, paperID, models.AttemptStatusInProgress).
			Order("started_at DESC").
			First(&existing).Error
		if err == nil {
			return &CreateAttemptResult{
				Attempt: &existing,
				Total:   existing.Total,
				Created: false,
			}, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to query in-progress attempt")
		}
		// fall through 到创建分支
	}

	// ── 2) 创建新 attempt ──────────────────────────────────
	paper, err := s.paper.GetPaper(paperID)
	if err != nil {
		return nil, err
	}
	if paper == nil {
		return nil, newServiceError("not_found", http.StatusNotFound, "paper not found")
	}

	// 试卷题目快照（用于 Total 和 MaxScore）
	var problems []models.Problem
	if err := s.db.Where("testpaper_id = ?", paperID).Find(&problems).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load paper problems")
	}
	total := len(problems)
	if total == 0 {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "paper has no problems")
	}

	maxScore := 0.0
	for _, p := range problems {
		if p.Score > 0 {
			maxScore += p.Score
		} else {
			maxScore += 1.0
		}
	}

	now := time.Now().UTC()
	attempt := models.PaperAttempt{
		UserID:    userID,
		PaperID:   paperID,
		CourseID:  paper.CourseID,
		Status:    models.AttemptStatusInProgress,
		MaxScore:  maxScore,
		Total:     total,
		StartedAt: now,
		UpdatedAt: now,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 强制重置时把旧 in_progress 全部 abandon；默认分支若走到这里说明没有 in_progress，
		// UPDATE 影响 0 行，无副作用，统一执行保持代码简单
		if err := tx.Model(&models.PaperAttempt{}).
			Where("user_id = ? AND paper_id = ? AND status = ?", userID, paperID, models.AttemptStatusInProgress).
			Updates(map[string]any{
				"status":     models.AttemptStatusAbandoned,
				"updated_at": now,
			}).Error; err != nil {
			return newServiceError("internal_error", http.StatusInternalServerError, "failed to abandon previous attempts")
		}

		if err := tx.Create(&attempt).Error; err != nil {
			return newServiceError("internal_error", http.StatusInternalServerError, "failed to create attempt")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &CreateAttemptResult{Attempt: &attempt, Total: total, Created: true}, nil
}

/* ════════════════════ GetAttempt ════════════════════ */

// GetAttempt 返回某次尝试的聚合状态 + 所有已提交记录。
// 校验 attempt 的归属：只有发起人可见。
func (s *AttemptService) GetAttempt(userID models.PrimaryKey, attemptID uint64) (*AttemptDetail, error) {
	var attempt models.PaperAttempt
	if err := s.db.Where("id = ?", attemptID).First(&attempt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, newServiceError("not_found", http.StatusNotFound, "attempt not found")
		}
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load attempt")
	}
	if attempt.UserID != userID {
		return nil, newServiceError("forbidden", http.StatusForbidden, "无权访问该 attempt")
	}

	var submissions []models.ProblemSubmission
	if err := s.db.Where("attempt_id = ?", attemptID).
		Order("submitted_at ASC").
		Find(&submissions).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load submissions")
	}

	return &AttemptDetail{Attempt: &attempt, Submissions: submissions}, nil
}

/* ════════════════════ SubmitProblem ════════════════════ */

// SubmitProblem 提交单道题目的答案。
// 流程：归属校验 → 服务端判题 → 插入 ProblemSubmission（唯一约束兜底重复提交）
//      → 更新 attempt 聚合 → 同步老的 AnswerRecord/WrongQuestion。
func (s *AttemptService) SubmitProblem(
	userID models.PrimaryKey,
	attemptID uint64,
	problemID uint64,
	userAnswer string,
) (*SubmitProblemResult, error) {
	problem, err := s.paper.GetProblem(problemID)
	if err != nil {
		return nil, err
	}
	if problem == nil {
		return nil, newServiceError("not_found", http.StatusNotFound, "problem not found")
	}

	isCorrect, score := gradeAnswer(problem, userAnswer)
	now := time.Now().UTC()
	var result SubmitProblemResult

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		// 锁定行，防止并发提交导致聚合错乱
		var attempt models.PaperAttempt
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", attemptID).
			First(&attempt).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return newServiceError("not_found", http.StatusNotFound, "attempt not found")
			}
			return newServiceError("internal_error", http.StatusInternalServerError, "failed to load attempt")
		}
		if attempt.UserID != userID {
			return newServiceError("forbidden", http.StatusForbidden, "无权访问该 attempt")
		}
		if attempt.Status != models.AttemptStatusInProgress {
			return newServiceError("attempt_closed", http.StatusConflict, "该 attempt 已完成或已放弃，不能再提交")
		}

		submission := models.ProblemSubmission{
			AttemptID:   attemptID,
			ProblemID:   problemID,
			UserID:      userID,
			UserAnswer:  userAnswer,
			IsCorrect:   isCorrect,
			Score:       score,
			SubmittedAt: now,
		}
		if err := tx.Create(&submission).Error; err != nil {
			if isDuplicateKeyError(err) {
				return newServiceError("already_submitted", http.StatusConflict, "该题目在本次尝试中已提交")
			}
			return newServiceError("internal_error", http.StatusInternalServerError, "failed to record submission")
		}

		// 更新 attempt 聚合
		attempt.Submitted += 1
		attempt.Score += score
		if isCorrect {
			attempt.Correct += 1
		}
		attempt.UpdatedAt = now
		if attempt.Submitted >= attempt.Total {
			attempt.Status = models.AttemptStatusCompleted
			attempt.CompletedAt = &now
		}
		if err := tx.Save(&attempt).Error; err != nil {
			return newServiceError("internal_error", http.StatusInternalServerError, "failed to update attempt")
		}

		// 同步老的 AnswerRecord：保持"做题记录"页继续工作
		if err := syncAnswerRecord(tx, userID, attempt.CourseID, attempt.PaperID, problemID, userAnswer, isCorrect, now); err != nil {
			return err
		}

		// 同步错题本（仅在答错时）
		if !isCorrect {
			if err := syncWrongQuestion(tx, userID, attempt.CourseID, attempt.PaperID, problemID, now); err != nil {
				return err
			}
		}

		result = SubmitProblemResult{
			Submission:    &submission,
			CorrectAnswer: strings.TrimSpace(problem.Answer),
			Attempt:       &attempt,
		}
		return nil
	})

	if txErr != nil {
		return nil, txErr
	}
	return &result, nil
}

/* ════════════════════ 判题辅助 ════════════════════ */

// gradeAnswer 根据题型自动判题，返回 (是否正确, 得分)。
func gradeAnswer(p *models.Problem, userAnswer string) (bool, float64) {
	correctRaw := strings.TrimSpace(p.Answer)
	qType := strings.TrimSpace(p.QuestionType)

	var isCorrect bool
	switch qType {
	case "singleChoice", "multiChoice", "trueOrFalse":
		isCorrect = normalizeChoiceAnswer(userAnswer) == normalizeChoiceAnswer(correctRaw)
	case "fillBlank":
		isCorrect = normalizeText(userAnswer) == normalizeText(correctRaw)
	case "shortAnswer":
		// 简答题暂不支持自动判题，统一判错并由教师/AI 后续介入
		isCorrect = false
	default:
		// 未知类型走宽松文本比对兜底
		isCorrect = normalizeText(userAnswer) == normalizeText(correctRaw)
	}

	score := 0.0
	if isCorrect {
		if p.Score > 0 {
			score = p.Score
		} else {
			score = 1.0
		}
	}
	return isCorrect, score
}

// normalizeChoiceAnswer 把选择题答案标准化：trim → upper → 去空 → 排序 → join。
// "a, c" 与 "C,A" 都会变成 "A,C"，判题不再受空格/大小写/顺序干扰。
func normalizeChoiceAnswer(s string) string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToUpper(p))
		if p != "" {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

// normalizeText 用于填空 / 兜底判题，去首尾空白 + 全小写。
func normalizeText(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

// isDuplicateKeyError 检测是否为唯一约束违反。
// GORM v1.25+ 支持 ErrDuplicatedKey；同时做 PG 错误字符串兜底，兼容老驱动。
func isDuplicateKeyError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "sqlstate 23505")
}

/* ════════════════════ 老数据同步 ════════════════════ */

// syncAnswerRecord 在事务内追加一条老的 AnswerRecord，保证旧"做题记录"页继续工作。
func syncAnswerRecord(
	tx *gorm.DB,
	userID models.PrimaryKey,
	courseID string,
	paperID uint64,
	problemID uint64,
	userAnswer string,
	isCorrect bool,
	at time.Time,
) error {
	// AnswerRecord.SelectedOption 列宽 16，长答案直接截断
	selected := userAnswer
	if len(selected) > 16 {
		selected = selected[:16]
	}

	record := models.AnswerRecord{
		UserID:         userID,
		CourseID:       courseID,
		PaperID:        paperID,
		ProblemID:      problemID,
		SelectedOption: selected,
		IsCorrect:      isCorrect,
		Mode:           "strict_practice",
		AnsweredAt:     at,
	}
	if err := tx.Create(&record).Error; err != nil {
		return newServiceError("internal_error", http.StatusInternalServerError, "failed to sync answer record")
	}
	return nil
}

// syncWrongQuestion 在事务内 upsert 错题本，复用老 answer_service 的语义。
func syncWrongQuestion(
	tx *gorm.DB,
	userID models.PrimaryKey,
	courseID string,
	paperID uint64,
	problemID uint64,
	at time.Time,
) error {
	var wrong models.WrongQuestion
	err := tx.Where("user_id = ? AND problem_id = ?", userID, problemID).First(&wrong).Error
	if err == nil {
		wrong.WrongCount += 1
		wrong.LastWrongAt = at
		if saveErr := tx.Save(&wrong).Error; saveErr != nil {
			return newServiceError("internal_error", http.StatusInternalServerError, "failed to update wrong question")
		}
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return newServiceError("internal_error", http.StatusInternalServerError, "failed to load wrong question")
	}

	wrong = models.WrongQuestion{
		UserID:      userID,
		ProblemID:   problemID,
		CourseID:    courseID,
		PaperID:     paperID,
		Status:      models.WrongStatusUnmastered,
		WrongCount:  1,
		LastWrongAt: at,
	}
	if err := tx.Create(&wrong).Error; err != nil {
		return newServiceError("internal_error", http.StatusInternalServerError, "failed to create wrong question")
	}
	return nil
}
