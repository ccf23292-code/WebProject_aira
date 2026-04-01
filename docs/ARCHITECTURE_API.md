# 前后端结构解析与接口说明

## 目录结构（核心）
```
AIRAWeb/
  aira-web-4/             # 前端（Next.js）
    apps/web/             # Web 端应用
    packages/shared/      # 共享类型与枚举
  back/                   # 后端（Go + Gin）
  docs/                   # 项目文档
  scripts/                # 本地启动脚本
```

## 前端结构（`aira-web-4/apps/web`）
- `src/app/`：App Router 页面
  - `login/`、`register/`：登录/注册
  - `courses/`：课程列表与试卷列表
  - `papers/[paperId]/`：试卷题目列表与收藏
  - `favorites/`：收藏列表
  - `recall/`：回忆卷相关页面（创建题目、评论、支持票等）
- `src/components/`：通用 UI 组件
- `src/hooks/`：自定义 hooks
- `src/lib/`：通用工具
  - `api.ts`：统一封装 fetch + Bearer Token
  - `auth.tsx`：登录/注册/登出相关调用与 token 管理
  - `mock.ts`：Mock 数据（对齐后端响应结构）

### 前端 API 约定
- 基地址：`NEXT_PUBLIC_API_URL`（默认 `http://localhost:3001/api`）
- 响应格式（后端统一）：`{ code, message, data }`
- token：存储在 `localStorage`，请求时自动注入 `Authorization: Bearer <token>`

## 后端结构（`back`）
- `main.go`：Gin 入口，注册路由与中间件
- `routers/`：控制器
  - `auth_controller.go`：注册/登录/登出
  - `paper_controller.go`：课程/试卷/题目浏览
  - `favorite_controller.go`：收藏
  - `admin_controller.go`：管理员上传/修改/删除
  - `recall_controller.go`：回忆卷与协作题库
- `services/`：业务逻辑
  - `auth_service.go`：内存用户与 token 管理
  - `paper_service.go`：课程/试卷/题目与收藏（含种子数据）
  - `recall_service.go`：回忆卷（PostgreSQL）
  - `db.go`：PostgreSQL 初始化
- `middlewares/`：鉴权与管理员权限
- `models/`：数据模型
- `utils/response.go`：统一响应结构

### 后端启动与数据库
- 端口：`3001`
- 数据库环境变量：`DATABASE_URL`
  - 使用 `back/.env` 或系统环境变量
  - `recall` 模块依赖 Postgres（自动迁移）

## API 接口说明
### 统一说明
- Base URL：`/api`
- 通用响应：
```json
{ "code": 200, "message": "success", "data": {} }
```
- 鉴权 Header：`Authorization: Bearer <accessToken>`

### 认证模块（公开）
- `POST /api/auth/register`
  - body: `{ username, email, password, confirmPassword, verificationCode, agreeToPolicy }`
- `POST /api/auth/login`
  - body: `{ username, password, otp?, rememberMe? }`
- `POST /api/auth/logout`
  - body: `{ refreshToken }`

### 浏览模块（公开）
- `GET /api/courses`
- `GET /api/courses/:course_id/papers`
- `GET /api/papers/:paper_id/problems`

### 收藏模块（需要登录）
- `GET /api/favorites?page=1&size=10`
- `POST /api/favorites`
  - body: `{ problem_id }`
- `DELETE /api/favorites/:problem_id`

### 管理员模块（需要登录 + 管理员权限）
- `POST /api/admin/papers`
  - body: `{ course_id, name }`
- `PUT /api/admin/papers/:paper_id`
  - body: `{ name }`
- `DELETE /api/admin/papers/:paper_id`
- `PUT /api/admin/problems/:problem_id`
  - body: `{ test?, answer?, options? }`

### 回忆卷模块（需要登录，Postgres）
- `GET /api/recall/courses/:course_id/papers`
- `POST /api/recall/courses/:course_id/papers`
  - body: `{ title }`
- `GET /api/recall/papers/:paper_id/question-types`
- `GET /api/recall/papers/:paper_id/questions/top?question_type=xxx`
- `GET /api/recall/papers/:paper_id/questions?question_type=xxx&sequence=1`
- `POST /api/recall/papers/:paper_id/questions`
  - body: `{ question_type, sequence, content, answer, options }`
- `PATCH /api/recall/questions/:question_id`
  - body: `{ content?, answer?, options? }`
- `POST /api/recall/questions/:question_id/support`
- `GET /api/recall/questions/:question_id/comments?page=1&size=10`
- `POST /api/recall/questions/:question_id/comments`
  - body: `{ content }`

> 详细字段定义可参考 `back/services/*.go` 中的请求/响应结构体，以及 `back/interface.json`（更完整的接口描述草案）。
