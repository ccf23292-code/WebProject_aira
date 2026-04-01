package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	"warehouse-web/models"
	"warehouse-web/services"
)

type question struct {
	ID           string `json:"id"`
	CourseID     string `json:"courseId"`
	Category     string `json:"category"`
	SequenceID   int    `json:"sequenceId"`
	QuestionType string `json:"questionType"`
	SourceURL    string `json:"sourceUrl"`
	Content      struct {
		Text      string   `json:"text"`
		ImageUrls []string `json:"imageUrls"`
	} `json:"content"`
	Options []struct {
		OptionID  string   `json:"optionId"`
		Text      string   `json:"text"`
		ImageUrls []string `json:"imageUrls"`
	} `json:"options"`
	Answer struct {
		Text             string   `json:"text"`
		CorrectOptionIds []string `json:"correctOptionIds"`
		CodeSnippet      string   `json:"codeSnippet"`
		CodeLanguage     string   `json:"codeLanguage"`
	} `json:"answer"`
	Difficulty string   `json:"difficulty"`
	Tag        []string `json:"tag"`
}

func main() {
	var inputDir string
	var courseID string
	flag.StringVar(&inputDir, "path", "", "path to paper json directory")
	flag.StringVar(&courseID, "course-id", "", "course id (kcdm)")
	flag.Parse()

	if strings.TrimSpace(inputDir) == "" || strings.TrimSpace(courseID) == "" {
		log.Fatal("missing --path or --course-id")
	}

	absPath, err := filepath.Abs(inputDir)
	if err != nil {
		log.Fatalf("failed to resolve path: %v", err)
	}

	files, err := filepath.Glob(filepath.Join(absPath, "*.json"))
	if err != nil {
		log.Fatalf("failed to list json files: %v", err)
	}
	if len(files) == 0 {
		log.Fatalf("no json files found under %s", absPath)
	}

	db, err := services.InitPostgres()
	if err != nil {
		log.Fatalf("database init failed: %v", err)
	}
	db = db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
	if err := db.AutoMigrate(&models.TestPaper{}, &models.Problem{}); err != nil {
		log.Fatalf("auto migrate failed: %v", err)
	}

	paperMap := make(map[string]*models.TestPaper)
	inserted := 0

	for _, file := range files {
		payload, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("read %s failed: %v", file, err)
		}

		var questions []question
		if err := json.Unmarshal(payload, &questions); err != nil {
			log.Fatalf("parse %s failed: %v", file, err)
		}

		for _, q := range questions {
			category := strings.TrimSpace(q.Category)
			if category == "" {
				category = "unknown"
			}

			paper := paperMap[category]
			if paper == nil {
				paper = &models.TestPaper{CourseID: courseID, Name: category, CreatedAt: time.Now().UTC()}
				if err := db.Where("course_id = ? AND name = ?", courseID, category).Find(paper).Error; err != nil {
					log.Fatalf("load paper failed: %v", err)
				}
				if paper.ID == 0 {
					if err := db.Create(paper).Error; err != nil {
						log.Fatalf("create paper failed: %v", err)
					}
				}
				paperMap[category] = paper
			}

			options := make([]models.Option, 0, len(q.Options))
			for _, opt := range q.Options {
				options = append(options, models.Option{Option: opt.OptionID, Text: opt.Text})
			}
			optionsJSON, _ := json.Marshal(options)
			tagsJSON, _ := json.Marshal(q.Tag)

			answer := strings.TrimSpace(q.Answer.Text)
			if len(q.Answer.CorrectOptionIds) > 0 {
				answer = strings.Join(q.Answer.CorrectOptionIds, ",")
			}

			problem := models.Problem{
				TestpaperID:  paper.ID,
				SourceID:     strings.TrimSpace(q.ID),
				Order:        q.SequenceID,
				SequenceID:   q.SequenceID,
				QuestionType: strings.TrimSpace(q.QuestionType),
				Category:     category,
				SourceURL:    strings.TrimSpace(q.SourceURL),
				Test:         strings.TrimSpace(q.Content.Text),
				Answer:       answer,
				Score:        0,
				Explanation:  "",
				Difficulty:   strings.TrimSpace(q.Difficulty),
				OptionsJSON:  optionsJSON,
				TagsJSON:     tagsJSON,
			}

			if err := db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "testpaper_id"}, {Name: "source_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"order", "sequence_id", "question_type", "category", "source_url", "test", "answer", "difficulty", "options_json", "tags_json", "score", "explanation"}),
			}).Create(&problem).Error; err != nil {
				log.Fatalf("insert problem failed: %v", err)
			}
			inserted++
		}
	}

	fmt.Printf("imported %d problems\n", inserted)
}
