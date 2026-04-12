# Admin Review Center Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove backend bootstrap admin creation, seed the admin account from test scripts, and build a unified admin review center for course description, teacher info, and grading standard submissions.

**Architecture:** Move admin seeding into an explicit backend command consumed by `scripts/test-import.sh`, so database state comes from scripts instead of service startup side effects. Standardize user-contributed course metadata into submission tables with `pending/approved/rejected` review states, then expose dedicated admin list/review endpoints. Build a frontend admin review page with separate sections for each submission type and inline approve/reject actions.

**Tech Stack:** Go, Gin, Gorm, SQLite test DB, PostgreSQL, Next.js, TypeScript

---

### Task 1: Replace bootstrap admin with explicit seed command

**Files:**
- Modify: `back/services/auth_service.go`
- Modify: `back/services/auth_service_test.go`
- Create: `back/cmd/seed_admin/main.go`
- Modify: `scripts/test-import.sh`

- [ ] **Step 1: Write or adjust failing tests**

```go
func TestRegisterPersistsUserAndProfile(t *testing.T) {
    // assert registered user is profile user 1 once bootstrap admin is removed
}
```

- [ ] **Step 2: Run auth tests to verify failure after removing bootstrap assumptions**

Run: `cd back && go test ./services -run 'TestRegisterPersistsUserAndProfile|TestLogoutRemovesRefreshTokenSession|TestVerificationCodePersistsAcrossServiceInstances'`
Expected: FAIL after removing bootstrap or keeping stale test assumptions.

- [ ] **Step 3: Implement minimal admin seed command and script wiring**

```go
// back/cmd/seed_admin/main.go
// upsert admin user/profile from flags and hash password with bcrypt

// scripts/test-import.sh
// run go run ./cmd/seed_admin --username admin --password admin@123
```

- [ ] **Step 4: Run backend verification**

Run: `cd back && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit admin seed changes**

```bash
git add back/services/auth_service.go back/services/auth_service_test.go back/cmd/seed_admin/main.go scripts/test-import.sh
git commit -m "feat: seed admin from test import script"
```

### Task 2: Put teacher and grading submissions behind review flow

**Files:**
- Modify: `back/models/course.go`
- Modify: `back/services/course_service.go`
- Modify: `back/services/course_service_test.go`
- Modify: `back/routers/course_controller.go`
- Modify: `back/routers/admin_controller.go`
- Modify: `back/main.go`
- Modify: `back/routers/course_description_router_test.go`
- Create: `back/routers/admin_review_router_test.go`

- [ ] **Step 1: Write failing tests for teacher and grading review behavior**

```go
func TestAddTeacherCreatesPendingSubmission(t *testing.T) {}
func TestApproveTeacherSubmissionPublishesTeacher(t *testing.T) {}
func TestAddGradingStandardCreatesPendingSubmission(t *testing.T) {}
func TestApproveGradingSubmissionPublishesStandard(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd back && go test ./services -run 'TestAddTeacherCreatesPendingSubmission|TestApproveTeacherSubmissionPublishesTeacher|TestAddGradingStandardCreatesPendingSubmission|TestApproveGradingSubmissionPublishesStandard'`
Expected: FAIL because current teacher/grading writes publish immediately.

- [ ] **Step 3: Implement minimal submission workflow**

```go
// back/models/course.go
// add TeacherSubmission and GradingStandardSubmission models

// back/services/course_service.go
// make AddTeacher/AddGradingStandard create pending submissions
// add admin list/review methods that publish Teacher/GradingStandard on approval

// back/routers/admin_controller.go
// add list/review endpoints for teacher and grading submissions
```

- [ ] **Step 4: Run backend verification**

Run: `cd back && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit review workflow changes**

```bash
git add back/models/course.go back/services/course_service.go back/services/course_service_test.go back/routers/course_controller.go back/routers/admin_controller.go back/main.go back/routers/course_description_router_test.go back/routers/admin_review_router_test.go
git commit -m "feat: add review workflow for course metadata"
```

### Task 3: Build frontend admin review center and update contributor flows

**Files:**
- Modify: `aira-web-4/packages/shared/src/types/index.ts`
- Modify: `aira-web-4/apps/web/src/components/layout/Navbar.tsx`
- Modify: `aira-web-4/apps/web/src/components/course/CourseCommunityPanel.tsx`
- Modify: `aira-web-4/apps/web/src/components/course/CourseDescriptionPanel.tsx`
- Create: `aira-web-4/apps/web/src/lib/adminReview.ts`
- Create: `aira-web-4/apps/web/src/app/admin/reviews/page.tsx`

- [ ] **Step 1: Add frontend types and API client**

```ts
// shared types for teacher/grading submissions and admin review payloads
// adminReview.ts with list/review methods per submission type
```

- [ ] **Step 2: Implement review center page**

```tsx
// /admin/reviews
// three sections: course descriptions, teacher info, grading standards
// each row shows status, content, approve/reject buttons
```

- [ ] **Step 3: Update user-facing submission copy**

```tsx
// CourseCommunityPanel and CourseDescriptionPanel should say submissions require admin approval
// navbar should show 管理审核 for admins
```

- [ ] **Step 4: Run frontend verification**

Run: `cd aira-web-4/apps/web && npm run typecheck`
Expected: PASS

- [ ] **Step 5: Commit frontend review center**

```bash
git add aira-web-4/packages/shared/src/types/index.ts aira-web-4/apps/web/src/components/layout/Navbar.tsx aira-web-4/apps/web/src/components/course/CourseCommunityPanel.tsx aira-web-4/apps/web/src/components/course/CourseDescriptionPanel.tsx aira-web-4/apps/web/src/lib/adminReview.ts aira-web-4/apps/web/src/app/admin/reviews/page.tsx
git commit -m "feat: add admin review center"
```

### Task 4: Update docs and verify repository state

**Files:**
- Modify: `README.md`
- Modify: `docs/README.md`
- Modify: `docs/ARCHITECTURE_API.md`
- Modify: `docs/TODO.md`

- [ ] **Step 1: Document seed-admin and unified review center**
- [ ] **Step 2: Run final verification**

Run:
- `git diff --check`
- `cd back && go test ./...`
- `cd aira-web-4/apps/web && npm run typecheck`

Expected: all pass

- [ ] **Step 3: Commit docs**

```bash
git add README.md docs/README.md docs/ARCHITECTURE_API.md docs/TODO.md docs/superpowers/plans/2026-04-09-admin-review-center.md
git commit -m "docs: record admin review center"
```
