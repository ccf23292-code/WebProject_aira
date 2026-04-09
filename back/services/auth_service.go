package services

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"warehouse-web/models"
)

const (
	accessTokenTTL       = 30 * time.Minute
	defaultRefreshToken  = 7 * 24 * time.Hour
	rememberMeRefreshTTL = 30 * 24 * time.Hour
	verificationCodeTTL  = 10 * time.Minute
	verificationCooldown = 60 * time.Second
)

var emailRegex = regexp.MustCompile(`^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$`)

// ServiceError 封装业务层错误码及 HTTP 状态。
type ServiceError struct {
	Code    string
	Message string
	Status  int
}

func (e *ServiceError) Error() string { return e.Message }

func newServiceError(code string, status int, message string) *ServiceError {
	return &ServiceError{Code: code, Status: status, Message: message}
}

// AuthService 提供认证相关的业务逻辑（数据库版本）。
type AuthService struct {
	db *gorm.DB
}

// NewAuthService 创建数据库版认证服务，并确保默认管理员存在。
func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{db: db}
}

type LoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	OTP        string `json:"otp"`
	RememberMe bool   `json:"rememberMe"`
}

type RegisterRequest struct {
	Username         string `json:"username"`
	Email            string `json:"email"`
	Password         string `json:"password"`
	ConfirmPassword  string `json:"confirmPassword"`
	VerificationCode string `json:"verificationCode"`
	AgreeToPolicy    bool   `json:"agreeToPolicy"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type VerificationCodeRequest struct {
	Email string `json:"email"`
}

type AuthResponse struct {
	UserID       string   `json:"userId"`
	DisplayName  string   `json:"displayName"`
	AccessToken  string   `json:"accessToken"`
	RefreshToken string   `json:"refreshToken"`
	Roles        []string `json:"roles"`
	ExpiresIn    int64    `json:"expiresIn"`
}

type RegisterResponse struct {
	AuthResponse
	OnboardingTasks []string `json:"onboardingTasks"`
}

type LogoutResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type VerificationCodeResponse struct {
	Sent      bool   `json:"sent"`
	Code      string `json:"code,omitempty"`
	ExpiresIn int64  `json:"expiresIn"`
}

// TokenClaims 是中间件从 accessToken 中提取的用户声明。
type TokenClaims struct {
	UserID   uint64
	Username string
	Role     models.Role
}

func (s *AuthService) Register(req RegisterRequest) (*RegisterResponse, error) {
	if err := s.validateRegister(req); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to hash password")
	}

	username := strings.TrimSpace(req.Username)
	email := strings.TrimSpace(req.Email)
	usernameKey := strings.ToLower(username)
	emailKey := strings.ToLower(email)
	now := time.Now().UTC()

	var response *RegisterResponse
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if exists, checkErr := userExistsByUsername(tx, usernameKey); checkErr != nil {
			return checkErr
		} else if exists {
			return newServiceError("conflict", http.StatusConflict, "username already registered")
		}
		if exists, checkErr := userExistsByEmail(tx, emailKey); checkErr != nil {
			return checkErr
		} else if exists {
			return newServiceError("conflict", http.StatusConflict, "email already registered")
		}

		if verifyErr := s.validateVerificationCode(tx, email, req.VerificationCode); verifyErr != nil {
			return verifyErr
		}

		user := &models.User{
			Username:     username,
			Email:        email,
			PasswordHash: string(hash),
			Role:         models.RoleStudent,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if createErr := tx.Create(user).Error; createErr != nil {
			return newServiceError("internal_error", http.StatusInternalServerError, "failed to create user")
		}

		if createErr := tx.Create(&models.UserProfile{
			UserID:    user.ID,
			Nickname:  username,
			AvatarURL: "",
			Level:     1,
			CreatedAt: now,
			UpdatedAt: now,
		}).Error; createErr != nil {
			return newServiceError("internal_error", http.StatusInternalServerError, "failed to create profile")
		}

		access, refresh, issueErr := s.issueTokens(tx, user, false)
		if issueErr != nil {
			return issueErr
		}

		response = &RegisterResponse{
			AuthResponse: AuthResponse{
				UserID:       formatUserID(user.ID),
				DisplayName:  username,
				AccessToken:  access,
				RefreshToken: refresh,
				Roles:        []string{string(user.Role)},
				ExpiresIn:    int64(accessTokenTTL.Seconds()),
			},
			OnboardingTasks: []string{"complete_profile", "bind_courses"},
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (s *AuthService) Login(req LoginRequest) (*AuthResponse, error) {
	if err := s.validateLogin(req); err != nil {
		return nil, err
	}

	usernameKey := strings.ToLower(strings.TrimSpace(req.Username))

	var user models.User
	if err := s.db.Where("LOWER(username) = ?", usernameKey).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, newServiceError("invalid_credentials", http.StatusUnauthorized, "username or password is incorrect")
		}
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load user")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, newServiceError("invalid_credentials", http.StatusUnauthorized, "username or password is incorrect")
	}

	var (
		displayName  string
		accessToken  string
		refreshToken string
	)
	err := s.db.Transaction(func(tx *gorm.DB) error {
		access, refresh, issueErr := s.issueTokens(tx, &user, req.RememberMe)
		if issueErr != nil {
			return issueErr
		}
		displayName = s.loadDisplayName(tx, user.ID, user.Username)
		accessToken = access
		refreshToken = refresh
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		UserID:       formatUserID(user.ID),
		DisplayName:  displayName,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Roles:        []string{string(user.Role)},
		ExpiresIn:    int64(accessTokenTTL.Seconds()),
	}, nil
}

func (s *AuthService) Logout(req LogoutRequest) (*LogoutResponse, error) {
	if strings.TrimSpace(req.RefreshToken) == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "refreshToken is required")
	}

	res := s.db.Where("refresh_token = ?", strings.TrimSpace(req.RefreshToken)).Delete(&models.AuthSession{})
	if res.Error != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to remove session")
	}
	if res.RowsAffected == 0 {
		return &LogoutResponse{Success: true, Message: "token already invalidated"}, nil
	}
	return &LogoutResponse{Success: true, Message: "logout successful"}, nil
}

func (s *AuthService) ValidateAccessToken(token string) (*TokenClaims, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("invalid token")
	}

	var session models.AuthSession
	if err := s.db.Where("access_token = ?", token).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("invalid token")
		}
		return nil, fmt.Errorf("failed to load session")
	}
	if time.Now().UTC().After(session.AccessExpiresAt) {
		_ = s.db.Delete(&session).Error
		return nil, fmt.Errorf("token expired")
	}

	var user models.User
	if err := s.db.Where("id = ?", session.UserID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			_ = s.db.Delete(&session).Error
			return nil, fmt.Errorf("user missing")
		}
		return nil, fmt.Errorf("failed to load user")
	}

	return &TokenClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
	}, nil
}

// SendVerificationCode 生成并缓存邮箱验证码（开发环境可选择回显）。
func (s *AuthService) SendVerificationCode(email string, echo bool) (*VerificationCodeResponse, error) {
	email = strings.TrimSpace(email)
	if !emailRegex.MatchString(email) {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "email format is invalid")
	}

	emailKey := strings.ToLower(email)
	now := time.Now().UTC()

	var entry models.EmailVerification
	err := s.db.Where("email = ?", emailKey).First(&entry).Error
	if err == nil && now.Sub(entry.LastSentAt) < verificationCooldown {
		return nil, newServiceError("too_many_requests", http.StatusTooManyRequests, "verification code sent too frequently")
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to save verification code")
	}

	code := generateVerificationCode()
	if err == nil {
		entry.Code = code
		entry.ExpiresAt = now.Add(verificationCodeTTL)
		entry.LastSentAt = now
		if saveErr := s.db.Save(&entry).Error; saveErr != nil {
			return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to save verification code")
		}
	} else {
		entry = models.EmailVerification{
			Email:      emailKey,
			Code:       code,
			ExpiresAt:  now.Add(verificationCodeTTL),
			LastSentAt: now,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if createErr := s.db.Create(&entry).Error; createErr != nil {
			return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to save verification code")
		}
	}

	resp := &VerificationCodeResponse{
		Sent:      true,
		ExpiresIn: int64(verificationCodeTTL.Seconds()),
	}
	if echo {
		resp.Code = code
	}
	return resp, nil
}

func (s *AuthService) validateRegister(req RegisterRequest) *ServiceError {
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return newServiceError("invalid_request", http.StatusBadRequest, "username is required")
	}
	if len(username) > 64 {
		return newServiceError("invalid_request", http.StatusBadRequest, "username must be <= 64 characters")
	}
	email := strings.TrimSpace(req.Email)
	if !emailRegex.MatchString(email) {
		return newServiceError("invalid_request", http.StatusBadRequest, "email format is invalid")
	}
	if err := validatePassword(req.Password); err != nil {
		return newServiceError("invalid_request", http.StatusBadRequest, err.Error())
	}
	if req.Password != req.ConfirmPassword {
		return newServiceError("invalid_request", http.StatusBadRequest, "password and confirmPassword do not match")
	}
	if len(strings.TrimSpace(req.VerificationCode)) != 6 {
		return newServiceError("invalid_verification_code", http.StatusUnprocessableEntity, "verification code must be 6 characters")
	}
	if !req.AgreeToPolicy {
		return newServiceError("policy_not_accepted", http.StatusBadRequest, "policy must be accepted before registration")
	}
	return nil
}

func (s *AuthService) validateLogin(req LoginRequest) *ServiceError {
	if strings.TrimSpace(req.Username) == "" {
		return newServiceError("invalid_request", http.StatusBadRequest, "username is required")
	}
	if strings.TrimSpace(req.Password) == "" {
		return newServiceError("invalid_request", http.StatusBadRequest, "password is required")
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}
	var hasLetter, hasNumber bool
	for _, r := range password {
		switch {
		case r >= '0' && r <= '9':
			hasNumber = true
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			hasLetter = true
		}
	}
	if !hasLetter || !hasNumber {
		return fmt.Errorf("password must contain both letters and numbers")
	}
	return nil
}

func generateToken() string { return uuid.NewString() }

func generateVerificationCode() string {
	const digits = "0123456789"
	b := make([]byte, 6)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			b[i] = digits[time.Now().UnixNano()%10]
			continue
		}
		b[i] = digits[n.Int64()]
	}
	return string(b)
}

func formatUserID(id models.PrimaryKey) string {
	return fmt.Sprintf("u-%08d", id)
}

func (s *AuthService) issueTokens(tx *gorm.DB, user *models.User, rememberMe bool) (string, string, error) {
	ttl := defaultRefreshToken
	if rememberMe {
		ttl = rememberMeRefreshTTL
	}
	access := generateToken()
	refresh := generateToken()
	now := time.Now().UTC()

	if err := tx.Where("user_id = ?", user.ID).Delete(&models.AuthSession{}).Error; err != nil {
		return "", "", newServiceError("internal_error", http.StatusInternalServerError, "failed to clear old sessions")
	}

	session := models.AuthSession{
		UserID:           user.ID,
		AccessToken:      access,
		RefreshToken:     refresh,
		AccessExpiresAt:  now.Add(accessTokenTTL),
		RefreshExpiresAt: now.Add(ttl),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := tx.Create(&session).Error; err != nil {
		return "", "", newServiceError("internal_error", http.StatusInternalServerError, "failed to create session")
	}

	user.RememberToken = models.Varchar(refresh)
	if err := tx.Save(user).Error; err != nil {
		return "", "", newServiceError("internal_error", http.StatusInternalServerError, "failed to update user session state")
	}
	return access, refresh, nil
}

func (s *AuthService) validateVerificationCode(tx *gorm.DB, email, code string) *ServiceError {
	emailKey := strings.ToLower(strings.TrimSpace(email))
	code = strings.TrimSpace(code)

	var entry models.EmailVerification
	if err := tx.Where("email = ?", emailKey).First(&entry).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return newServiceError("invalid_verification_code", http.StatusUnprocessableEntity, "verification code invalid or expired")
		}
		return newServiceError("internal_error", http.StatusInternalServerError, "failed to load verification code")
	}
	if time.Now().UTC().After(entry.ExpiresAt) || entry.Code != code {
		_ = tx.Delete(&entry).Error
		return newServiceError("invalid_verification_code", http.StatusUnprocessableEntity, "verification code invalid or expired")
	}
	if err := tx.Delete(&entry).Error; err != nil {
		return newServiceError("internal_error", http.StatusInternalServerError, "failed to clear verification code")
	}
	return nil
}

func (s *AuthService) loadDisplayName(tx *gorm.DB, userID uint64, fallback string) string {
	var profile models.UserProfile
	if err := tx.Where("user_id = ?", userID).First(&profile).Error; err == nil && strings.TrimSpace(profile.Nickname) != "" {
		return profile.Nickname
	}
	return fallback
}

func userExistsByUsername(db *gorm.DB, usernameKey string) (bool, *ServiceError) {
	var count int64
	if err := db.Model(&models.User{}).Where("LOWER(username) = ?", usernameKey).Count(&count).Error; err != nil {
		return false, newServiceError("internal_error", http.StatusInternalServerError, "failed to check username")
	}
	return count > 0, nil
}

func userExistsByEmail(db *gorm.DB, emailKey string) (bool, *ServiceError) {
	var count int64
	if err := db.Model(&models.User{}).Where("LOWER(email) = ?", emailKey).Count(&count).Error; err != nil {
		return false, newServiceError("internal_error", http.StatusInternalServerError, "failed to check email")
	}
	return count > 0, nil
}
