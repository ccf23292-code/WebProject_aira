# Homepage Message Edit/Delete Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让首页留言支持作者本人编辑和删除，并保持现有公开展示与登录发布能力不变。

**Architecture:** 后端在现有 `homepage_messages` 模型上补充更新与删除服务方法，并通过鉴权校验“仅作者可操作”。前端在首页留言卡片上只对当前登录用户自己的留言显示“编辑 / 删除”，编辑采用原地展开表单，删除采用确认后直接提交。

**Tech Stack:** Go, Gin, Gorm, Next.js, TypeScript, React hooks

---

### Task 1: 后端留言编辑/删除

**Files:**
- Modify: `back/services/homepage_service.go`
- Modify: `back/routers/homepage_controller.go`
- Modify: `back/main.go`
- Test: `back/services/homepage_service_test.go`

- [ ] **Step 1: 写失败测试**

```go
func TestUpdateHomepageMessageAllowsAuthor(t *testing.T) {}
func TestUpdateHomepageMessageRejectsNonAuthor(t *testing.T) {}
func TestDeleteHomepageMessageAllowsAuthor(t *testing.T) {}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd back && go test ./services -run 'Test(UpdateHomepageMessageAllowsAuthor|UpdateHomepageMessageRejectsNonAuthor|DeleteHomepageMessageAllowsAuthor)$'`
Expected: FAIL because `UpdateMessage` / `DeleteMessage` do not exist

- [ ] **Step 3: 写最小实现**

```go
func (s *HomepageService) UpdateMessage(userID, messageID uint64, content string) (*models.HomepageMessage, error)
func (s *HomepageService) DeleteMessage(userID, messageID uint64) error
```

并在控制器里新增：

```go
group.PUT("/homepage/messages/:id", ctl.UpdateMessage)
group.DELETE("/homepage/messages/:id", ctl.DeleteMessage)
```

- [ ] **Step 4: 跑测试确认通过**

Run: `cd back && go test ./services -run 'Test(UpdateHomepageMessageAllowsAuthor|UpdateHomepageMessageRejectsNonAuthor|DeleteHomepageMessageAllowsAuthor)$'`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add back/services/homepage_service.go back/routers/homepage_controller.go back/main.go back/services/homepage_service_test.go
git commit -m "feat: add homepage message edit and delete api"
```

### Task 2: 前端留言卡片交互

**Files:**
- Modify: `aira-web-4/apps/web/src/lib/homepage.ts`
- Modify: `aira-web-4/packages/shared/src/types/index.ts`
- Modify: `aira-web-4/apps/web/src/app/page.tsx`

- [ ] **Step 1: 先补类型与 API**

```ts
export interface UpdateHomepageMessageDto { content: string }
export function updateHomepageMessage(id: number, payload: UpdateHomepageMessageDto) {}
export function deleteHomepageMessage(id: number) {}
```

- [ ] **Step 2: 在首页留言卡片增加作者态**

```ts
const isOwner = user?.userId === formatFrontendUserId(item.user_id)
```

并只对作者显示“编辑 / 删除”。

- [ ] **Step 3: 实现原地编辑与删除确认**

```tsx
{editing ? <textarea ... /> : <p>{item.content}</p>}
```

- [ ] **Step 4: 跑类型检查**

Run: `cd aira-web-4/apps/web && npm run typecheck`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add aira-web-4/apps/web/src/lib/homepage.ts aira-web-4/packages/shared/src/types/index.ts aira-web-4/apps/web/src/app/page.tsx
git commit -m "feat: support editing homepage messages"
```

### Task 3: 验证与推送

**Files:**
- Modify: `README.md` if behavior notes need update

- [ ] **Step 1: 运行完整验证**

```bash
git diff --check
cd back && go test ./...
cd aira-web-4/apps/web && npm run typecheck
```

- [ ] **Step 2: 推送**

```bash
git push origin dzz
```
