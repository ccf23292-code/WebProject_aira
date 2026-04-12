# Course Community Closure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the course community loop so course search, course comments, teacher comments, grading standards, and teacher directory all work against the backend contract instead of split frontend fallbacks.

**Architecture:** Keep the existing course detail workspace UI, but move the missing teacher directory and read-path contract into the backend. Backend becomes the source of truth for course community data; frontend keeps only minimal request/normalization logic and removes localStorage-driven behavior. Use small service-layer tests in Go to drive backend contract changes, then wire the frontend to the stabilized endpoints.

**Tech Stack:** Go, Gin, GORM, SQLite test DB, Next.js, TypeScript

---

### Task 1: Stabilize course community backend contract

**Files:**
- Modify: `back/models/course.go`
- Modify: `back/services/course_service.go`
- Modify: `back/routers/paper_controller.go`
- Modify: `back/routers/course_controller.go`
- Modify: `back/main.go`
- Create: `back/services/course_service_test.go`

- [ ] **Step 1: Write failing backend tests for search alias, teacher directory, and grading route behavior**

```go
func TestListCoursesAcceptsQueryAlias(t *testing.T) { /* seeds course and asserts query search works */ }
func TestTeacherDirectoryIsPersistedPerCourse(t *testing.T) { /* creates teacher and asserts list returns it */ }
func TestCommentsAndGradingReturnDisplayFields(t *testing.T) { /* creates teacher/comment/grading and asserts names/timestamps */ }
```

- [ ] **Step 2: Run backend tests to verify they fail**

Run: `cd back && go test ./services -run 'TestListCoursesAcceptsQueryAlias|TestTeacherDirectoryIsPersistedPerCourse|TestCommentsAndGradingReturnDisplayFields'`
Expected: FAIL because teacher persistence and normalized read contract are incomplete.

- [ ] **Step 3: Add the minimal backend implementation**

```go
// back/models/course.go
// Add Teacher timestamps and denormalized JSON view fields for comment/grading responses.

// back/services/course_service.go
// Add ListTeachers/CreateTeacher, normalize display names, accept query alias through router.

// back/routers/paper_controller.go
// Register GET comment/grading/teacher routes and read query from `query` first, fallback to `q`.

// back/routers/course_controller.go
// Register POST /teachers and POST /grading-standards.
```

- [ ] **Step 4: Run backend tests to verify they pass**

Run: `cd back && go test ./services -run 'TestListCoursesAcceptsQueryAlias|TestTeacherDirectoryIsPersistedPerCourse|TestCommentsAndGradingReturnDisplayFields'`
Expected: PASS

- [ ] **Step 5: Run full backend verification**

Run: `cd back && go test ./...`
Expected: PASS

- [ ] **Step 6: Commit backend contract changes**

```bash
git add back/models/course.go back/services/course_service.go back/routers/paper_controller.go back/routers/course_controller.go back/main.go back/services/course_service_test.go
git commit -m "feat: close course community backend contract"
```

### Task 2: Remove frontend local fallback and align with backend API

**Files:**
- Modify: `aira-web-4/apps/web/src/lib/courseCommunity.ts`
- Modify: `aira-web-4/apps/web/src/components/course/CourseCommunityPanel.tsx`
- Modify: `aira-web-4/apps/web/src/app/courses/page.tsx`
- Modify: `aira-web-4/apps/web/src/app/courses/[courseId]/page.tsx`
- Modify: `aira-web-4/packages/shared/src/types/index.ts`

- [ ] **Step 1: Write the frontend-facing contract checklist before code changes**

```text
- search uses `query`
- teacher directory comes from backend
- grading uses `/grading-standards`
- comments and grading read server values first
- no localStorage fallback required for the happy path
```

- [ ] **Step 2: Implement minimal frontend integration changes**

```ts
// courseCommunity.ts
// replace teacher-directory localStorage helpers with API calls
// remove fallback write paths

// CourseCommunityPanel.tsx
// fetch teacher list from backend, create teachers through backend, keep selected teacher state stable

// courses/page.tsx
// keep using `query`, matching backend alias support
```

- [ ] **Step 3: Run frontend typecheck**

Run: `cd aira-web-4/apps/web && npm run typecheck`
Expected: PASS

- [ ] **Step 4: Commit frontend integration changes**

```bash
git add aira-web-4/apps/web/src/lib/courseCommunity.ts aira-web-4/apps/web/src/components/course/CourseCommunityPanel.tsx aira-web-4/apps/web/src/app/courses/page.tsx aira-web-4/apps/web/src/app/courses/[courseId]/page.tsx aira-web-4/packages/shared/src/types/index.ts
git commit -m "feat: align course community frontend with backend"
```

### Task 3: Update docs and verify the merged branch state

**Files:**
- Modify: `docs/ARCHITECTURE_API.md`
- Modify: `docs/TODO.md`
- Modify: `README.md`

- [ ] **Step 1: Update API docs with the final course community contract**

```md
- GET /api/courses?query=
- GET /api/courses/:course_id/teachers
- GET/POST /api/courses/:course_id/comments
- GET/POST /api/courses/:course_id/teachers/:teacher_id/comments
- GET/POST /api/courses/:course_id/teachers/:teacher_id/grading-standards
```

- [ ] **Step 2: Mark this closure stage in the roadmap**

```md
P1.0: course community contract closed
- backend read/write aligned
- teacher directory persisted
- frontend fallback removed
```

- [ ] **Step 3: Run final repository verification**

Run:
- `git diff --check`
- `cd back && go test ./...`
- `cd aira-web-4/apps/web && npm run typecheck`

Expected: all pass

- [ ] **Step 4: Commit docs and status update**

```bash
git add README.md docs/ARCHITECTURE_API.md docs/TODO.md docs/superpowers/plans/2026-04-09-course-community-closure.md
git commit -m "docs: record course community closure stage"
```
