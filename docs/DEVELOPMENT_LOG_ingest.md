# 开发日志 · Ingest 模块（上传题库 / 题解清洗）

> 本次工作在 `dev-iwijs` 分支完成，时间：2026-05-29
> 模块覆盖：后端 7 个新文件、4 个 model/路由扩展；前端 6 个新页面/组件；课程目录全量导入（2307 门 ZJU 真实课程）

---

## 1. 背景与目标

原项目让用户题目入库走两条路：
- **手工管理** ([`/admin/papers`](../back/routers/admin_controller.go))：admin 在后台逐题敲
- **回忆卷** (recall)：学生考完凭印象逐题打字，多人接力 +1 投票

但仍缺一条**最常见的现实路径** —— 用户手里有 PDF/扫描件/Word 真题，希望"一键上传，AI 帮我拆题、入库"。

本次新增的 **Ingest 模块** 填上这条路径：

```
用户上传文件 → AI 自动清洗成结构化 JSON → 管理员审核 → 正式入题库
```

并在过程中：
- 试卷采用结构化命名（年份 + 学期 + 考试类型），同标识自动合并避免分裂
- LLM 清洗后即时跑题目级查重，给上传者和管理员预警

最终落到的存储是与原系统 100% 兼容的 `test_papers` + `problems` 表 —— 学生做题侧的所有功能（收藏、错题、做题流水、AI 题解）自动复用。

---

## 2. 端到端流程

```
┌───────────────────────────────────────────────────────────────────┐
│  用户                          worker / 后端                        │
│                                                                   │
│  POST /api/ingest/upload                                          │
│    multipart: kind, course_id?, new_course_name?, year/sem/exam, │
│              target_paper_id?, file                              │
│         │                                                         │
│         ▼                                                         │
│  IngestJob 入库 (status=pending)                                  │
│         │                                                         │
│         ▼  ──── worker 每 5s 抢占一条 ────                         │
│  status=processing                                                │
│         │                                                         │
│         ▼  ExtractTextFromFile                                    │
│  PDF/DOCX/MD 走 Go 解析；JPG/PNG 走 Qwen-VL API                   │
│         │                                                         │
│         ▼  CleanQuestionText / CleanExplanationText               │
│  DeepSeek json_object 模式 → {"items":[...]}                      │
│         │                                                         │
│         ▼  仅题目流程：FindSimilarProblems                         │
│  3-gram + Jaccard ≥ 0.70 → dedup_warnings                         │
│         │                                                         │
│         ▼                                                         │
│  status=awaiting_review                                           │
│                                                                   │
│  用户看 /upload/jobs/:id     admin 看 /admin/ingest/:id            │
│  渲染预览 + 查重提示          编辑 JSON + 三段命名 + 查重对比       │
│                                                                   │
│         ▼ admin 点"发布到题库"                                     │
│  publishQuestions / publishExplanations                          │
│  ┌───────────────────────────────────────────────────────────┐    │
│  │ 题目：(course_id, year, sem, exam) 命中 → 合并；否则新建  │    │
│  │ 题解：按 sequence_id 映射到目标卷的 problems，写 explain  │    │
│  └───────────────────────────────────────────────────────────┘    │
│         │                                                         │
│         ▼                                                         │
│  status=published                                                 │
│                                                                   │
│  学生在 /courses/:id/papers/:id 做题，看不出来历差异              │
└───────────────────────────────────────────────────────────────────┘
```

---

## 3. 数据模型

### 3.1 新表 `ingest_jobs`

[`back/models/ingest_job.go`](../back/models/ingest_job.go)

| 字段 | 类型 | 含义 |
|---|---|---|
| `id` | uint64 PK | 自增 |
| `user_id` | uint64 | 上传者 |
| `kind` | varchar(16) | `question` / `explanation` |
| `course_id` | varchar(64) | 已有课程 ID（与 new_course_name 二选一） |
| `new_course_name` | varchar(128) | 用户选"新增课程"时填，admin 审核时决定是否落 `courses` |
| `paper_name` | varchar(256) | 兼容老前端的自由文本试卷名 |
| `year` / `semester` / `exam_type` | int / varchar(16/32) | 结构化三段，自动合并 key |
| `target_paper_id` | uint64* | 题解流程必填 |
| `filename` / `storage_path` / `mime` / `size` | — | 文件元信息 |
| `status` | varchar(32) | 状态机（见下） |
| `error_message` | text | 失败原因 |
| `raw_text` | text | 预处理提取的 Markdown |
| `parsed_json` | jsonb | LLM 结构化结果（admin 可编辑） |
| `dedup_warnings` | jsonb | 查重命中（仅题目流程） |
| `llm_model` | varchar(64) | 实际使用的模型名 |
| `reviewer_id` / `reviewed_at` / `published_at` | — | 审核轨迹 |
| `created_at` / `updated_at` | — | 时间戳 |

#### 状态机

```
pending          ── worker 抢占 ──▶ processing
processing       ── 清洗成功 ──▶ awaiting_review
processing       ── 清洗失败 ──▶ failed
awaiting_review  ── admin 发布 ──▶ published
awaiting_review  ── admin 拒绝 ──▶ rejected
```

### 3.2 `test_papers` 扩展

[`back/models/testpaper.go`](../back/models/testpaper.go)

新加：
- `year int gorm:"index"`
- `semester varchar(16) gorm:"index"`
- `exam_type varchar(32) gorm:"index"`

**未加 DB 级唯一约束** —— 历史 import_papers 导入的数据三段为零值，加约束会冲突。合并逻辑放在 service 层用 `WHERE` 软匹配。

允许取值（前后端共享）：
- `PaperSemesters = ["春夏","秋冬","暑期","全年"]`
- `PaperExamTypes = ["期中","期末","小测","模考","自测","其他"]`

---

## 4. 后端架构

### 4.1 文件清单

| 文件 | 职责 |
|---|---|
| [`services/ingest_extract.go`](../back/services/ingest_extract.go) | 文件 → 文本派发器（mime/扩展名分流） |
| [`services/vision_client.go`](../back/services/vision_client.go) | Qwen-VL OCR 客户端（OpenAI 兼容协议） |
| [`services/ingest_pipeline.go`](../back/services/ingest_pipeline.go) | LLM 清洗（强制 JSON + 自检 prompt） |
| [`services/dedup.go`](../back/services/dedup.go) | n-gram + Jaccard 查重 |
| [`services/ingest_service.go`](../back/services/ingest_service.go) | 业务编排 + 状态机 + 发布入库 |
| [`routers/ingest_controller.go`](../back/routers/ingest_controller.go) | HTTP 端点（用户 + admin 两组） |
| [`models/ingest_job.go`](../back/models/ingest_job.go) | IngestJob 表 |

### 4.2 文件预处理（`ExtractTextFromFile`）

| 类型 | 实现 | 依赖 |
|---|---|---|
| `.md` / `.markdown` / `.txt` | `os.ReadFile` 直读 | stdlib |
| `.pdf` | `ledongthuc/pdf` 抽文本 | 新增依赖 |
| `.docx` | `archive/zip` + `encoding/xml` 自解 | stdlib（零新依赖） |
| `.jpg` / `.jpeg` / `.png` | Qwen-VL `chat.completions` data URL | 千问 VL API |

PDF 仅适用于数字版；扫描版图片 PDF 提取会失败并提示"请转 DOCX 或图片"。
DOCX 解析直接读 `word/document.xml`，遍历 `<w:t>` 文本节点 + `<w:p>` 段落分隔，无需外部依赖。

### 4.3 LLM 清洗 prompt 设计

[`ingest_pipeline.go::questionCleanSystemPrompt`](../back/services/ingest_pipeline.go)

**强制 JSON 模式**：用 DeepSeek 的 `response_format: json_object` 让模型必产合法 JSON。提示词中显式包含 "JSON" 字样以满足该模式的要求。

**自检规则 A/B/C/D**（题目流程）：
- **A 排版修复**：公式补 `$...$`、选项内容补全、统一中英文标点
- **B OCR 字符纠正**：0/O、1/l/I、5/S、○/O、全/半角标点按上下文修复
- **C 答案-题干自洽性**：单选答案必须在 options 字母里；判断题映射 True/False；矛盾留空不硬塞
- **D 严禁脑补**：缺漏字段留空，不补全官方解析，不猜难度/标签

题解流程的 prompt 简化（只要 sequence_id + content_md），同样有 A/B/D 规则。

### 4.4 查重算法（[`dedup.go`](../back/services/dedup.go)）

零外部依赖、零 LLM 调用、同步执行：

1. **归一化** `normalizeForDedup`：
   - 删空白
   - 删 LaTeX 包围符 (`$ \ { } ^ ...`)、Markdown 装饰 (`* _ # ...`)
   - 删 ASCII / 中文标点（含中文引号 U+201C/D、U+2018/9）
   - 全角字母数字 → 半角
   - 英文转小写

2. **n-gram 切分**（rune 级 3-gram，对中文友好）

3. **Jaccard 相似度** `|A ∩ B| / |A ∪ B|`

4. **同课粗筛 + 阈值过滤**：阈值 0.70，每道新题最多保留 top-3 匹配

性能：1000 道题 * 30 道新题 ≈ 30k 次集合比较，本地 PG 测试亚秒完成。后续若题量上万再考虑 MinHash + LSH 或上 embedding + pgvector。

### 4.5 试卷自动合并

[`ingest_service.go::publishQuestions`](../back/services/ingest_service.go)

```go
if year > 0 && semester != "" && exam_type != "" {
    tx.Where("course_id = ? AND year = ? AND semester = ? AND exam_type = ?", ...).First(&existing)
    // 命中 → 复用 existing.ID，order 接续；否则新建
}
```

合并时 `Order` 字段从现有最大值 +1 续起，保证题目顺序不乱。
`SourceID` 模板 `ingest:<job_id>:<filename>:<i+1>`（用循环序号而非 sequence_id，避免 LLM 输出同 seq 时撞 `(testpaper_id, source_id)` 唯一约束）。

### 4.6 worker 集成

[`cmd/worker/main.go`](../back/cmd/worker/main.go) 改造：保留老的 `ProcessQuestion`（向后兼容），新增 GORM 初始化 + 每 5 秒一轮 `ingest.ProcessNextPending` 直到 pending 队列吃完。

### 4.7 LLM 配置（环境变量）

[`.env.example`](../back/.env.example)：

```bash
# 文本清洗（DeepSeek 兼容 OpenAI 协议）
LLM_API_KEY=
LLM_BASE_URL=https://api.deepseek.com/v1
LLM_MODEL=deepseek-chat
LLM_TIMEOUT_SECONDS=60

# 图片 OCR（默认走阿里通义千问 VL）
LLM_VISION_API_KEY=
LLM_VISION_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1
LLM_VISION_MODEL=qwen-vl-max
LLM_VISION_TIMEOUT_SECONDS=90
```

两个 key 都未配置时：
- 文本类上传清洗返回 `503 llm_disabled`
- 图片上传返回 `400 vision_disabled`，**其他格式不受影响**

---

## 5. HTTP API

### 用户端（需登录）
- `POST /api/ingest/upload` — multipart 上传，创建 job
- `GET  /api/ingest/my` — 我的上传记录（分页 + 状态过滤）
- `GET  /api/ingest/:id` — 单条详情（本人或 admin 可见）

### 管理员端
- `GET   /api/admin/ingest` — 审核队列
- `GET   /api/admin/ingest/:id` — 详情
- `PATCH /api/admin/ingest/:id` — 编辑（含 parsed_json / 三段命名）
- `POST  /api/admin/ingest/:id/publish` — 发布入题库
- `POST  /api/admin/ingest/:id/reject` — 拒绝（带原因）

---

## 6. 前端

### 6.1 新增页面

| 路由 | 文件 | 用途 |
|---|---|---|
| `/upload` | [`app/upload/page.tsx`](../aira-web-4/apps/web/src/app/upload/page.tsx) | hub：对照表 + 上传文件 / 回忆卷 两张并列卡 |
| `/upload/file` | [`app/upload/file/page.tsx`](../aira-web-4/apps/web/src/app/upload/file/page.tsx) | 上传表单（支持 `?courseId=xxx` 预填）|
| `/upload/jobs` | [`app/upload/jobs/page.tsx`](../aira-web-4/apps/web/src/app/upload/jobs/page.tsx) | 我的上传记录（轮询）|
| `/upload/jobs/[id]` | [`app/upload/jobs/[id]/page.tsx`](../aira-web-4/apps/web/src/app/upload/jobs/[id]/page.tsx) | 详情：渲染 markdown 卡片 + 查重提示（不暴露 raw JSON）|
| `/admin/ingest` | [`app/admin/ingest/page.tsx`](../aira-web-4/apps/web/src/app/admin/ingest/page.tsx) | 审核队列 |
| `/admin/ingest/[id]` | [`app/admin/ingest/[id]/page.tsx`](../aira-web-4/apps/web/src/app/admin/ingest/[id]/page.tsx) | 审核详情：三段编辑 + JSON 编辑器 + 查重 side-by-side |

### 6.2 新组件

[`components/CourseCombobox.tsx`](../aira-web-4/apps/web/src/components/CourseCombobox.tsx) — 带搜索的课程选择器，替换 3 处原生 `<select>`：
- name / code / id 模糊匹配
- 最多渲染 50 项（性能保护，2307 课无压力）
- 可选"+ 新增课程"特殊项（`allowNew` prop）
- 主题色 `brand` / `amber` 切换

### 6.3 关键交互细节

- **用户上传详情页**：parsed_json 不直接展示，而是按 item 渲染成漂亮的题目卡片（含 LaTeX）。每题型独立编号（"判断题第 1 题" 而非全局序号）。
- **dedup 警告**：用户视图简洁列表；admin 视图带 side-by-side snippet 对比。
- **课程详情页右栏**：原本只有"回忆卷"孤零零，改成两张并列卡 📤 上传文件 + ✍️ 回忆卷，预填 courseId 跳转。
- **顶栏菜单**：登录后多 "上传题库"；admin 多 "上传审核"。

---

## 7. 课程目录对齐

之前 `courses` 表只有少量手工创建的演示课程，新增功能后用户能选的课程极少。

执行：
```bash
go run ./cmd/import_courses --path data/course
```

把 `data/course/zju_courses.json` 里 **2307 门 ZJU 真实课程** 一次性导入。每条 `id = kcdm`（如 `CS1018F`、`MED0621GZ`），保证课程 ID 为纯 ASCII，路由稳定。

`ensureCourseByName` 仍保留作为新课程兜底，ID 改用 `course_<UTC timestamp>` 形式（早期版本曾用中文 slug，引发 URL 编码问题）。

---

## 8. 踩过的坑

1. **Raw string + 反引号**：prompt 文案里有"```"被 Go raw-string 截断 → 改为描述性文字。
2. **课程 ID 含中文** → URL `encodeURIComponent` → `useParams` 拿到的还是编码版 → "course not found"。改为纯 ASCII 时间戳生成。
3. **PowerShell 与 PostgreSQL 编码错位**：从 PowerShell 直接 `psql -c "WHERE name = '中文'"` 会被传成 GBK → 报 UTF8 解码错。改用 `psql -f file.sql` 配合文件 BOM。
4. **`go mod tidy` 卡死**：default GOPROXY 在国内不稳。`go env -w GOPROXY=https://goproxy.cn,direct` 解决。
5. **Chocolatey 安装 PostgreSQL 卡 initdb**：现象是进程在但 CPU 时间不涨。耐心等可以，或杀进程后手动 `pg_ctl register` + `Start-Service` 完成收尾。
6. **唯一约束 `idx_source_paper` 被撞**：我曾把 `sequence_id` 按 question_type 重新编号 → 同任务内 4 个题型都拿到 seq=1 → source_id 模板 `ingest:<job>:<file>:<seq>` 4 个完全一样。修复：source_id 改用循环序号 `i+1`，per-type 编号下沉到前端显示层（保留 sequence_id 与题解上传的匹配语义）。
7. **GORM `Table().Joins().Scan()` 在某些场景不返回结果**：dedup 查询改写为 `db.Raw(...)` 显式 SQL，规避兼容性问题。

---

## 9. 已知限制 / 未来工作

- **Embedding 级语义查重**：当前 n-gram 算法对"换数字保结构"重题敏感，但对"改写、换说法"重题查不到。下一步可考虑接 embedding API + pgvector。
- **题解流程的 sequence_id 歧义**：sequence_id 由 LLM 按源文档编号生成，可能全局或按类型。`publishExplanations` 用 `bySeq[seq]` 映射，若同卷不同类型出现相同 seq 会撞键。短期可在 prompt 里强约束"按全局题号"，长期改为 `(question_type, sequence_id)` 复合 key。
- **试卷查重提示**：现在自动合并是"硬合"，admin 没法选择"我想新建一份"。可加 UI 开关。
- **失败任务的重试**：状态机里 `failed` 终态不可达 `pending`。可加"重新清洗"按钮（保留文件，重写 status）。
- **上传配额 / 速率限制**：worker 单进程串行处理，热点时段可能堆积。可加 token bucket 或多实例 worker。
- **图片上传必须 admin 配 vision key**：用户上传 jpg/png 而未配 `LLM_VISION_API_KEY` 时拒绝。可在 UI 上禁用图片选项作 UX 改进。

---

## 10. 测试备忘

### 端到端手测路径

1. 登录 admin（`qiuyinxi / admin123`）
2. http://localhost:3000/upload → 选 📤 上传文件
3. 课程搜 "数据结构" → 选 ZJU 真实课程（如 `CS1018F`）
4. 年份 2024 / 学期 秋冬 / 考试 期末
5. 上传 `test_upload_sample.md` 或任意 markdown 题集
6. 跳到 `/upload/jobs/{id}` 等 30s 内 awaiting_review
7. 切到 `/admin/ingest/{id}` 审核 → 发布
8. 去 `/courses/CS1018F/papers/<paper_id>` 做题 ✅
9. **重传同样三段** 同样文件 → 应该看到黄色查重警告，发布后题目并入同一份卷子

### 日志期望（worker 第二次上传时）
```
ingest: job N kind=question course_id="CS1018F" items=4
dedup: course=CS1018F existing_problems=4 new_questions=4
dedup: course=CS1018F matched=4
ingest: job N wrote 4 dedup_warnings
```

---

## 11. 配套文档

- 用户启动指南：[`RUN.md`](../RUN.md)（桌面）
- 协作约定：[`CLAUDE.md`](../CLAUDE.md)
- 已加章节 "Recall（回忆卷） vs Ingest（上传清洗） —— 不是重复造轮子"
