package services

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"warehouse-web/models"
)

// ProblemExplanationService 提供题解查询、编辑与投票能力。
type ProblemExplanationService struct {
	db    *gorm.DB
	paper *PaperService
}

type UpsertProblemExplanationRequest struct {
	ContentMD string `json:"content_md"`
}

type VoteProblemExplanationRequest struct {
	Value int `json:"value"`
}

func NewProblemExplanationService(db *gorm.DB, paper *PaperService) *ProblemExplanationService {
	return &ProblemExplanationService{db: db, paper: paper}
}

func (s *ProblemExplanationService) ListProblemExplanations(problemID uint64, viewerID *models.PrimaryKey) (*models.ProblemExplanationListData, error) {
	problem, err := s.paper.GetProblem(problemID)
	if err != nil {
		return nil, err
	}

	var rows []models.ProblemExplanation
	if err := s.db.Where("problem_id = ?", problemID).
		Order("score DESC").
		Order("updated_at DESC").
		Limit(3).
		Find(&rows).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load explanations")
	}

	items, err := s.inflateExplanationItems(rows, viewerID)
	if err != nil {
		return nil, err
	}

	var myItem *models.ProblemExplanationItem
	if viewerID != nil {
		var mine models.ProblemExplanation
		if err := s.db.Where("problem_id = ? AND user_id = ?", problemID, *viewerID).First(&mine).Error; err == nil {
			if !containsExplanation(items, mine.ID) {
				enriched, enrichErr := s.inflateExplanationItems([]models.ProblemExplanation{mine}, viewerID)
				if enrichErr != nil {
					return nil, enrichErr
				}
				if len(enriched) == 1 {
					myItem = &enriched[0]
				}
			}
		} else if err != nil && err != gorm.ErrRecordNotFound {
			return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load my explanation")
		}
	}

	return &models.ProblemExplanationListData{
		OfficialExplanation: problem.Explanation,
		Items:               items,
		MyItem:              myItem,
	}, nil
}

func (s *ProblemExplanationService) UpsertProblemExplanation(problemID uint64, userID models.PrimaryKey, req UpsertProblemExplanationRequest) (*models.ProblemExplanationItem, error) {
	content := strings.TrimSpace(req.ContentMD)
	if content == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "content_md 不能为空")
	}
	if _, err := s.paper.GetProblem(problemID); err != nil {
		return nil, err
	}

	var explanation models.ProblemExplanation
	err := s.db.Where("problem_id = ? AND user_id = ?", problemID, userID).First(&explanation).Error
	now := time.Now().UTC()
	if err == nil {
		explanation.ContentMD = content
		explanation.UpdatedAt = now
		if saveErr := s.db.Save(&explanation).Error; saveErr != nil {
			return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to update explanation")
		}
	} else if err == gorm.ErrRecordNotFound {
		explanation = models.ProblemExplanation{
			ProblemID: problemID,
			UserID:    userID,
			ContentMD: content,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if createErr := s.db.Create(&explanation).Error; createErr != nil {
			return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to create explanation")
		}
	} else {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load explanation")
	}

	items, enrichErr := s.inflateExplanationItems([]models.ProblemExplanation{explanation}, &userID)
	if enrichErr != nil {
		return nil, enrichErr
	}
	return &items[0], nil
}

func (s *ProblemExplanationService) UpdateProblemExplanation(problemID, explanationID uint64, userID models.PrimaryKey, req UpsertProblemExplanationRequest) (*models.ProblemExplanationItem, error) {
	content := strings.TrimSpace(req.ContentMD)
	if content == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "content_md 不能为空")
	}

	var explanation models.ProblemExplanation
	if err := s.db.Where("id = ? AND problem_id = ?", explanationID, problemID).First(&explanation).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, newServiceError("not_found", http.StatusNotFound, "explanation not found")
		}
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load explanation")
	}
	if explanation.UserID != userID {
		return nil, newServiceError("forbidden", http.StatusForbidden, "只能编辑自己的题解")
	}

	explanation.ContentMD = content
	explanation.UpdatedAt = time.Now().UTC()
	if err := s.db.Save(&explanation).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to update explanation")
	}
	items, enrichErr := s.inflateExplanationItems([]models.ProblemExplanation{explanation}, &userID)
	if enrichErr != nil {
		return nil, enrichErr
	}
	return &items[0], nil
}

func (s *ProblemExplanationService) VoteProblemExplanation(problemID, explanationID uint64, userID models.PrimaryKey, value int) (*models.ProblemExplanationItem, error) {
	if value != -1 && value != 0 && value != 1 {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "value 只能是 -1 / 0 / 1")
	}

	var explanation models.ProblemExplanation
	if err := s.db.Where("id = ? AND problem_id = ?", explanationID, problemID).First(&explanation).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, newServiceError("not_found", http.StatusNotFound, "explanation not found")
		}
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load explanation")
	}
	if explanation.UserID == userID {
		return nil, newServiceError("forbidden", http.StatusForbidden, "不能给自己的题解投票")
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var vote models.ProblemExplanationVote
		oldValue := 0
		if loadErr := tx.Where("explanation_id = ? AND user_id = ?", explanationID, userID).First(&vote).Error; loadErr == nil {
			oldValue = vote.Value
		} else if loadErr != nil && loadErr != gorm.ErrRecordNotFound {
			return newServiceError("internal_error", http.StatusInternalServerError, "failed to load vote")
		}

		applyVoteDelta(&explanation, oldValue, value)
		explanation.UpdatedAt = time.Now().UTC()
		if saveErr := tx.Save(&explanation).Error; saveErr != nil {
			return newServiceError("internal_error", http.StatusInternalServerError, "failed to update vote counters")
		}

		switch {
		case oldValue == 0 && value == 0:
			return nil
		case value == 0:
			if deleteErr := tx.Where("explanation_id = ? AND user_id = ?", explanationID, userID).Delete(&models.ProblemExplanationVote{}).Error; deleteErr != nil {
				return newServiceError("internal_error", http.StatusInternalServerError, "failed to remove vote")
			}
		case oldValue == 0:
			vote = models.ProblemExplanationVote{
				ExplanationID: explanationID,
				UserID:        userID,
				Value:         value,
			}
			if createErr := tx.Create(&vote).Error; createErr != nil {
				return newServiceError("internal_error", http.StatusInternalServerError, "failed to create vote")
			}
		default:
			if updateErr := tx.Model(&models.ProblemExplanationVote{}).
				Where("explanation_id = ? AND user_id = ?", explanationID, userID).
				Update("value", value).Error; updateErr != nil {
				return newServiceError("internal_error", http.StatusInternalServerError, "failed to update vote")
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	items, enrichErr := s.inflateExplanationItems([]models.ProblemExplanation{explanation}, &userID)
	if enrichErr != nil {
		return nil, enrichErr
	}
	return &items[0], nil
}

func (s *ProblemExplanationService) inflateExplanationItems(rows []models.ProblemExplanation, viewerID *models.PrimaryKey) ([]models.ProblemExplanationItem, error) {
	if len(rows) == 0 {
		return []models.ProblemExplanationItem{}, nil
	}

	userIDs := make([]uint64, 0, len(rows))
	for _, row := range rows {
		userIDs = append(userIDs, row.UserID)
	}

	var users []models.User
	if err := s.db.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load users")
	}
	var profiles []models.UserProfile
	if err := s.db.Where("user_id IN ?", userIDs).Find(&profiles).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load profiles")
	}

	userMap := make(map[uint64]models.User, len(users))
	for _, user := range users {
		userMap[user.ID] = user
	}
	profileMap := make(map[uint64]models.UserProfile, len(profiles))
	for _, profile := range profiles {
		profileMap[profile.UserID] = profile
	}

	voteMap := map[uint64]int{}
	if viewerID != nil {
		var votes []models.ProblemExplanationVote
		rowIDs := make([]uint64, 0, len(rows))
		for _, row := range rows {
			rowIDs = append(rowIDs, row.ID)
		}
		if err := s.db.Where("user_id = ? AND explanation_id IN ?", *viewerID, rowIDs).Find(&votes).Error; err != nil {
			return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load votes")
		}
		for _, vote := range votes {
			voteMap[vote.ExplanationID] = vote.Value
		}
	}

	items := make([]models.ProblemExplanationItem, 0, len(rows))
	for _, row := range rows {
		authorName := userMap[row.UserID].Username
		if profile, ok := profileMap[row.UserID]; ok && strings.TrimSpace(profile.Nickname) != "" {
			authorName = profile.Nickname
		}
		item := models.ProblemExplanationItem{
			ID:         row.ID,
			ProblemID:  row.ProblemID,
			AuthorID:   row.UserID,
			AuthorName: authorName,
			ContentMD:  row.ContentMD,
			UpVotes:    row.UpVotes,
			DownVotes:  row.DownVotes,
			MyVote:     voteMap[row.ID],
			CanEdit:    viewerID != nil && row.UserID == *viewerID,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		}
		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		leftScore := items[i].UpVotes - items[i].DownVotes
		rightScore := items[j].UpVotes - items[j].DownVotes
		if leftScore == rightScore {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		return leftScore > rightScore
	})
	return items, nil
}

func containsExplanation(items []models.ProblemExplanationItem, explanationID uint64) bool {
	for _, item := range items {
		if item.ID == explanationID {
			return true
		}
	}
	return false
}

func applyVoteDelta(explanation *models.ProblemExplanation, oldValue, newValue int) {
	switch oldValue {
	case 1:
		explanation.UpVotes--
	case -1:
		explanation.DownVotes--
	}
	switch newValue {
	case 1:
		explanation.UpVotes++
	case -1:
		explanation.DownVotes++
	}
	explanation.Score = explanation.UpVotes - explanation.DownVotes
}
