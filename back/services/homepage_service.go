package services

import (
	"net/http"
	"strings"

	"gorm.io/gorm"

	"warehouse-web/models"
)

// HomepageService manages homepage message board data.
type HomepageService struct {
	db *gorm.DB
}

func NewHomepageService(db *gorm.DB) *HomepageService {
	return &HomepageService{db: db}
}

// ListMessages returns homepage messages ordered by newest first.
func (s *HomepageService) ListMessages() ([]models.HomepageMessage, error) {
	var items []models.HomepageMessage
	if err := s.db.Order("id DESC").Find(&items).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load homepage messages")
	}
	s.hydrateDisplayFields(items)
	return items, nil
}

// AddMessage creates one homepage message for a logged-in user.
func (s *HomepageService) AddMessage(userID uint64, content string) (*models.HomepageMessage, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "content 不能为空")
	}

	item := &models.HomepageMessage{
		UserID:  userID,
		Content: content,
	}
	if err := s.db.Create(item).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to create homepage message")
	}
	s.hydrateOne(item)
	return item, nil
}

func (s *HomepageService) hydrateOne(item *models.HomepageMessage) {
	if item == nil {
		return
	}
	items := []models.HomepageMessage{*item}
	s.hydrateDisplayFields(items)
	*item = items[0]
}

func (s *HomepageService) hydrateDisplayFields(items []models.HomepageMessage) {
	if len(items) == 0 {
		return
	}

	userIDs := make([]uint64, 0, len(items))
	for _, item := range items {
		userIDs = append(userIDs, item.UserID)
	}

	var profiles []models.UserProfile
	if err := s.db.Where("user_id IN ?", userIDs).Find(&profiles).Error; err != nil {
		return
	}
	profileMap := make(map[uint64]models.UserProfile, len(profiles))
	for _, profile := range profiles {
		profileMap[profile.UserID] = profile
	}

	var users []models.User
	if err := s.db.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return
	}
	userMap := make(map[uint64]models.User, len(users))
	for _, user := range users {
		userMap[user.ID] = user
	}

	for index := range items {
		user := userMap[items[index].UserID]
		profile, hasProfile := profileMap[items[index].UserID]
		if hasProfile && strings.TrimSpace(profile.Nickname) != "" {
			items[index].UserName = profile.Nickname
		} else {
			items[index].UserName = user.Username
		}
		if hasProfile {
			items[index].AvatarURL = profile.AvatarURL
		}
		if strings.TrimSpace(items[index].UserName) == "" {
			items[index].UserName = "匿名同学"
		}
	}
}
