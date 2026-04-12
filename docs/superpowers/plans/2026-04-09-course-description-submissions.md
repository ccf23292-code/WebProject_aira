# Course Description Submissions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the generic course-card capability labels with real course descriptions from the database and add a submission-review flow so normal users can propose description edits for admin approval.

**Architecture:** Keep `courses.description` as the live, published description. Add a new `course_description_submissions` table to store user proposals and their review state. Expose user submission endpoints under `/api/courses/:course_id/...` and admin review endpoints under `/api/admin/...`, then wire the course list/detail pages to read the live description and submit pending updates.

**Tech Stack:** Go, Gin, Gorm, SQLite test DB, Next.js, TypeScript

---

### Task 1: Add backend description submission workflow

**Files:**
- Modify: `back/models/course.go`
- Modify: `back/services/course_service.go`
- Modify: `back/routers/course_controller.go`
- Modify: `back/routers/admin_controller.go`
- Modify: `back/main.go`
- Modify: `back/services/course_service_test.go`
- Create: `back/routers/course_description_router_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestSubmitCourseDescriptionCreatesPendingSubmission(t *testing.T) {}
func TestApproveCourseDescriptionSubmissionPublishesDescription(t *testing.T) {}
func TestCourseDescriptionRoutesAreRegistered(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd back && go test ./services -run 'TestSubmitCourseDescriptionCreatesPendingSubmission|TestApproveCourseDescriptionSubmissionPublishesDescription' && go test ./routers -run TestCourseDescriptionRoutesAreRegistered`
Expected: FAIL because the model and routes do not exist yet.

- [ ] **Step 3: Implement the minimal backend**

```go
// back/models/course.go
// add CourseDescriptionSubmission model with pending/approved/rejected status

// back/services/course_service.go
// add submit/list-my/review methods that update courses.description on approval

// back/routers/course_controller.go
// add POST /courses/:course_id/description-submissions and GET /courses/:course_id/description-submissions/mine

// back/routers/admin_controller.go
// add GET /admin/course-description-submissions and POST /admin/course-description-submissions/:id/review
```

- [ ] **Step 4: Run backend verification**

Run: `cd back && go test ./...`
Expected: PASS

- [ ] **Step 5: Commit backend changes**

```bash
git add back/models/course.go back/services/course_service.go back/routers/course_controller.go back/routers/admin_controller.go back/main.go back/services/course_service_test.go back/routers/course_description_router_test.go
git commit -m "feat: add course description submission workflow"
```

### Task 2: Update course pages to use real descriptions

**Files:**
- Modify: `aira-web-4/packages/shared/src/types/index.ts`
- Modify: `aira-web-4/apps/web/src/app/courses/page.tsx`
- Modify: `aira-web-4/apps/web/src/app/courses/[courseId]/page.tsx`
- Create: `aira-web-4/apps/web/src/components/course/CourseDescriptionPanel.tsx`
- Create: `aira-web-4/apps/web/src/lib/courseDescription.ts`

- [ ] **Step 1: Implement typed API client and description panel**

```ts
// courseDescription.ts
// getMyDescriptionSubmissions(courseId), submitDescriptionSuggestion(courseId, content)

// CourseDescriptionPanel.tsx
// show published description, submission textarea, latest pending/rejected state
```

- [ ] **Step 2: Simplify course cards**

```tsx
// courses/page.tsx
// remove capability chips and render course.description as the only body copy
```

- [ ] **Step 3: Wire course detail page**

```tsx
// courses/[courseId]/page.tsx
// insert CourseDescriptionPanel above CourseCommunityPanel
```

- [ ] **Step 4: Run frontend verification**

Run: `cd aira-web-4/apps/web && npm run typecheck`
Expected: PASS

- [ ] **Step 5: Commit frontend changes**

```bash
git add aira-web-4/packages/shared/src/types/index.ts aira-web-4/apps/web/src/app/courses/page.tsx aira-web-4/apps/web/src/app/courses/[courseId]/page.tsx aira-web-4/apps/web/src/components/course/CourseDescriptionPanel.tsx aira-web-4/apps/web/src/lib/courseDescription.ts
git commit -m "feat: show course descriptions and submit edits"
```

### Task 3: Update docs and verify repository state

**Files:**
- Modify: `README.md`
- Modify: `docs/ARCHITECTURE_API.md`
- Modify: `docs/TODO.md`
- Modify: `docs/README.md`

- [ ] **Step 1: Document the new submission/review flow**

```md
- published description comes from `courses.description`
- users submit pending edits
- admins approve or reject
```

- [ ] **Step 2: Run final verification**

Run:
- `git diff --check`
- `cd back && go test ./...`
- `cd aira-web-4/apps/web && npm run typecheck`

Expected: all pass

- [ ] **Step 3: Commit docs**

```bash
git add README.md docs/ARCHITECTURE_API.md docs/TODO.md docs/README.md docs/superpowers/plans/2026-04-09-course-description-submissions.md
git commit -m "docs: record course description submission flow"
```
