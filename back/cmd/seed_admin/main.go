package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"warehouse-web/models"
	"warehouse-web/services"
)

func main() {
	username := flag.String("username", "admin", "admin username")
	password := flag.String("password", "", "admin password")
	email := flag.String("email", "admin@example.com", "admin email")
	nickname := flag.String("nickname", "admin", "admin nickname")
	flag.Parse()

	if strings.TrimSpace(*password) == "" {
		log.Fatal("password is required")
	}

	db, err := services.InitPostgres()
	if err != nil {
		log.Fatalf("database init failed: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.UserProfile{}); err != nil {
		log.Fatalf("database migrate failed: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	now := time.Now().UTC()
	var user models.User
	err = db.Where("LOWER(username) = ?", strings.ToLower(strings.TrimSpace(*username))).First(&user).Error
	if err != nil {
		user = models.User{
			Username:     strings.TrimSpace(*username),
			Email:        strings.TrimSpace(*email),
			PasswordHash: string(hash),
			Role:         models.RoleAdmin,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := db.Create(&user).Error; err != nil {
			log.Fatalf("create admin user: %v", err)
		}
	} else {
		user.Email = strings.TrimSpace(*email)
		user.PasswordHash = string(hash)
		user.Role = models.RoleAdmin
		user.UpdatedAt = now
		if err := db.Save(&user).Error; err != nil {
			log.Fatalf("update admin user: %v", err)
		}
	}

	var profile models.UserProfile
	if err := db.Where("user_id = ?", user.ID).First(&profile).Error; err != nil {
		profile = models.UserProfile{
			UserID:    user.ID,
			Nickname:  strings.TrimSpace(*nickname),
			AvatarURL: "",
			Level:     1,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := db.Create(&profile).Error; err != nil {
			log.Fatalf("create admin profile: %v", err)
		}
	} else {
		profile.Nickname = strings.TrimSpace(*nickname)
		profile.UpdatedAt = now
		if err := db.Save(&profile).Error; err != nil {
			log.Fatalf("update admin profile: %v", err)
		}
	}

	fmt.Printf("seeded admin user %s\n", user.Username)
}
