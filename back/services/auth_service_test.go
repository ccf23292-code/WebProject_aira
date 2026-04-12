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

type fakeMailer struct {
	emails []string
	codes  []string
}

func (m *fakeMailer) SendVerificationCode(email, code string, expiresIn time.Duration) error {
	m.emails = append(m.emails, email)
	m.codes = append(m.codes, code)
	return nil
}

func newAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:auth-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.UserProfile{},
		&models.AuthSession{},
		&models.EmailVerification{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestRegisterPersistsUserAndProfile(t *testing.T) {
	db := newAuthTestDB(t)
	service := NewAuthService(db)

	sendResp, err := service.SendVerificationCode("alice@zju.edu.cn", true)
	if err != nil {
		t.Fatalf("send verification code: %v", err)
	}

	registerResp, err := service.Register(RegisterRequest{
		Username:         "alice",
		Email:            "alice@zju.edu.cn",
		Password:         "Alice123",
		ConfirmPassword:  "Alice123",
		VerificationCode: sendResp.Code,
		AgreeToPolicy:    true,
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if registerResp.AccessToken == "" || registerResp.RefreshToken == "" {
		t.Fatalf("expected tokens in register response: %#v", registerResp)
	}

	restarted := NewAuthService(db)
	loginResp, err := restarted.Login(LoginRequest{
		Username: "alice",
		Password: "Alice123",
	})
	if err != nil {
		t.Fatalf("login after restart: %v", err)
	}
	if loginResp.DisplayName != "alice" {
		t.Fatalf("unexpected display name: %#v", loginResp)
	}

	profileService := NewProfileService(db)
	profile, err := profileService.GetProfile(1)
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	if profile.Username != "alice" || profile.Email != "alice@zju.edu.cn" || profile.Level != 1 {
		t.Fatalf("unexpected profile after register: %#v", profile)
	}
}

func TestLogoutRemovesRefreshTokenSession(t *testing.T) {
	db := newAuthTestDB(t)
	service := NewAuthService(db)

	sendResp, err := service.SendVerificationCode("bob@zju.edu.cn", true)
	if err != nil {
		t.Fatalf("send verification code: %v", err)
	}
	registerResp, err := service.Register(RegisterRequest{
		Username:         "bob",
		Email:            "bob@zju.edu.cn",
		Password:         "Bob12345",
		ConfirmPassword:  "Bob12345",
		VerificationCode: sendResp.Code,
		AgreeToPolicy:    true,
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if _, err := service.ValidateAccessToken(registerResp.AccessToken); err != nil {
		t.Fatalf("validate before logout: %v", err)
	}

	if _, err := service.Logout(LogoutRequest{RefreshToken: registerResp.RefreshToken}); err != nil {
		t.Fatalf("logout: %v", err)
	}

	if _, err := service.ValidateAccessToken(registerResp.AccessToken); err == nil {
		t.Fatalf("expected access token to be invalid after logout")
	}
}

func TestVerificationCodePersistsAcrossServiceInstances(t *testing.T) {
	db := newAuthTestDB(t)
	service := NewAuthService(db)

	sendResp, err := service.SendVerificationCode("carol@zju.edu.cn", true)
	if err != nil {
		t.Fatalf("send verification code: %v", err)
	}

	restarted := NewAuthService(db)
	if _, err := restarted.Register(RegisterRequest{
		Username:         "carol",
		Email:            "carol@zju.edu.cn",
		Password:         "Carol123",
		ConfirmPassword:  "Carol123",
		VerificationCode: sendResp.Code,
		AgreeToPolicy:    true,
	}); err != nil {
		t.Fatalf("register after restart: %v", err)
	}
}

func TestSendVerificationCodeRequiresZJUDomain(t *testing.T) {
	db := newAuthTestDB(t)
	service := NewAuthService(db)

	if _, err := service.SendVerificationCode("alice@example.com", true); err == nil {
		t.Fatalf("expected non-zju email to be rejected")
	}
}

func TestSendVerificationCodeUsesMailerWhenEchoDisabled(t *testing.T) {
	db := newAuthTestDB(t)
	mailer := &fakeMailer{}
	service := NewAuthService(db, mailer)

	resp, err := service.SendVerificationCode("dave@zju.edu.cn", false)
	if err != nil {
		t.Fatalf("send verification code: %v", err)
	}
	if !resp.Sent {
		t.Fatalf("expected sent response")
	}
	if len(mailer.emails) != 1 || mailer.emails[0] != "dave@zju.edu.cn" {
		t.Fatalf("expected mailer to receive target email, got %#v", mailer.emails)
	}
	if len(mailer.codes) != 1 || len(mailer.codes[0]) != 6 {
		t.Fatalf("expected mailer to receive a 6-digit code, got %#v", mailer.codes)
	}
}
