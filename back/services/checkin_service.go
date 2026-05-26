package services

import (
	"errors"
	"net/http"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"warehouse-web/models"
)

// CheckinService 提供每日签到相关的业务逻辑。
type CheckinService struct {
	db *gorm.DB
}

// NewCheckinService 创建 CheckinService。
func NewCheckinService(db *gorm.DB) *CheckinService {
	return &CheckinService{db: db}
}

// dateLayout 是签到日期统一的字符串格式（YYYY-MM-DD）。
const dateLayout = "2006-01-02"

// todayDate 取服务器本地时区下的日期字符串。
// 注意：以服务器为准，不信任前端传来的日期，避免被改本机时间刷连签。
func todayDate() string {
	return time.Now().Format(dateLayout)
}

// yesterdayDate 返回服务器本地时区下昨日的日期字符串。
func yesterdayDate() string {
	return time.Now().AddDate(0, 0, -1).Format(dateLayout)
}

// GetTodayStatus 查询用户当前的签到状态。
// 用户尚未签到过时返回零值状态（CheckedToday=false, ContinuousDays=0 …）。
func (s *CheckinService) GetTodayStatus(userID models.PrimaryKey) (models.CheckinStatus, error) {
	var record models.UserCheckin
	err := s.db.Where("user_id = ?", userID).First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.CheckinStatus{
				CheckedToday:    false,
				LastCheckinDate: "",
				ContinuousDays:  0,
				MaxContinuous:   0,
				TotalDays:       0,
			}, nil
		}
		return models.CheckinStatus{}, newServiceError("internal_error", http.StatusInternalServerError, "failed to load checkin status")
	}

	today := todayDate()
	return models.CheckinStatus{
		CheckedToday:    record.LastCheckinDate == today,
		LastCheckinDate: record.LastCheckinDate,
		ContinuousDays:  record.ContinuousDays,
		MaxContinuous:   record.MaxContinuous,
		TotalDays:       record.TotalDays,
	}, nil
}

// CheckIn 处理一次签到请求。
//   - 当日已签到：返回 *ServiceError{Code: "already_checked", Status: 409}
//   - 上次签到为昨日：连续天数 +1
//   - 上次签到更早或首次签到：连续天数重置为 1
func (s *CheckinService) CheckIn(userID models.PrimaryKey) (models.CheckinStatus, error) {
	today := todayDate()
	yesterday := yesterdayDate()
	now := time.Now().UTC()

	var status models.CheckinStatus

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var record models.UserCheckin
		// 行级锁：避免同一用户并发签到导致连签 +2
		loadErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", userID).
			First(&record).Error

		switch {
		case loadErr == nil:
			// 已有记录
			if record.LastCheckinDate == today {
				return newServiceError("already_checked", http.StatusConflict, "今日已签到")
			}
			if record.LastCheckinDate == yesterday {
				record.ContinuousDays++
			} else {
				record.ContinuousDays = 1
			}
			record.LastCheckinDate = today
			record.TotalDays++
			if record.ContinuousDays > record.MaxContinuous {
				record.MaxContinuous = record.ContinuousDays
			}
			record.UpdatedAt = now
			if saveErr := tx.Save(&record).Error; saveErr != nil {
				return newServiceError("internal_error", http.StatusInternalServerError, "failed to update checkin")
			}

		case errors.Is(loadErr, gorm.ErrRecordNotFound):
			// 首次签到，插入一行
			record = models.UserCheckin{
				UserID:          userID,
				LastCheckinDate: today,
				ContinuousDays:  1,
				MaxContinuous:   1,
				TotalDays:       1,
				UpdatedAt:       now,
			}
			if createErr := tx.Create(&record).Error; createErr != nil {
				return newServiceError("internal_error", http.StatusInternalServerError, "failed to create checkin")
			}

		default:
			return newServiceError("internal_error", http.StatusInternalServerError, "failed to load checkin")
		}

		status = models.CheckinStatus{
			CheckedToday:    true,
			LastCheckinDate: record.LastCheckinDate,
			ContinuousDays:  record.ContinuousDays,
			MaxContinuous:   record.MaxContinuous,
			TotalDays:       record.TotalDays,
		}
		return nil
	})

	if err != nil {
		return models.CheckinStatus{}, err
	}
	return status, nil
}
