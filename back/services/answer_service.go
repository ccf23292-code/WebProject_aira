package services

import (
	"net/http"
	"time"

	"gorm.io/gorm"

	"warehouse-web/models"
)

// AnswerService 记录做题记录并维护错题。
type AnswerService struct {
	db    *gorm.DB
	paper *PaperService
}

func NewAnswerService(db *gorm.DB, paper *PaperService) *AnswerService {
	return &AnswerService{db: db, paper: paper}
}

type AnswerRecordRequest struct {
	PaperID        uint64 `json:"paper_id"`
	ProblemID      uint64 `json:"problem_id"`
	SelectedOption string `json:"selected_option"`
	IsCorrect      bool   `json:"is_correct"`
	Mode           string `json:"mode"`
}

type AnswerBatchRecordRequest struct {
	Answers []AnswerRecordRequest `json:"answers"`
}

// RecordAnswer 写入做题记录，若答错则更新错题本。
func (s *AnswerService) RecordAnswer(userID uint64, req AnswerRecordRequest) error {
	return s.recordAnswer(userID, req)
}

// RecordAnswersBatch 批量写入做题记录，用于模拟考交卷。
func (s *AnswerService) RecordAnswersBatch(userID uint64, answers []AnswerRecordRequest) error {
	if len(answers) == 0 {
		return newServiceError("invalid_request", http.StatusBadRequest, "answers 不能为空")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		scoped := &AnswerService{db: tx, paper: s.paper}
		for _, answer := range answers {
			if err := scoped.recordAnswer(userID, answer); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *AnswerService) recordAnswer(userID uint64, req AnswerRecordRequest) error {
	problem, err := s.paper.GetProblem(req.ProblemID)
	if err != nil || problem == nil {
		return newServiceError("not_found", http.StatusNotFound, "problem not found")
	}
	paper, err := s.paper.GetPaper(req.PaperID)
	if err != nil || paper == nil {
		return newServiceError("not_found", http.StatusNotFound, "paper not found")
	}
	courseID := paper.CourseID

	record := models.AnswerRecord{
		UserID:         userID,
		CourseID:       courseID,
		PaperID:        req.PaperID,
		ProblemID:      req.ProblemID,
		SelectedOption: req.SelectedOption,
		IsCorrect:      req.IsCorrect,
		Mode:           req.Mode,
		AnsweredAt:     time.Now().UTC(),
	}
	if err := s.db.Create(&record).Error; err != nil {
		return newServiceError("internal_error", http.StatusInternalServerError, "failed to record answer")
	}

	if !req.IsCorrect {
		if err := s.upsertWrongQuestion(userID, courseID, req.PaperID, req.ProblemID); err != nil {
			return err
		}
	}

	return nil
}

func (s *AnswerService) upsertWrongQuestion(userID uint64, courseID string, paperID, problemID uint64) error {
	var wrong models.WrongQuestion
	err := s.db.Where("user_id = ? AND problem_id = ?", userID, problemID).First(&wrong).Error
	if err == nil {
		wrong.WrongCount += 1
		wrong.LastWrongAt = time.Now().UTC()
		if err := s.db.Save(&wrong).Error; err != nil {
			return newServiceError("internal_error", http.StatusInternalServerError, "failed to update wrong question")
		}
		return nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return newServiceError("internal_error", http.StatusInternalServerError, "failed to update wrong question")
	}

	wrong = models.WrongQuestion{
		UserID:      userID,
		ProblemID:   problemID,
		CourseID:    courseID,
		PaperID:     paperID,
		Status:      models.WrongStatusUnmastered,
		WrongCount:  1,
		LastWrongAt: time.Now().UTC(),
	}
	if err := s.db.Create(&wrong).Error; err != nil {
		return newServiceError("internal_error", http.StatusInternalServerError, "failed to create wrong question")
	}
	return nil
}

// ListAnswerRecords 返回做题记录（按时间倒序）。
func (s *AnswerService) ListAnswerRecords(userID uint64, page, size int) ([]models.AnswerRecord, int, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}

	var total int64
	if err := s.db.Model(&models.AnswerRecord{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, newServiceError("internal_error", http.StatusInternalServerError, "failed to load records")
	}

	var records []models.AnswerRecord
	if err := s.db.Where("user_id = ?", userID).
		Order("answered_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&records).Error; err != nil {
		return nil, 0, newServiceError("internal_error", http.StatusInternalServerError, "failed to load records")
	}

	return records, int(total), nil
}
