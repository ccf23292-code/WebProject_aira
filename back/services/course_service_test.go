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
		&models.TeacherSubmission{},
		&models.CourseComment{},
		&models.TeacherComment{},
		&models.GradingStandard{},
		&models.GradingStandardSubmission{},
		&models.CourseDescriptionSubmission{},
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
	if err := db.Create(&models.GradingStandard{
		CourseID:  "CS1018F",
		TeacherID: "t-li",
		Standard:  "平时 40%，期末 60%",
	}).Error; err != nil {
		t.Fatalf("seed grading standard: %v", err)
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

func TestAddTeacherCreatesPendingSubmission(t *testing.T) {
	db := newCourseTestDB(t)
	service := NewCourseService(db)

	if err := db.Create(&models.Course{ID: "CS1018F", Code: "CS1018F", Name: "数据结构基础"}).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}

	item, err := service.AddTeacher("CS1018F", 8, AddTeacherRequest{Name: "张老师", Title: "2025 春"})
	if err != nil {
		t.Fatalf("add teacher submission: %v", err)
	}

	if item.Status != models.CourseDescriptionSubmissionPending {
		t.Fatalf("expected pending teacher submission, got %#v", item)
	}

	teachers, err := service.ListTeachers("CS1018F")
	if err != nil {
		t.Fatalf("list teachers: %v", err)
	}
	if len(teachers) != 0 {
		t.Fatalf("expected no published teachers before review, got %#v", teachers)
	}
}

func TestApproveTeacherSubmissionPublishesTeacher(t *testing.T) {
	db := newCourseTestDB(t)
	service := NewCourseService(db)

	if err := db.Create(&models.Course{ID: "CS1018F", Code: "CS1018F", Name: "数据结构基础"}).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}

	item, err := service.AddTeacher("CS1018F", 8, AddTeacherRequest{Name: "张老师", Title: "2025 春"})
	if err != nil {
		t.Fatalf("add teacher submission: %v", err)
	}

	reviewed, err := service.ReviewTeacherSubmission(uint64(item.ID), 1, ReviewCourseDescriptionRequest{Action: "approve"})
	if err != nil {
		t.Fatalf("review teacher submission: %v", err)
	}
	if reviewed.Status != models.CourseDescriptionSubmissionApproved {
		t.Fatalf("expected approved teacher submission, got %#v", reviewed)
	}

	teachers, err := service.ListTeachers("CS1018F")
	if err != nil {
		t.Fatalf("list teachers: %v", err)
	}
	if len(teachers) != 1 || teachers[0].Name != "张老师" {
		t.Fatalf("expected published teacher, got %#v", teachers)
	}
}

func TestAddGradingStandardCreatesPendingSubmission(t *testing.T) {
	db := newCourseTestDB(t)
	service := NewCourseService(db)

	if err := db.Create(&models.Course{ID: "CS1018F", Code: "CS1018F", Name: "数据结构基础"}).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	if err := db.Create(&models.Teacher{ID: "t-li", CourseID: "CS1018F", Name: "李老师"}).Error; err != nil {
		t.Fatalf("create teacher: %v", err)
	}

	item, err := service.AddGradingStandard("CS1018F", "t-li", 8, AddGradingStandardRequest{Standard: "平时 40%，期末 60%"})
	if err != nil {
		t.Fatalf("add grading submission: %v", err)
	}
	if item.Status != models.CourseDescriptionSubmissionPending {
		t.Fatalf("expected pending grading submission, got %#v", item)
	}

	standards, err := service.GetGradingStandards("CS1018F", "t-li")
	if err != nil {
		t.Fatalf("get grading standards: %v", err)
	}
	if len(standards) != 0 {
		t.Fatalf("expected no published grading standards before review, got %#v", standards)
	}
}

func TestApproveGradingSubmissionPublishesStandard(t *testing.T) {
	db := newCourseTestDB(t)
	service := NewCourseService(db)

	if err := db.Create(&models.Course{ID: "CS1018F", Code: "CS1018F", Name: "数据结构基础"}).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	if err := db.Create(&models.Teacher{ID: "t-li", CourseID: "CS1018F", Name: "李老师"}).Error; err != nil {
		t.Fatalf("create teacher: %v", err)
	}

	item, err := service.AddGradingStandard("CS1018F", "t-li", 8, AddGradingStandardRequest{Standard: "平时 40%，期末 60%"})
	if err != nil {
		t.Fatalf("add grading submission: %v", err)
	}

	reviewed, err := service.ReviewGradingStandardSubmission(uint64(item.ID), 1, ReviewCourseDescriptionRequest{Action: "approve"})
	if err != nil {
		t.Fatalf("review grading submission: %v", err)
	}
	if reviewed.Status != models.CourseDescriptionSubmissionApproved {
		t.Fatalf("expected approved grading submission, got %#v", reviewed)
	}

	standards, err := service.GetGradingStandards("CS1018F", "t-li")
	if err != nil {
		t.Fatalf("get grading standards: %v", err)
	}
	if len(standards) != 1 || standards[0].Standard != "平时 40%，期末 60%" {
		t.Fatalf("expected published grading standard, got %#v", standards)
	}
}

func TestSubmitCourseDescriptionCreatesPendingSubmission(t *testing.T) {
	db := newCourseTestDB(t)
	service := NewCourseService(db)

	if err := db.Create(&models.Course{ID: "CS1018F", Code: "CS1018F", Name: "数据结构基础"}).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}

	item, err := service.SubmitCourseDescription("CS1018F", 8, "这门课重点在树和图。")
	if err != nil {
		t.Fatalf("submit course description: %v", err)
	}

	if item.Status != models.CourseDescriptionSubmissionPending {
		t.Fatalf("expected pending status, got %#v", item)
	}
	if item.Content != "这门课重点在树和图。" {
		t.Fatalf("unexpected submission content: %#v", item)
	}

	items, err := service.ListMyCourseDescriptionSubmissions("CS1018F", 8)
	if err != nil {
		t.Fatalf("list my submissions: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one submission, got %#v", items)
	}
}

func TestApproveCourseDescriptionSubmissionPublishesDescription(t *testing.T) {
	db := newCourseTestDB(t)
	service := NewCourseService(db)

	if err := db.Create(&models.Course{
		ID:          "CS1018F",
		Code:        "CS1018F",
		Name:        "数据结构基础",
		Description: "旧简介",
	}).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}

	item, err := service.SubmitCourseDescription("CS1018F", 8, "新的课程简介")
	if err != nil {
		t.Fatalf("submit course description: %v", err)
	}

	reviewed, err := service.ReviewCourseDescriptionSubmission(uint64(item.ID), 1, ReviewCourseDescriptionRequest{
		Action:     "approve",
		ReviewNote: "内容准确",
	})
	if err != nil {
		t.Fatalf("review submission: %v", err)
	}

	if reviewed.Status != models.CourseDescriptionSubmissionApproved {
		t.Fatalf("expected approved status, got %#v", reviewed)
	}

	course, err := service.GetCourse("CS1018F")
	if err != nil {
		t.Fatalf("get course: %v", err)
	}
	if course.Description != "新的课程简介" {
		t.Fatalf("expected course description to be published, got %#v", course)
	}
}
