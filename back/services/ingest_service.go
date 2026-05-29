package services

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"warehouse-web/models"
)

// IngestService 串联文件上传 → LLM 清洗 → 审核 → 入库 全流程。
type IngestService struct {
	db      *gorm.DB
	llm     *LLMService
	vision  *VisionClient
	paper   *PaperService
}

func NewIngestService(db *gorm.DB, llm *LLMService, vision *VisionClient, paper *PaperService) *IngestService {
	return &IngestService{db: db, llm: llm, vision: vision, paper: paper}
}

// ───────────────────────── 上传创建 ─────────────────────────

// CreateJobInput 是创建上传任务的入参，由 controller 把 multipart 解析后填好。
type CreateJobInput struct {
	UserID        models.PrimaryKey
	Kind          string  // "question" / "explanation"
	CourseID      string  // 已有课程
	NewCourseName string  // 用户选"新增课程"时填
	PaperName     string  // 兼容老前端，自由文本试卷名（结构化字段为空时使用）
	TargetPaperID *uint64 // 题解流程必填

	// 结构化试卷命名 — 题目流程优先；三段都填后会自动合并到 (course, year, semester, exam_type) 相同的现有试卷。
	Year     int
	Semester string
	ExamType string

	Filename    string
	StoragePath string
	Mime        string
	Size        int64
}

// CreateJob 落库一条 pending 任务，等 worker 拉起。
func (s *IngestService) CreateJob(in CreateJobInput) (*models.IngestJob, error) {
	if in.Kind != models.IngestKindQuestion && in.Kind != models.IngestKindExplanation {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "kind 只能是 question 或 explanation")
	}
	if strings.TrimSpace(in.CourseID) == "" && strings.TrimSpace(in.NewCourseName) == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "请选择课程或填写新课程名")
	}

	// 题目流程：要么填了结构化三段（推荐路径），要么填了自由 PaperName（旧路径，仅兼容）
	structuredOK := in.Year > 0 && strings.TrimSpace(in.Semester) != "" && strings.TrimSpace(in.ExamType) != ""
	if in.Kind == models.IngestKindQuestion && !structuredOK && strings.TrimSpace(in.PaperName) == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "请填写试卷年份、学期和考试类型")
	}
	if in.Kind == models.IngestKindExplanation && (in.TargetPaperID == nil || *in.TargetPaperID == 0) {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "题解上传必须指定 target_paper_id")
	}

	// 题目流程：若结构化三段填了，自动拼成 PaperName 作为显示名（admin 审核时仍可改）
	paperName := strings.TrimSpace(in.PaperName)
	if in.Kind == models.IngestKindQuestion && structuredOK {
		paperName = composePaperName(in.Year, in.Semester, in.ExamType)
	}

	job := &models.IngestJob{
		UserID:        in.UserID,
		Kind:          in.Kind,
		CourseID:      strings.TrimSpace(in.CourseID),
		NewCourseName: strings.TrimSpace(in.NewCourseName),
		PaperName:     paperName,
		TargetPaperID: in.TargetPaperID,
		Year:          in.Year,
		Semester:      strings.TrimSpace(in.Semester),
		ExamType:      strings.TrimSpace(in.ExamType),
		Filename:      in.Filename,
		StoragePath:   in.StoragePath,
		Mime:          in.Mime,
		Size:          in.Size,
		Status:        models.IngestStatusPending,
	}
	if err := s.db.Create(job).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "创建上传任务失败")
	}
	return job, nil
}

// composePaperName 按 "{year} {semester}{examType}" 规则拼显示名，如 "2024 秋冬期末"。
func composePaperName(year int, semester, examType string) string {
	semester = strings.TrimSpace(semester)
	examType = strings.TrimSpace(examType)
	if year > 0 && semester != "" && examType != "" {
		return strconv.Itoa(year) + " " + semester + examType
	}
	return strings.TrimSpace(semester + examType)
}

// ───────────────────────── 查询 ─────────────────────────

func (s *IngestService) ListMyJobs(userID models.PrimaryKey, status string, page, size int) ([]models.IngestJob, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	q := s.db.Model(&models.IngestJob{}).Where("user_id = ?", userID)
	if status != "" {
		q = q.Where("status = ?", status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, newServiceError("internal_error", http.StatusInternalServerError, "查询失败")
	}

	var list []models.IngestJob
	if err := q.Order("id DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&list).Error; err != nil {
		return nil, 0, newServiceError("internal_error", http.StatusInternalServerError, "查询失败")
	}
	return list, total, nil
}

func (s *IngestService) GetJob(id uint64) (*models.IngestJob, error) {
	var job models.IngestJob
	if err := s.db.First(&job, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, newServiceError("not_found", http.StatusNotFound, "任务不存在")
		}
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "查询失败")
	}
	return &job, nil
}

func (s *IngestService) AdminListJobs(status string, page, size int) ([]models.IngestJob, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	q := s.db.Model(&models.IngestJob{})
	if status != "" {
		q = q.Where("status = ?", status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, newServiceError("internal_error", http.StatusInternalServerError, "查询失败")
	}

	var list []models.IngestJob
	if err := q.Order("id DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&list).Error; err != nil {
		return nil, 0, newServiceError("internal_error", http.StatusInternalServerError, "查询失败")
	}
	return list, total, nil
}

// ───────────────────────── 审核期编辑 ─────────────────────────

// AdminUpdateInput 允许 admin 在审核阶段调整：
//   - 绑定的课程 / 试卷
//   - 结构化命名三段（修改后会自动重算 PaperName）
//   - 解析后的 JSON
type AdminUpdateInput struct {
	CourseID      *string
	NewCourseName *string
	PaperName     *string // 老路径：自由文本
	Year          *int
	Semester      *string
	ExamType      *string
	TargetPaperID *uint64
	ParsedJSON    *json.RawMessage // 仅当本次想覆盖时传
}

func (s *IngestService) AdminUpdateJob(id uint64, in AdminUpdateInput) (*models.IngestJob, error) {
	var job models.IngestJob
	if err := s.db.First(&job, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, newServiceError("not_found", http.StatusNotFound, "任务不存在")
		}
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "查询失败")
	}
	if job.Status != models.IngestStatusAwaitingReview {
		return nil, newServiceError(
			"invalid_state",
			http.StatusConflict,
			"仅 awaiting_review 状态可编辑，当前: "+job.Status,
		)
	}

	if in.CourseID != nil {
		job.CourseID = strings.TrimSpace(*in.CourseID)
	}
	if in.NewCourseName != nil {
		job.NewCourseName = strings.TrimSpace(*in.NewCourseName)
	}
	if in.Year != nil {
		job.Year = *in.Year
	}
	if in.Semester != nil {
		job.Semester = strings.TrimSpace(*in.Semester)
	}
	if in.ExamType != nil {
		job.ExamType = strings.TrimSpace(*in.ExamType)
	}
	// PaperName 的处理：admin 若直接传了 paper_name 用它；否则若结构化三段齐全，重算
	if in.PaperName != nil {
		job.PaperName = strings.TrimSpace(*in.PaperName)
	} else if job.Year > 0 && job.Semester != "" && job.ExamType != "" {
		job.PaperName = composePaperName(job.Year, job.Semester, job.ExamType)
	}
	if in.TargetPaperID != nil {
		job.TargetPaperID = in.TargetPaperID
	}
	if in.ParsedJSON != nil {
		var probe any
		if err := json.Unmarshal(*in.ParsedJSON, &probe); err != nil {
			return nil, newServiceError("invalid_request", http.StatusBadRequest, "parsed_json 不是合法 JSON")
		}
		job.ParsedJSON = datatypes.JSON(*in.ParsedJSON)
	}

	if err := s.db.Save(&job).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "保存失败")
	}
	return &job, nil
}

// ───────────────────────── 拒绝 ─────────────────────────

func (s *IngestService) RejectJob(id uint64, reviewerID models.PrimaryKey, reason string) (*models.IngestJob, error) {
	var job models.IngestJob
	if err := s.db.First(&job, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, newServiceError("not_found", http.StatusNotFound, "任务不存在")
		}
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "查询失败")
	}
	if job.Status != models.IngestStatusAwaitingReview {
		return nil, newServiceError("invalid_state", http.StatusConflict, "仅 awaiting_review 状态可拒绝")
	}

	now := time.Now().UTC()
	job.Status = models.IngestStatusRejected
	job.ReviewerID = &reviewerID
	job.ReviewedAt = &now
	if strings.TrimSpace(reason) != "" {
		job.ErrorMessage = reason
	}
	if err := s.db.Save(&job).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "保存失败")
	}
	return &job, nil
}

// ───────────────────────── 发布入库 ─────────────────────────

// PublishJob 把审核通过的 IngestJob 落地到正式题库：
//   - kind=question:     新建 TestPaper + 一批 Problem
//   - kind=explanation:  按 sequence_id 将 content_md 写入 Problem.Explanation
//
// 整个过程在事务里完成；失败回滚不改 Status。
func (s *IngestService) PublishJob(id uint64, reviewerID models.PrimaryKey) (*models.IngestJob, error) {
	var job models.IngestJob
	if err := s.db.First(&job, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, newServiceError("not_found", http.StatusNotFound, "任务不存在")
		}
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "查询失败")
	}
	if job.Status != models.IngestStatusAwaitingReview {
		return nil, newServiceError("invalid_state", http.StatusConflict, "仅 awaiting_review 状态可发布")
	}
	if len(job.ParsedJSON) == 0 {
		return nil, newServiceError("invalid_state", http.StatusBadRequest, "解析结果为空，不能发布")
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1) 确认课程：若 admin 没填 CourseID，但填了 NewCourseName，则先创建/复用 course
		courseID := job.CourseID
		if courseID == "" {
			if job.NewCourseName == "" {
				return newServiceError("invalid_state", http.StatusBadRequest, "请先选择或创建课程后再发布")
			}
			cid, err := ensureCourseByName(tx, job.NewCourseName)
			if err != nil {
				return err
			}
			courseID = cid
			job.CourseID = cid
		}

		// 2) 按 kind 分发
		switch job.Kind {
		case models.IngestKindQuestion:
			if err := publishQuestions(tx, &job, courseID); err != nil {
				return err
			}
		case models.IngestKindExplanation:
			if err := publishExplanations(tx, &job); err != nil {
				return err
			}
		default:
			return newServiceError("invalid_state", http.StatusBadRequest, "未知 kind: "+job.Kind)
		}

		// 3) 更新任务状态
		now := time.Now().UTC()
		job.Status = models.IngestStatusPublished
		job.ReviewerID = &reviewerID
		job.ReviewedAt = &now
		job.PublishedAt = &now
		return tx.Save(&job).Error
	})
	if err != nil {
		if _, ok := err.(*ServiceError); ok {
			return nil, err
		}
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "发布失败: "+err.Error())
	}
	return &job, nil
}

// ensureCourseByName 查找同名课程，没找到则创建。
// 课程主键是字符串 ID，用作 URL path 片段 —— 必须 ASCII 安全。
// 故采用 "course_<UTC timestamp>" 形式，不把中文/空格塞进 ID。
func ensureCourseByName(tx *gorm.DB, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", newServiceError("invalid_request", http.StatusBadRequest, "课程名为空")
	}

	var existing models.Course
	err := tx.Where("name = ?", name).First(&existing).Error
	if err == nil {
		return existing.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", newServiceError("internal_error", http.StatusInternalServerError, "查找课程失败")
	}

	// ID 用纯 ASCII 时间戳，避免 URL 编码问题；name 保留中文给用户看。
	courseID := "course_" + time.Now().UTC().Format("20060102150405")
	course := models.Course{
		ID:   courseID,
		Name: name,
	}
	if err := tx.Create(&course).Error; err != nil {
		return "", newServiceError("internal_error", http.StatusInternalServerError, "创建课程失败")
	}
	return courseID, nil
}

func publishQuestions(tx *gorm.DB, job *models.IngestJob, courseID string) error {
	if strings.TrimSpace(job.PaperName) == "" {
		return newServiceError("invalid_state", http.StatusBadRequest, "试卷名为空，不能发布")
	}

	items, err := decodeItems(job.ParsedJSON)
	if err != nil {
		return err
	}
	if err := ValidateQuestionItems(items); err != nil {
		return newServiceError("invalid_state", http.StatusBadRequest, "题目校验失败: "+err.Error())
	}

	// 自动合并语义：结构化三段齐全时，找 (course, year, semester, exam_type) 完全匹配的现有 paper
	// 找到 → 复用、题目 append（Order 从现有最大值 +1 续起）
	// 找不到 → 新建 paper
	var paper models.TestPaper
	orderStart := 0
	if job.Year > 0 && job.Semester != "" && job.ExamType != "" {
		var existing models.TestPaper
		err := tx.Where(
			"course_id = ? AND year = ? AND semester = ? AND exam_type = ?",
			courseID, job.Year, job.Semester, job.ExamType,
		).First(&existing).Error
		if err == nil {
			paper = existing
			// 找出现有最大 Order，新题从其后开始排
			var maxOrder int
			tx.Model(&models.Problem{}).
				Where("testpaper_id = ?", existing.ID).
				Select("COALESCE(MAX(\"order\"), 0)").
				Scan(&maxOrder)
			orderStart = maxOrder
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return newServiceError("internal_error", http.StatusInternalServerError, "查找已有试卷失败: "+err.Error())
		}
	}
	if paper.ID == 0 {
		paper = models.TestPaper{
			CourseID:  courseID,
			Name:      job.PaperName,
			Year:      job.Year,
			Semester:  job.Semester,
			ExamType:  job.ExamType,
			CreatedAt: time.Now().UTC(),
		}
		if err := tx.Create(&paper).Error; err != nil {
			return newServiceError("internal_error", http.StatusInternalServerError, "创建试卷失败")
		}
	}

	for i, item := range items {
		options := []models.Option{}
		if rawOpts, ok := item["options"].([]any); ok {
			for _, raw := range rawOpts {
				if optMap, ok := raw.(map[string]any); ok {
					optChar, _ := optMap["option"].(string)
					optText, _ := optMap["text"].(string)
					options = append(options, models.Option{Option: optChar, Text: optText})
				}
			}
		}
		optionsBytes, _ := json.Marshal(options)

		tags := []string{}
		if rawTags, ok := item["tags"].([]any); ok {
			for _, t := range rawTags {
				if str, ok := t.(string); ok {
					tags = append(tags, str)
				}
			}
		}
		tagsBytes, _ := json.Marshal(tags)

		seq := i + 1
		if v, ok := numberValue(item["sequence_id"]); ok {
			seq = int(v)
		}

		problem := models.Problem{
			TestpaperID:  paper.ID,
			// SourceID 用 (job_id, filename, 循环序号) 确保同任务内 100% 唯一，
			// 避免 LLM 输出的 sequence_id 在同任务内撞车（如按题型独立编号时）。
			SourceID:   "ingest:" + jobSourceTag(job, i+1),
			Order:      orderStart + i + 1,
			SequenceID: seq,
			QuestionType: stringValue(item["question_type"]),
			Test:         stringValue(item["test"]),
			Answer:       stringValue(item["answer"]),
			Explanation:  stringValue(item["explanation"]),
			Difficulty:   stringValue(item["difficulty"]),
			OptionsJSON:  datatypes.JSON(optionsBytes),
			TagsJSON:     datatypes.JSON(tagsBytes),
			Status:       "processed",
		}
		if err := tx.Create(&problem).Error; err != nil {
			return newServiceError("internal_error", http.StatusInternalServerError, "插入题目失败: "+err.Error())
		}
	}
	return nil
}

func publishExplanations(tx *gorm.DB, job *models.IngestJob) error {
	if job.TargetPaperID == nil || *job.TargetPaperID == 0 {
		return newServiceError("invalid_state", http.StatusBadRequest, "题解上传缺少 target_paper_id")
	}

	items, err := decodeItems(job.ParsedJSON)
	if err != nil {
		return err
	}
	if err := ValidateExplanationItems(items); err != nil {
		return newServiceError("invalid_state", http.StatusBadRequest, "题解校验失败: "+err.Error())
	}

	// 一次性把目标卷的题目按 sequence_id 取出来，做映射，避免 N 次 SELECT。
	var problems []models.Problem
	if err := tx.Where("testpaper_id = ?", *job.TargetPaperID).Find(&problems).Error; err != nil {
		return newServiceError("internal_error", http.StatusInternalServerError, "加载目标卷题目失败")
	}
	bySeq := make(map[int]*models.Problem, len(problems))
	for i := range problems {
		bySeq[problems[i].SequenceID] = &problems[i]
	}

	for i, item := range items {
		seqVal, _ := numberValue(item["sequence_id"])
		seq := int(seqVal)
		content := stringValue(item["content_md"])
		problem, ok := bySeq[seq]
		if !ok {
			return newServiceError(
				"invalid_state",
				http.StatusBadRequest,
				"第 "+stringFromInt(i+1)+" 条题解的 sequence_id="+stringFromInt(seq)+" 在目标卷中无对应题目",
			)
		}
		problem.Explanation = content
		if err := tx.Save(problem).Error; err != nil {
			return newServiceError("internal_error", http.StatusInternalServerError, "写入官方题解失败: "+err.Error())
		}
	}
	return nil
}

// ───────────────────────── worker 处理 ─────────────────────────

// ProcessNextPending 由 worker 周期性调用：拉起一条 pending 任务，跑完一个生命周期。
// 用 SELECT ... FOR UPDATE SKIP LOCKED 风格，但 GORM 上简化为 update-where 占坑。
// 返回 true 表示本轮处理了一条任务，false 表示无可处理任务（worker 可以歇一下）。
func (s *IngestService) ProcessNextPending(ctx context.Context) bool {
	// 1) 用 UPDATE 抢占一条 pending 任务
	var job models.IngestJob
	tx := s.db.Where("status = ?", models.IngestStatusPending).
		Order("id ASC").
		Limit(1)
	if err := tx.First(&job).Error; err != nil {
		// 没有 pending 任务
		return false
	}

	// 抢占：把它改成 processing；并发场景下要求 status 仍是 pending 才生效。
	res := s.db.Model(&models.IngestJob{}).
		Where("id = ? AND status = ?", job.ID, models.IngestStatusPending).
		Updates(map[string]any{
			"status":     models.IngestStatusProcessing,
			"updated_at": time.Now().UTC(),
		})
	if res.Error != nil || res.RowsAffected == 0 {
		return false
	}

	// 2) 跑预处理 + LLM 清洗
	if err := s.runPipeline(ctx, &job); err != nil {
		log.Printf("ingest: job %d pipeline failed: %v", job.ID, err)
		s.markFailed(job.ID, err.Error())
		return true
	}
	return true
}

func (s *IngestService) runPipeline(ctx context.Context, job *models.IngestJob) error {
	// 2.1 提取文本
	text, err := ExtractTextFromFile(ctx, job.StoragePath, s.vision)
	if err != nil {
		return err
	}

	// 2.2 LLM 清洗
	var result *IngestCleanResult
	switch job.Kind {
	case models.IngestKindQuestion:
		result, err = CleanQuestionText(ctx, s.llm, text)
		if err == nil {
			err = ValidateQuestionItems(result.Items)
		}
	case models.IngestKindExplanation:
		result, err = CleanExplanationText(ctx, s.llm, text)
		if err == nil {
			err = ValidateExplanationItems(result.Items)
		}
	default:
		err = newServiceError("invalid_state", http.StatusBadRequest, "未知 kind: "+job.Kind)
	}
	if err != nil {
		return err
	}

	// 2.3 写回 awaiting_review
	parsedJSON := datatypes.JSON(result.RawJSON)
	// 若 LLM 直接给了顶层数组的兜底情况，统一存成 {"items":[...]} 让前端编辑更稳定
	if !strings.HasPrefix(strings.TrimSpace(result.RawJSON), "{") {
		envBytes, _ := json.Marshal(map[string]any{"items": result.Items})
		parsedJSON = datatypes.JSON(envBytes)
	}

	// 2.4 仅题目流程做查重；查重失败不阻塞主流程，仅记日志
	updates := map[string]any{
		"raw_text":    text,
		"parsed_json": parsedJSON,
		"llm_model":   result.Model,
		"status":      models.IngestStatusAwaitingReview,
		"updated_at":  time.Now().UTC(),
	}
	log.Printf("ingest: job %d kind=%s course_id=%q items=%d", job.ID, job.Kind, job.CourseID, len(result.Items))
	if job.Kind == models.IngestKindQuestion && job.CourseID != "" {
		matches, derr := FindSimilarProblems(s.db, job.CourseID, result.Items)
		if derr != nil {
			log.Printf("ingest: job %d dedup skipped: %v", job.ID, derr)
		} else if len(matches) > 0 {
			warnBytes, _ := json.Marshal(matches)
			updates["dedup_warnings"] = datatypes.JSON(warnBytes)
			log.Printf("ingest: job %d wrote %d dedup_warnings", job.ID, len(matches))
		} else {
			log.Printf("ingest: job %d dedup ran, 0 matches", job.ID)
		}
	} else {
		log.Printf("ingest: job %d dedup skipped (kind=%s course_id=%q)", job.ID, job.Kind, job.CourseID)
	}

	return s.db.Model(&models.IngestJob{}).
		Where("id = ?", job.ID).
		Updates(updates).Error
}

func (s *IngestService) markFailed(id uint64, msg string) {
	s.db.Model(&models.IngestJob{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        models.IngestStatusFailed,
			"error_message": msg,
			"updated_at":    time.Now().UTC(),
		})
}

// ───────────────────────── 工具 ─────────────────────────

func decodeItems(raw datatypes.JSON) ([]map[string]any, error) {
	if len(raw) == 0 {
		return nil, newServiceError("invalid_state", http.StatusBadRequest, "parsed_json 为空")
	}
	// 优先 envelope
	var envelope struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil && envelope.Items != nil {
		return envelope.Items, nil
	}
	// 兜底顶层数组
	var arr []map[string]any
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	return nil, newServiceError("invalid_state", http.StatusBadRequest, "parsed_json 结构非法")
}

func stringValue(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func numberValue(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

func stringFromInt(i int) string {
	return strconv.Itoa(i)
}

// jobSourceTag 构造 problems.source_id 用的稳定标记，确保同任务多次发布不会撞 source_id。
func jobSourceTag(job *models.IngestJob, seq int) string {
	base := filepath.Base(job.StoragePath)
	if strings.TrimSpace(base) == "" {
		base = "job"
	}
	return strconv.FormatUint(job.ID, 10) + ":" + base + ":" + strconv.Itoa(seq)
}
