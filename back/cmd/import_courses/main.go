package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gorm.io/gorm/clause"

	"warehouse-web/models"
	"warehouse-web/services"
)

type courseFile struct {
	Courses []courseItem `json:"courses"`
}

type courseItem struct {
	Kcdm   string `json:"kcdm"`
	Xskcdm string `json:"xskcdm"`
	Kcmc   string `json:"kcmc"`
	Kkxy   string `json:"kkxy"`
	Kclb   string `json:"kclb"`
	Xf     string `json:"xf"`
}

func main() {
	var inputDir string
	var onlyName string
	var onlyCode string
	flag.StringVar(&inputDir, "path", "", "path to course json directory")
	flag.StringVar(&onlyName, "only-name", "", "only import courses with this name")
	flag.StringVar(&onlyCode, "only-code", "", "only import courses with this code (xskcdm or kcdm)")
	flag.Parse()

	if strings.TrimSpace(inputDir) == "" {
		log.Fatal("missing --path to course json directory")
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
	if err := db.AutoMigrate(&models.Course{}); err != nil {
		log.Fatalf("auto migrate failed: %v", err)
	}

	inserted := 0
	onlyName = strings.TrimSpace(onlyName)
	onlyCode = strings.TrimSpace(onlyCode)

	for _, file := range files {
		payload, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("read %s failed: %v", file, err)
		}

		var data courseFile
		if err := json.Unmarshal(payload, &data); err != nil {
			log.Fatalf("parse %s failed: %v", file, err)
		}

		for _, item := range data.Courses {
			id := strings.TrimSpace(item.Kcdm)
			code := strings.TrimSpace(item.Kcdm)
			name := strings.TrimSpace(item.Kcmc)
			if id == "" || name == "" {
				continue
			}
			if onlyName != "" && name != onlyName {
				continue
			}
			if onlyCode != "" && code != onlyCode && id != onlyCode {
				continue
			}

			credits, _ := strconv.ParseFloat(strings.TrimSpace(item.Xf), 64)
			course := models.Course{
				ID:       id,
				Code:     code,
				Name:     name,
				College:  strings.TrimSpace(item.Kkxy),
				Category: strings.TrimSpace(item.Kclb),
				Credits:  credits,
			}

			if err := db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{"code", "name", "college", "category", "credits"}),
			}).Create(&course).Error; err != nil {
				log.Fatalf("insert course %s failed: %v", id, err)
			}
			inserted++
		}
	}

	fmt.Printf("imported %d courses\n", inserted)
}
