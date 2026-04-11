package main

import (
	"flag"
	"log"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"warehouse-web/models"
	"warehouse-web/services"
)

func main() {
	username := flag.String("username", "", "username")
	password := flag.String("password", "", "password")
	email := flag.String("email", "", "email")
	nickname := flag.String("nickname", "", "nickname")
	roleValue := flag.String("role", string(models.RoleStudent), "role: student or admin")
	flag.Parse()

	if strings.TrimSpace(*username) == "" {
		log.Fatal("username is required")
	}
	if strings.TrimSpace(*password) == "" {
		log.Fatal("password is required")
	}
	if strings.TrimSpace(*email) == "" {
		log.Fatal("email is required")
	}

	role, err := models.ParseRole(*roleValue)
	if err != nil {
		log.Fatal(err)
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
	usernameValue := strings.TrimSpace(*username)
	emailValue := strings.TrimSpace(*email)
	nicknameValue := strings.TrimSpace(*nickname)
	if nicknameValue == "" {
		nicknameValue = usernameValue
	}

	var user models.User
	err = db.Where("LOWER(username) = ?", strings.ToLower(usernameValue)).First(&user).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Fatalf("load user: %v", err)
	}
	if err == gorm.ErrRecordNotFound {
		user = models.User{
			Username:     usernameValue,
			Email:        emailValue,
			PasswordHash: string(hash),
			Role:         role,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := db.Create(&user).Error; err != nil {
			log.Fatalf("create user: %v", err)
		}
	} else {
		user.Email = emailValue
		user.PasswordHash = string(hash)
		user.Role = role
		user.UpdatedAt = now
		if err := db.Save(&user).Error; err != nil {
			log.Fatalf("update user: %v", err)
		}
	}

	var profile models.UserProfile
	err = db.Where("user_id = ?", user.ID).First(&profile).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Fatalf("load profile: %v", err)
	}
	if err == gorm.ErrRecordNotFound {
		profile = models.UserProfile{
			UserID:    user.ID,
			Nickname:  nicknameValue,
			AvatarURL: "",
			Level:     1,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := db.Create(&profile).Error; err != nil {
			log.Fatalf("create profile: %v", err)
		}
	} else {
		profile.Nickname = nicknameValue
		profile.UpdatedAt = now
		if err := db.Save(&profile).Error; err != nil {
			log.Fatalf("update profile: %v", err)
		}
	}
}
