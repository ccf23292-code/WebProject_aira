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

func newCourseTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:course-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.UserProfile{},
		&models.Course{},
		&models.Teacher{},
		&models.CourseComment{},
		&models.TeacherComment{},
		&models.GradingStandard{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestCommentsAndGradingReturnDisplayFields(t *testing.T) {
	db := newCourseTestDB(t)
	service := NewCourseService(db)

	if err := db.Create(&models.Course{ID: "CS1018F", Code: "CS1018F", Name: "数据结构基础"}).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	if err := db.Create(&models.User{ID: 7, Username: "alice", Email: "alice@zju.edu.cn", PasswordHash: "hash", Role: models.RoleStudent}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&models.UserProfile{UserID: 7, Nickname: "Alice 同学", Level: 1}).Error; err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if err := db.Create(&models.Teacher{ID: "t-li", CourseID: "CS1018F", Name: "李老师"}).Error; err != nil {
		t.Fatalf("create teacher: %v", err)
	}

	if _, err := service.AddCourseComment("CS1018F", 7, "课程评论"); err != nil {
		t.Fatalf("add course comment: %v", err)
	}
	if _, err := service.AddTeacherComment("CS1018F", "t-li", 7, "教师评论"); err != nil {
		t.Fatalf("add teacher comment: %v", err)
	}
	if _, err := service.AddGradingStandard("CS1018F", "t-li", AddGradingStandardRequest{Standard: "平时 40%，期末 60%"}); err != nil {
		t.Fatalf("add grading standard: %v", err)
	}

	courseComments, err := service.GetCourseComments("CS1018F")
	if err != nil {
		t.Fatalf("get course comments: %v", err)
	}
	if len(courseComments) != 1 {
		t.Fatalf("expected 1 course comment, got %d", len(courseComments))
	}
	if courseComments[0].UserName != "Alice 同学" {
		t.Fatalf("expected course comment user_name to be hydrated, got %#v", courseComments[0])
	}
	if courseComments[0].CreatedAt.IsZero() {
		t.Fatalf("expected course comment created_at to be set, got %#v", courseComments[0])
	}

	teacherComments, err := service.GetTeacherComments("CS1018F", "t-li")
	if err != nil {
		t.Fatalf("get teacher comments: %v", err)
	}
	if len(teacherComments) != 1 {
		t.Fatalf("expected 1 teacher comment, got %d", len(teacherComments))
	}
	if teacherComments[0].TeacherName != "李老师" || teacherComments[0].UserName != "Alice 同学" {
		t.Fatalf("expected teacher and user display fields, got %#v", teacherComments[0])
	}

	standards, err := service.GetGradingStandards("CS1018F", "t-li")
	if err != nil {
		t.Fatalf("get grading standards: %v", err)
	}
	if len(standards) != 1 {
		t.Fatalf("expected 1 grading standard, got %d", len(standards))
	}
	if standards[0].TeacherName != "李老师" {
		t.Fatalf("expected grading standard teacher_name to be hydrated, got %#v", standards[0])
	}
	if standards[0].CreatedAt.IsZero() {
		t.Fatalf("expected grading standard created_at to be set, got %#v", standards[0])
	}
}
