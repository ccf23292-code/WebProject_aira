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

func newHomepageTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:homepage-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.UserProfile{},
		&models.HomepageMessage{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestAddHomepageMessageRejectsBlankContent(t *testing.T) {
	db := newHomepageTestDB(t)
	service := NewHomepageService(db)

	if _, err := service.AddMessage(1, "   "); err == nil {
		t.Fatal("expected blank content to be rejected")
	}
}

func TestAddHomepageMessagePersistsMessage(t *testing.T) {
	db := newHomepageTestDB(t)
	service := NewHomepageService(db)

	if err := db.Create(&models.User{ID: 9, Username: "student", Email: "student@example.com", PasswordHash: "hash", Role: models.RoleStudent}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&models.UserProfile{UserID: 9, Nickname: "普通同学", Level: 1}).Error; err != nil {
		t.Fatalf("create profile: %v", err)
	}

	item, err := service.AddMessage(9, "希望增加章节归类和留言互动。")
	if err != nil {
		t.Fatalf("add message: %v", err)
	}

	if item.Content != "希望增加章节归类和留言互动。" {
		t.Fatalf("unexpected content: %#v", item)
	}
	if item.UserName != "普通同学" {
		t.Fatalf("expected hydrated nickname, got %#v", item)
	}

	var stored models.HomepageMessage
	if err := db.First(&stored, item.ID).Error; err != nil {
		t.Fatalf("load stored message: %v", err)
	}
	if stored.Content != "希望增加章节归类和留言互动。" {
		t.Fatalf("unexpected stored content: %#v", stored)
	}
}

func TestListHomepageMessagesHydratesProfileFields(t *testing.T) {
	db := newHomepageTestDB(t)
	service := NewHomepageService(db)

	if err := db.Create(&models.User{ID: 7, Username: "alice", Email: "alice@example.com", PasswordHash: "hash", Role: models.RoleStudent}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&models.UserProfile{
		UserID:    7,
		Nickname:  "Alice 同学",
		AvatarURL: "/static/avatars/alice.png",
		Level:     2,
	}).Error; err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if err := db.Create(&models.HomepageMessage{
		UserID:  7,
		Content: "建议增加管理员审核备注的可视化输入。",
	}).Error; err != nil {
		t.Fatalf("create message: %v", err)
	}

	items, err := service.ListMessages()
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].UserName != "Alice 同学" {
		t.Fatalf("expected nickname to be hydrated, got %#v", items[0])
	}
	if items[0].AvatarURL != "/static/avatars/alice.png" {
		t.Fatalf("expected avatar to be hydrated, got %#v", items[0])
	}
}
