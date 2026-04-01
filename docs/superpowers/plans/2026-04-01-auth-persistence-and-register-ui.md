# Auth Persistence And Register UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Persist users and auth tokens in Postgres so accounts survive backend restarts, and update the register form to support password visibility plus the requested field order.

**Architecture:** Replace the current in-memory `AuthService` storage with Gorm-backed tables for users, verification codes, and auth sessions. Keep profile, favorites, wrongbook, and answer-record tables keyed by the same `user_id`, and extend profile responses with stable user-facing fields such as nickname, avatar, and level so auth and profile layers stay aligned.

**Tech Stack:** Go, Gin, Gorm, PostgreSQL, Next.js, React, TypeScript

---

### Task 1: Persist auth state in the database

**Files:**
- Create: `back/models/auth_session.go`
- Create: `back/models/email_verification.go`
- Modify: `back/services/auth_service.go`
- Modify: `back/main.go`
- Test: `back/services/auth_service_test.go`

- [ ] **Step 1: Write failing auth persistence tests**

```go
func TestRegisterPersistsUserAndProfile(t *testing.T) { /* register -> restart service -> login succeeds */ }
func TestLogoutRemovesRefreshTokenSession(t *testing.T) { /* logout -> token invalid */ }
func TestVerificationCodePersistsAcrossServiceInstances(t *testing.T) { /* send code -> rebuild service -> register works */ }
```

- [ ] **Step 2: Run the failing auth tests**

Run: `cd back && go test ./services -run 'Test(RegisterPersistsUserAndProfile|LogoutRemovesRefreshTokenSession|VerificationCodePersistsAcrossServiceInstances)$'`
Expected: FAIL because `AuthService` still uses in-memory maps and service restart loses state.

- [ ] **Step 3: Add database-backed auth models**

```go
type AuthSession struct {
  ID           uint64
  UserID       uint64
  AccessToken  string
  RefreshToken string
  AccessExpAt  time.Time
  RefreshExpAt time.Time
}

type EmailVerification struct {
  ID         uint64
  Email      string
  Code       string
  ExpiresAt  time.Time
  LastSentAt time.Time
}
```

- [ ] **Step 4: Rebuild `AuthService` around Gorm**

```go
type AuthService struct {
  db *gorm.DB
}

func NewAuthService(db *gorm.DB) *AuthService { ... }
func (s *AuthService) Register(...) (*RegisterResponse, error) { /* create user + default profile */ }
func (s *AuthService) Login(...) (*AuthResponse, error) { /* query DB, bcrypt compare, create session row */ }
func (s *AuthService) ValidateAccessToken(token string) (*TokenClaims, error) { /* DB lookup */ }
func (s *AuthService) Logout(...) (*LogoutResponse, error) { /* delete session row */ }
func (s *AuthService) SendVerificationCode(...) (*VerificationCodeResponse, error) { /* upsert verification row */ }
```

- [ ] **Step 5: Migrate new auth tables and update wiring**

Run: update `back/main.go` so `NewAuthService(db)` is used and `AutoMigrate` includes `AuthSession` and `EmailVerification`.

- [ ] **Step 6: Run the auth persistence tests again**

Run: `cd back && go test ./services -run 'Test(RegisterPersistsUserAndProfile|LogoutRemovesRefreshTokenSession|VerificationCodePersistsAcrossServiceInstances)$'`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add back/models/auth_session.go back/models/email_verification.go back/services/auth_service.go back/services/auth_service_test.go back/main.go
git commit -m "feat: persist auth users and sessions in database"
```

### Task 2: Align profile data with persisted users

**Files:**
- Modify: `back/models/user_profile.go`
- Modify: `back/services/profile_service.go`
- Modify: `back/routers/profile_controller.go`
- Modify: `aira-web-4/packages/shared/src/types/index.ts`
- Test: `back/services/profile_service_test.go`

- [ ] **Step 1: Write failing profile test**

```go
func TestGetProfileReturnsLevelAndIdentityFields(t *testing.T) { /* profile should expose nickname/avatar/level/username/email */ }
```

- [ ] **Step 2: Run the failing profile test**

Run: `cd back && go test ./services -run '^TestGetProfileReturnsLevelAndIdentityFields$'`
Expected: FAIL because profile model/response lacks the new fields.

- [ ] **Step 3: Extend profile model and response shape**

```go
type UserProfile struct {
  ...
  Level int `gorm:"default:1"`
}

type ProfileView struct {
  ID        uint64
  UserID    uint64
  Username  string
  Email     string
  Nickname  string
  AvatarURL string
  Level     int
}
```

- [ ] **Step 4: Make `GetProfile` return merged user + profile data**

Use `users` + `user_profiles` so profile APIs expose the stable identity fields the frontend depends on.

- [ ] **Step 5: Run the profile test again**

Run: `cd back && go test ./services -run '^TestGetProfileReturnsLevelAndIdentityFields$'`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add back/models/user_profile.go back/services/profile_service.go back/routers/profile_controller.go aira-web-4/packages/shared/src/types/index.ts
git commit -m "feat: expose persistent user profile fields"
```

### Task 3: Update register UI and password visibility

**Files:**
- Modify: `aira-web-4/apps/web/src/app/register/page.tsx`
- Optionally Create: `aira-web-4/apps/web/src/components/form/PasswordInput.tsx`
- Test: `aira-web-4/apps/web` typecheck

- [ ] **Step 1: Adjust register form layout**

Field order must become:

```text
用户名
密码
再次输入密码
邮箱（右侧保留获取验证码按钮）
验证码
```

- [ ] **Step 2: Add password visibility toggles**

Implement one reusable password input with a right-side eye button for both password fields.

- [ ] **Step 3: Preserve existing validation behavior**

Keep:
- password / confirm match check
- 6-digit verification code check
- policy checkbox

- [ ] **Step 4: Run frontend typecheck**

Run: `cd aira-web-4/apps/web && npm run typecheck`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add aira-web-4/apps/web/src/app/register/page.tsx aira-web-4/apps/web/src/components/form/PasswordInput.tsx
git commit -m "feat: improve register form usability"
```

### Task 4: Final regression verification

**Files:**
- Modify: `docs/TODO.md` (only if auth persistence status needs documenting)

- [ ] **Step 1: Run backend test suite**

Run: `cd back && go test ./...`
Expected: PASS

- [ ] **Step 2: Run frontend typecheck**

Run: `cd aira-web-4/apps/web && npm run typecheck`
Expected: PASS

- [ ] **Step 3: Manual verification checklist**

Verify locally:
- register a new account
- restart with `scripts/dev.sh`
- login with the same account still works
- password fields support show/hide
- profile API still returns nickname/avatar/level

- [ ] **Step 4: Commit final docs update if needed**

```bash
git add docs/TODO.md
git commit -m "docs: update auth persistence progress"
```
