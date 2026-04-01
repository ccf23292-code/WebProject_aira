package services

import (
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"warehouse-web/models"
)

func newProfileTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:profile-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.UserProfile{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestGetProfileReturnsLevelAndIdentityFields(t *testing.T) {
	db := newProfileTestDB(t)
	if err := db.Create(&models.User{
		ID:           9,
		Username:     "dora",
		Email:        "dora@zju.edu.cn",
		PasswordHash: "hash",
		Role:         models.RoleStudent,
	}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	service := NewProfileService(db)
	profile, err := service.GetProfile(9)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}

	if profile.Username != "dora" || profile.Email != "dora@zju.edu.cn" {
		t.Fatalf("expected identity fields on profile view: %#v", profile)
	}
	if profile.Level != 1 {
		t.Fatalf("expected default level 1, got %#v", profile)
	}
	if profile.Nickname != "dora" {
		t.Fatalf("expected default nickname copied from username, got %#v", profile)
	}
}
