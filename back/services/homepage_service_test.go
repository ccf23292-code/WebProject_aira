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

func TestUpdateHomepageMessageAllowsAuthor(t *testing.T) {
	db := newHomepageTestDB(t)
	service := NewHomepageService(db)

	if err := db.Create(&models.User{ID: 3, Username: "owner", Email: "owner@example.com", PasswordHash: "hash", Role: models.RoleStudent}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&models.HomepageMessage{UserID: 3, Content: "旧内容"}).Error; err != nil {
		t.Fatalf("create message: %v", err)
	}

	item, err := service.UpdateMessage(3, 1, "新内容")
	if err != nil {
		t.Fatalf("update message: %v", err)
	}
	if item.Content != "新内容" {
		t.Fatalf("expected updated content, got %#v", item)
	}
}

func TestUpdateHomepageMessageRejectsNonAuthor(t *testing.T) {
	db := newHomepageTestDB(t)
	service := NewHomepageService(db)

	if err := db.Create(&models.User{ID: 3, Username: "owner", Email: "owner@example.com", PasswordHash: "hash", Role: models.RoleStudent}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&models.User{ID: 4, Username: "other", Email: "other@example.com", PasswordHash: "hash", Role: models.RoleStudent}).Error; err != nil {
		t.Fatalf("create other user: %v", err)
	}
	if err := db.Create(&models.HomepageMessage{ID: 7, UserID: 3, Content: "旧内容"}).Error; err != nil {
		t.Fatalf("create message: %v", err)
	}

	if _, err := service.UpdateMessage(4, 7, "新内容"); err == nil {
		t.Fatal("expected non-author update to fail")
	}
}

func TestDeleteHomepageMessageAllowsAuthor(t *testing.T) {
	db := newHomepageTestDB(t)
	service := NewHomepageService(db)

	if err := db.Create(&models.User{ID: 3, Username: "owner", Email: "owner@example.com", PasswordHash: "hash", Role: models.RoleStudent}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&models.HomepageMessage{ID: 9, UserID: 3, Content: "待删除"}).Error; err != nil {
		t.Fatalf("create message: %v", err)
	}

	if err := service.DeleteMessage(3, 9); err != nil {
		t.Fatalf("delete message: %v", err)
	}

	var count int64
	if err := db.Model(&models.HomepageMessage{}).Where("id = ?", 9).Count(&count).Error; err != nil {
		t.Fatalf("count messages: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected message to be deleted, count=%d", count)
	}
}
