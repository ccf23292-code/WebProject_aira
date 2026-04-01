package services

import (
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"warehouse-web/models"
)

// ProfileService 提供用户资料管理。
type ProfileService struct {
	db *gorm.DB
}

func NewProfileService(db *gorm.DB) *ProfileService {
	return &ProfileService{db: db}
}

type ProfileUpdateRequest struct {
	Nickname string `json:"nickname"`
}

// GetProfile 返回用户资料。
func (s *ProfileService) GetProfile(userID uint64) (*models.UserProfile, error) {
	var user models.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, newServiceError("not_found", http.StatusNotFound, "user not found")
		}
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load user")
	}

	var profile models.UserProfile
	err := s.db.Where("user_id = ?", userID).First(&profile).Error
	if err == nil {
		profile.Username = user.Username
		profile.Email = user.Email
		if profile.Level <= 0 {
			profile.Level = 1
		}
		if strings.TrimSpace(profile.Nickname) == "" {
			profile.Nickname = user.Username
		}
		return &profile, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load profile")
	}

	now := time.Now().UTC()
	profile = models.UserProfile{
		UserID:    userID,
		Nickname:  user.Username,
		AvatarURL: "",
		Level:     1,
		CreatedAt: now,
		UpdatedAt: now,
		Username:  user.Username,
		Email:     user.Email,
	}
	if err := s.db.Create(&profile).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to create profile")
	}
	return &profile, nil
}

// UpdateProfile 更新昵称。
func (s *ProfileService) UpdateProfile(userID uint64, req ProfileUpdateRequest) (*models.UserProfile, error) {
	profile, err := s.GetProfile(userID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Nickname)
	if name != "" {
		profile.Nickname = name
	}
	if err := s.db.Save(profile).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to update profile")
	}
	return profile, nil
}

// UpdateAvatar 更新头像 URL。
func (s *ProfileService) UpdateAvatar(userID uint64, url string) (*models.UserProfile, error) {
	profile, err := s.GetProfile(userID)
	if err != nil {
		return nil, err
	}
	profile.AvatarURL = strings.TrimSpace(url)
	if err := s.db.Save(profile).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to update avatar")
	}
	return profile, nil
}
