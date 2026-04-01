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

func newAnswerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:answers-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	ddl := []string{
		`CREATE TABLE test_papers (
			id INTEGER PRIMARY KEY,
			course_id TEXT,
			name TEXT,
			created_at DATETIME
		);`,
		`CREATE TABLE problems (
			id INTEGER PRIMARY KEY,
			testpaper_id INTEGER,
			source_id TEXT,
			"order" INTEGER,
			sequence_id INTEGER,
			question_type TEXT,
			category TEXT,
			source_url TEXT,
			test TEXT,
			answer TEXT,
			score REAL,
			explanation TEXT,
			difficulty TEXT,
			options_json TEXT,
			tags_json TEXT
		);`,
		`CREATE TABLE answer_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			course_id TEXT,
			paper_id INTEGER,
			problem_id INTEGER,
			selected_option TEXT,
			is_correct BOOLEAN,
			mode TEXT,
			answered_at DATETIME
		);`,
		`CREATE TABLE wrong_questions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			problem_id INTEGER,
			course_id TEXT,
			paper_id INTEGER,
			note TEXT,
			status TEXT,
			wrong_count INTEGER,
			last_wrong_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		);`,
	}
	for _, stmt := range ddl {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("exec ddl: %v", err)
		}
	}
	return db
}

func seedAnswerFixture(t *testing.T, db *gorm.DB) {
	t.Helper()

	paper := models.TestPaper{
		ID:        9,
		CourseID:  "CS1018F",
		Name:      "FDS",
		CreatedAt: time.Now().UTC(),
	}
	if err := db.Create(&paper).Error; err != nil {
		t.Fatalf("create paper: %v", err)
	}

	problems := []models.Problem{
		{ID: 101, TestpaperID: paper.ID, SourceID: "a", Order: 1, Test: "p1", Answer: "A"},
		{ID: 102, TestpaperID: paper.ID, SourceID: "b", Order: 2, Test: "p2", Answer: "B"},
	}
	for _, problem := range problems {
		if err := db.Create(&problem).Error; err != nil {
			t.Fatalf("create problem: %v", err)
		}
	}
}

func TestRecordAnswersBatchCreatesRecordsAndWrongQuestions(t *testing.T) {
	db := newAnswerTestDB(t)
	seedAnswerFixture(t, db)
	service := NewAnswerService(db, NewPaperService(db))

	err := service.RecordAnswersBatch(7, []AnswerRecordRequest{
		{
			PaperID:        9,
			ProblemID:      101,
			SelectedOption: "A",
			IsCorrect:      true,
			Mode:           "exam",
		},
		{
			PaperID:        9,
			ProblemID:      102,
			SelectedOption: "A",
			IsCorrect:      false,
			Mode:           "exam",
		},
	})
	if err != nil {
		t.Fatalf("record batch: %v", err)
	}

	var recordCount int64
	if err := db.Model(&models.AnswerRecord{}).Count(&recordCount).Error; err != nil {
		t.Fatalf("count answer records: %v", err)
	}
	if recordCount != 2 {
		t.Fatalf("expected 2 answer records, got %d", recordCount)
	}

	var wrongCount int64
	if err := db.Model(&models.WrongQuestion{}).Count(&wrongCount).Error; err != nil {
		t.Fatalf("count wrong questions: %v", err)
	}
	if wrongCount != 1 {
		t.Fatalf("expected 1 wrong question, got %d", wrongCount)
	}
}
