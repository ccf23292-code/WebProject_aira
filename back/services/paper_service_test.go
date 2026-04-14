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

func newPaperServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:paper-service-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.TestPaper{}, &models.Problem{}, &models.RecallPaper{}, &models.RecallQuestion{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestConvertRecallPaperCreatesPaperAndArchivesSource(t *testing.T) {
	db := newPaperServiceTestDB(t)
	now := time.Now().UTC()

	recallPaper := models.RecallPaper{
		CourseID:  "CS101",
		Title:     "2025 Final Recall",
		CreatedBy: 7,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.Create(&recallPaper).Error; err != nil {
		t.Fatalf("create recall paper: %v", err)
	}

	questions := []models.RecallQuestion{
		{
			PaperID:      recallPaper.ID,
			QuestionType: "singleChoice",
			Sequence:     1,
			Content:      "old version",
			Answer:       "A",
			SupportCount: 1,
			CreatedAt:    now,
			UpdatedAt:    now.Add(-time.Minute),
		},
		{
			PaperID:      recallPaper.ID,
			QuestionType: "singleChoice",
			Sequence:     1,
			Content:      "best version",
			Answer:       "B",
			OptionsJSON:  []byte(`[{"option":"A","text":"x"},{"option":"B","text":"y"}]`),
			SupportCount: 5,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			PaperID:      recallPaper.ID,
			QuestionType: "shortAnswer",
			Sequence:     2,
			Content:      "Explain CAP theorem",
			Answer:       "Consistency, Availability, Partition tolerance",
			SupportCount: 2,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}
	if err := db.Create(&questions).Error; err != nil {
		t.Fatalf("create recall questions: %v", err)
	}

	service := NewPaperService(db)
	result, err := service.ConvertRecallPaper(recallPaper.ID, ConvertRecallPaperRequest{})
	if err != nil {
		t.Fatalf("convert recall paper: %v", err)
	}
	if result.QuestionCount != 2 {
		t.Fatalf("expected 2 converted questions, got %d", result.QuestionCount)
	}

	var paper models.TestPaper
	if err := db.First(&paper, result.PaperID).Error; err != nil {
		t.Fatalf("load created paper: %v", err)
	}
	if paper.Name != recallPaper.Title {
		t.Fatalf("expected paper name %q, got %q", recallPaper.Title, paper.Name)
	}

	var problems []models.Problem
	if err := db.Where("testpaper_id = ?", paper.ID).Order("\"order\" asc").Find(&problems).Error; err != nil {
		t.Fatalf("load created problems: %v", err)
	}
	if len(problems) != 2 {
		t.Fatalf("expected 2 problems, got %d", len(problems))
	}

	foundBestVersion := false
	for _, problem := range problems {
		if problem.Test == "best version" {
			foundBestVersion = true
			if string(problem.OptionsJSON) == "" || string(problem.OptionsJSON) == "[]" {
				t.Fatalf("expected choice options to be preserved")
			}
		}
	}
	if !foundBestVersion {
		t.Fatalf("expected highest-supported recall question to be converted")
	}

	var archived models.RecallPaper
	if err := db.First(&archived, recallPaper.ID).Error; err != nil {
		t.Fatalf("reload recall paper: %v", err)
	}
	if archived.ConvertedPaperID == nil || *archived.ConvertedPaperID != paper.ID {
		t.Fatalf("expected converted paper id %d, got %#v", paper.ID, archived.ConvertedPaperID)
	}
	if archived.ConvertedAt == nil {
		t.Fatalf("expected converted_at to be populated")
	}
}
