package services

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

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

// accessMetadata 存储 accessToken 对应的用户信息与过期时间。
type accessMetadata struct {
	UserID   uint64
	Username string
	Role     models.Role
	ExpAt    time.Time
}

// refreshMetadata 存储 refreshToken 的元数据。
type refreshMetadata struct {
	Username  string
	ExpiresAt time.Time
}

type verificationEntry struct {
	Code       string
	ExpiresAt  time.Time
	LastSentAt time.Time
}

// ────────────────────────── ServiceError ──────────────────────────

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

// ────────────────────────── AuthService ──────────────────────────

// AuthService 提供认证相关的业务逻辑（内存版本，无数据库依赖）。
type AuthService struct {
	mu            sync.RWMutex
	users         map[string]*models.User
	usersByEmail  map[string]*models.User
	accessTokens  map[string]*accessMetadata
	refreshTokens map[string]*refreshMetadata
	verifications map[string]*verificationEntry
	idSeq         models.PrimaryKey
}

// NewAuthService 创建并返回一个预置管理员账户的 AuthService。
func NewAuthService() *AuthService {
	svc := &AuthService{
		users:         make(map[string]*models.User),
		usersByEmail:  make(map[string]*models.User),
		accessTokens:  make(map[string]*accessMetadata),
		refreshTokens: make(map[string]*refreshMetadata),
		verifications: make(map[string]*verificationEntry),
	}
	svc.bootstrapAdmin()
	return svc
}

// ────────────────────────── 请求 / 响应 DTO ──────────────────────────

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

// ────────────────────────── 核心业务方法 ──────────────────────────

// Register 处理用户注册逻辑。
func (s *AuthService) Register(req RegisterRequest) (*RegisterResponse, error) {
	if err := s.validateRegister(req); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to hash password")
	}

	usernameKey := strings.ToLower(strings.TrimSpace(req.Username))
	emailKey := strings.ToLower(strings.TrimSpace(req.Email))
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[usernameKey]; exists {
		return nil, newServiceError("conflict", http.StatusConflict, "username already registered")
	}
	if _, exists := s.usersByEmail[emailKey]; exists {
		return nil, newServiceError("conflict", http.StatusConflict, "email already registered")
	}

	s.idSeq++
	user := &models.User{
		ID:           s.idSeq,
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         models.RoleStudent,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.users[usernameKey] = user
	s.usersByEmail[emailKey] = user

	access, refresh, _ := s.issueTokensLocked(user, false)

	return &RegisterResponse{
		AuthResponse: AuthResponse{
			UserID:       formatUserID(user.ID),
			DisplayName:  user.Username,
			AccessToken:  access,
			RefreshToken: refresh,
			Roles:        []string{string(user.Role)},
			ExpiresIn:    int64(accessTokenTTL.Seconds()),
		},
		OnboardingTasks: []string{"complete_profile", "bind_courses"},
	}, nil
}

// Login 处理用户登录逻辑。
func (s *AuthService) Login(req LoginRequest) (*AuthResponse, error) {
	if err := s.validateLogin(req); err != nil {
		return nil, err
	}

	usernameKey := strings.ToLower(strings.TrimSpace(req.Username))

	s.mu.RLock()
	user, exists := s.users[usernameKey]
	s.mu.RUnlock()

	if !exists {
		return nil, newServiceError("invalid_credentials", http.StatusUnauthorized, "username or password is incorrect")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, newServiceError("invalid_credentials", http.StatusUnauthorized, "username or password is incorrect")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	access, refresh, _ := s.issueTokensLocked(user, req.RememberMe)

	return &AuthResponse{
		UserID:       formatUserID(user.ID),
		DisplayName:  user.Username,
		AccessToken:  access,
		RefreshToken: refresh,
		Roles:        []string{string(user.Role)},
		ExpiresIn:    int64(accessTokenTTL.Seconds()),
	}, nil
}

// Logout 删除对应的 refreshToken。
func (s *AuthService) Logout(req LogoutRequest) (*LogoutResponse, error) {
	if strings.TrimSpace(req.RefreshToken) == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "refreshToken is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.refreshTokens[req.RefreshToken]; exists {
		delete(s.refreshTokens, req.RefreshToken)
		return &LogoutResponse{Success: true, Message: "logout successful"}, nil
	}
	return &LogoutResponse{Success: true, Message: "token already invalidated"}, nil
}

// ────────────────────────── Token 验证（供中间件调用） ──────────────────────────

// TokenClaims 是中间件从 accessToken 中提取的用户声明。
type TokenClaims struct {
	UserID   uint64
	Username string
	Role     models.Role
}

// ValidateAccessToken 校验 accessToken 的有效性并返回用户信息。
func (s *AuthService) ValidateAccessToken(token string) (*TokenClaims, error) {
	s.mu.RLock()
	meta, ok := s.accessTokens[token]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("invalid token")
	}
	if time.Now().After(meta.ExpAt) {
		s.mu.Lock()
		delete(s.accessTokens, token)
		s.mu.Unlock()
		return nil, fmt.Errorf("token expired")
	}
	return &TokenClaims{
		UserID:   meta.UserID,
		Username: meta.Username,
		Role:     meta.Role,
	}, nil
}

// ────────────────────────── 内部辅助方法 ──────────────────────────

func (s *AuthService) issueTokensLocked(user *models.User, rememberMe bool) (string, string, time.Duration) {
	s.gcLocked()
	s.revokeUserTokensLocked(strings.ToLower(user.Username))

	access := generateToken()
	refresh := generateToken()
	ttl := defaultRefreshToken
	if rememberMe {
		ttl = rememberMeRefreshTTL
	}

	s.accessTokens[access] = &accessMetadata{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		ExpAt:    time.Now().Add(accessTokenTTL),
	}
	s.refreshTokens[refresh] = &refreshMetadata{
		Username:  strings.ToLower(user.Username),
		ExpiresAt: time.Now().Add(ttl),
	}
	user.RememberToken = models.Varchar(refresh)
	return access, refresh, ttl
}

func (s *AuthService) revokeUserTokensLocked(username string) {
	for token, meta := range s.refreshTokens {
		if meta.Username == username {
			delete(s.refreshTokens, token)
		}
	}
	for token, meta := range s.accessTokens {
		if strings.ToLower(meta.Username) == username {
			delete(s.accessTokens, token)
		}
	}
}

func (s *AuthService) gcLocked() {
	now := time.Now()
	for token, meta := range s.refreshTokens {
		if now.After(meta.ExpiresAt) {
			delete(s.refreshTokens, token)
		}
	}
	for token, meta := range s.accessTokens {
		if now.After(meta.ExpAt) {
			delete(s.accessTokens, token)
		}
	}
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
	if err := s.validateVerificationCode(email, req.VerificationCode); err != nil {
		return err
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

func (s *AuthService) bootstrapAdmin() {
	password := "Admin@123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	s.idSeq++
	now := time.Now().UTC()
	admin := &models.User{
		ID:           s.idSeq,
		Username:     "admin",
		Email:        "admin@example.com",
		PasswordHash: string(hash),
		Role:         models.RoleAdmin,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.users[strings.ToLower(admin.Username)] = admin
	s.usersByEmail[strings.ToLower(admin.Email)] = admin
}

// SendVerificationCode 生成并缓存邮箱验证码（开发环境可选择回显）。
func (s *AuthService) SendVerificationCode(email string, echo bool) (*VerificationCodeResponse, error) {
	email = strings.TrimSpace(email)
	if !emailRegex.MatchString(email) {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "email format is invalid")
	}

	now := time.Now().UTC()
	emailKey := strings.ToLower(email)

	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, ok := s.verifications[emailKey]; ok {
		if now.Sub(entry.LastSentAt) < verificationCooldown {
			return nil, newServiceError("too_many_requests", http.StatusTooManyRequests, "verification code sent too frequently")
		}
	}

	code := generateVerificationCode()
	s.verifications[emailKey] = &verificationEntry{
		Code:       code,
		ExpiresAt:  now.Add(verificationCodeTTL),
		LastSentAt: now,
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

func (s *AuthService) validateVerificationCode(email, code string) *ServiceError {
	emailKey := strings.ToLower(strings.TrimSpace(email))
	code = strings.TrimSpace(code)

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.verifications[emailKey]
	if !ok {
		return newServiceError("invalid_verification_code", http.StatusUnprocessableEntity, "verification code invalid or expired")
	}
	if time.Now().UTC().After(entry.ExpiresAt) {
		delete(s.verifications, emailKey)
		return newServiceError("invalid_verification_code", http.StatusUnprocessableEntity, "verification code invalid or expired")
	}
	if entry.Code != code {
		return newServiceError("invalid_verification_code", http.StatusUnprocessableEntity, "verification code invalid or expired")
	}

	delete(s.verifications, emailKey)
	return nil
}
