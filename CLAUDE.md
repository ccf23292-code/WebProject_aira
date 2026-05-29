# AIRAWeb 项目协作指南

本项目是一个**面向高校课程的在线刷题 / 课程评价 / 智能复习平台**。

> 这份文件供 Claude Code 等 AI 协作工具自动读取，也供新加入的开发者快速上手。
> 更新时请保持简洁、聚焦于"AI 助手 / 新成员真正需要的信息"，详细使用文档请放 README。

---

## 仓库结构

```
.
├── aira-web-4/              # 前端 (Next.js monorepo)
│   ├── apps/web/            # 主应用
│   │   ├── src/app/         # App Router 路由（文件路径 = URL）
│   │   ├── src/components/  # 可复用 React 组件
│   │   ├── src/lib/         # 业务工具（api、auth、各模块辅助）
│   │   └── public/          # 静态资源
│   └── packages/shared/     # 前后端共享 TS 类型 (@aira/shared)
├── back/                    # Go 后端
│   ├── models/              # GORM 数据模型
│   ├── services/            # 业务逻辑
│   ├── routers/             # Gin HTTP 控制器
│   ├── middlewares/         # 认证、HTTPS、CORS
│   ├── utils/               # 统一响应封装
│   ├── cmd/                 # 独立命令（worker, seed, import）
│   └── main.go              # 入口
├── data/                    # 题目/课程导入数据
└── scripts/                 # 启动 / 导入脚本
```

---

## 技术栈

**前端**：Next.js 16 (App Router) + TypeScript + Tailwind CSS + react-markdown + KaTeX  
**后端**：Go 1.22 + Gin + GORM + PostgreSQL + JWT + SMTP + OpenAI SDK  
**认证**：Bearer token (存 localStorage)  
**响应格式**：`{ code, message, data }` 统一包装

---

## 本地开发

### 前端
```bash
cd aira-web-4
npm install
npm run dev:web      # 监听 :3000
npm run typecheck    # 跨 workspace 跑 tsc（提交前务必跑）
```

### 后端
```bash
cd back
cp .env.example .env
# 填 DATABASE_URL, AUTH_SECRET
# 开发建议: DEV_EMAIL_ECHO=true, REQUIRE_HTTPS=false
go run main.go       # 监听 :3001
```

### 跨服务联调
前端默认调 `http://localhost:3001/api`，在 `aira-web-4/apps/web/src/lib/api.ts` 改 `API_BASE` 或设环境变量 `NEXT_PUBLIC_API_URL`。

---

## 代码约定（patterns）

### 1. 后端三层架构

每个业务模块都按 `models / services / routers` 三层组织：

```go
// models/foo.go           — GORM struct
type Foo struct {
  ID PrimaryKey `gorm:"primaryKey;autoIncrement" json:"id"`
  ...
}

// services/foo_service.go — 业务逻辑
type FooService struct { db *gorm.DB }
func (s *FooService) Do(...) (*Result, error) {
  if invalid {
    return nil, newServiceError("invalid_request", http.StatusBadRequest, "原因")
  }
  ...
}

// routers/foo_controller.go — HTTP 处理
func (ctl *FooController) Do(c *gin.Context) {
  result, err := ctl.service.Do(...)
  if err != nil { ctl.handleError(c, err); return }
  utils.JSONSuccess(c, http.StatusOK, result)
}
```

新建 controller 时**参考 `routers/problem_explanation_controller.go`**，它有完整的 parseParam / currentUserID / handleError 辅助方法。

### 2. 后端用户主键

```go
type PrimaryKey = uint64   // 定义在 models/user.go
```
所有用户 ID 用这个。controller 里拿当前用户：

```go
val, _ := c.Get(middlewares.CtxKeyUserID)
userID := val.(models.PrimaryKey)
```

### 3. 新建后端表

在 `back/main.go` 的 `AutoMigrate(...)` 列表里**显式注册**新 model：

```go
if err := db.AutoMigrate(
    &models.User{},
    ...
    &models.YourNewModel{},   // ← 加这里
); err != nil { ... }
```
不注册的话表不会自动建。

### 4. 前端 API 调用

```ts
import { api } from '@/lib/api';

const data = await api.get<Foo>('/foo');           // 自动注入 Bearer token
const data = await api.post<Bar>('/bar', payload);
const data = await api.delete<Baz>('/baz');
```
`api.*` 自动解包 `{code, message, data}` → 返回 `data`。`code >= 400` 时抛异常。

### 5. 前端登录态

```tsx
import { useAuth } from '@/lib/auth';

const { user, isLoggedIn, login, logout } = useAuth();
// user: { userId: string, displayName: string, roles: string[] } | null
```

### 6. 前端共享类型

所有跟后端对接的 TS 接口都定义在 `aira-web-4/packages/shared/src/types/index.ts`，import 路径是 `@aira/shared`。新接口需要前后端同步加。

### 7. Tailwind 主题

- `brand-*` (50–900) → 项目主蓝色系
- 语义色：`green` 成功 / `red` 错误 / `amber` 警告 / `purple` AI 相关
- 详见 `aira-web-4/apps/web/tailwind.config.ts`

### 8. Markdown / 数学公式渲染

用 `import { MarkdownBlock } from '@/components/Markdown'`。
**KaTeX 仅支持** `$...$`（行内）和 `$$...$$`（块级）。
**不支持** `\(...\)` 和 `\[...\]` —— 这些会被 markdown 当字符转义吃掉。

### 9. Recall（回忆卷） vs Ingest（上传清洗） —— 不是重复造轮子

这两个模块都是"用户提交 → admin 审核 → 入 `problems` 表"，**但输入路径完全不同**，请勿合并：

| 维度 | Recall（`recall_*` 表） | Ingest（`ingest_jobs` 表） |
|---|---|---|
| 用户场景 | 考完试当场凭印象敲题 | 手头有 PDF / DOCX / 图片，整份上传 |
| 输入形态 | 网页表单逐题打字 | 文件上传，LLM 自动结构化 |
| 协作模式 | 多人对同一题 `+1 我也记得`，按 support_count 拣选 | 单人单文件，admin 直接审 LLM 输出 |
| 入口 | `/courses/[id]/recall` | `/upload` |
| 依赖 LLM | 否 | 强依赖（DeepSeek 清洗 + 千问 VL OCR） |
| 入正式题库的 admin 动作 | `/admin/reviews` → "convert recall paper" → `paper_service.ConvertRecallPaper` | `/admin/ingest` → "发布" → `ingest_service.PublishJob` |

加新功能前先判断属于哪条路径，别在错的那张表上扩字段。

### 10. Ingest 模块开发者要点

> 详细的设计、流程、踩坑、未来工作见 [`docs/DEVELOPMENT_LOG_ingest.md`](docs/DEVELOPMENT_LOG_ingest.md)。
> 下面仅列**改这块代码时必须知道**的最小集。

**文件分工：**
- `services/ingest_extract.go` — 文件 → 文本派发（PDF/DOCX/MD/IMG）
- `services/vision_client.go` — Qwen-VL OCR（独立 key 体系，与 DeepSeek 不共用）
- `services/ingest_pipeline.go` — LLM 清洗 prompt + JSON 强制 + 自检规则 A/B/C/D
- `services/dedup.go` — n-gram + Jaccard 同课粗筛（零外部依赖、零 LLM 调用）
- `services/ingest_service.go` — 状态机 / CreateJob / runPipeline / PublishJob / 合并语义
- `routers/ingest_controller.go` — 7 个端点（用户 + admin 分组）
- `cmd/worker/main.go` — 每 5s 轮询 `ProcessNextPending`

**三个不变量必须维护：**

1. **`SourceID` 用循环序号 `i+1`，不用 `sequence_id`**：唯一约束 `(testpaper_id, source_id)` 不能撞，LLM 输出的 sequence_id 在同任务内可能重复（按题型独立编号时）。
2. **试卷自动合并的 key 是 `(course_id, year, semester, exam_type)` 完全相等**：四元组任一不同都新建独立 paper。
3. **per-type 题号是显示层概念，不写进 DB**：DB 里 `sequence_id` 原样保留 LLM 输出（保 source_id 唯一 + 保题解 sequence_id 匹配语义）；前端在渲染时按数组组内位置算"判断题第 N 题"。

**LLM 配置：**
- `LLM_API_KEY`（DeepSeek）未配 → 文本清洗整体 503
- `LLM_VISION_API_KEY`（千问 VL）未配 → 图片上传 400 `vision_disabled`，其他格式不受影响

**Worker 改动必须重启 worker 进程**（不只是后端）。dedup 逻辑和文件预处理实际跑在 worker 里，后端只服务 HTTP。

---

## 已知坑

1. **`globals.css` 的 `@import` 顺序**：Turbopack 严格遵守 CSS 标准，`@import "katex"` **必须**排在 `@tailwind` 之前。反过来会构建失败。

2. **后端 PostgreSQL 特定语法**：`Problem.OptionsJSON` 用 `gorm:"type:jsonb"`、`cmd/worker/get_answer.go` 用 `$1` 占位符——切到 SQLite 需要适配。

3. **`.next/` 缓存幽灵引用**：切分支后可能引用已删文件导致 typecheck 报奇怪的错。`rm -rf aira-web-4/apps/web/.next` 后重跑即可。

4. **CGO 依赖**：`gorm.io/driver/sqlite` 用了 `mattn/go-sqlite3`，需要 C 编译器。要纯 Go SQLite 请换 `glebarez/sqlite`。

5. **未提交即删支**：dev preview / 临时调试页面建议加入 `.gitignore` 防误提交。

---

## 协作模型

本项目通过**集成仓 (integration repo)** 协作开发，由集成仓主人统一向原作者仓库提 PR：

```
upstream (原作者，只读)
   ↑ 单次 PR
集成仓 origin (大家都是 collaborator)
   ↑ 多次 PR
每人的 dev-<username> 或 feature/<name> 分支
```

### 工作分支命名

- `dev-<username>` — 个人长期工作分支
- `feature/<name>` — 短期独立功能分支（适合 PR review）

### 标准流程

```bash
git checkout main
git pull                              # 同步集成仓最新
git checkout -b feature/xxx           # 或在 dev-<self> 上继续
# ... 写代码 ...
git add ...
git commit -m "feat: 简短描述"
git push -u origin feature/xxx
# 在集成仓 GitHub 页面发 PR: base=main, compare=feature/xxx
```

### Commit message 风格

`<type>: <简短描述>`，type 建议：
- `feat` 新功能
- `fix` bug 修复
- `chore` 配置 / 清理
- `refactor` 重构
- `docs` 文档
- `test` 测试

---

## 关键约定（请勿违反）

- ❌ **API key / secret 严禁进仓库**（`.env` 已 gitignore）
- ❌ **不要重复造已有功能** —— 动手前先 `git log --all --oneline | grep <关键词>` 看看
- ❌ **不要直接 push 到 `main`** —— 走 PR 让队友 review
- ❌ **不要 `force push` 到 `main` 或他人分支**
- ✅ **改后端表后** 在 `main.go` 加 AutoMigrate
- ✅ **改前后端契约** 在 `packages/shared/src/types/` 同步更新
- ✅ **commit 前** `npm run typecheck`（前端）/ `go build ./...`（后端）
