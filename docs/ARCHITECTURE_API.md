# AIRAWeb 架构与接口说明

这份文档描述当前分支的真实代码结构，不再保留“最初原型”的旧接口说明。

## 1. 总体架构

```text
Browser
  -> Next.js Web
    -> /api requests
      -> Gin Router
        -> Service
          -> Gorm
            -> PostgreSQL
```

当前项目分成 5 条主线：
- 课程与试卷浏览
- 做题与记录
- 收藏与错题本
- 题解协作
- 用户认证与个人中心

## 2. 仓库结构

```text
AIRAWeb/
  aira-web-4/
    apps/web/
      src/app/                 # Next.js 页面
      src/components/          # UI 组件
      src/lib/                 # api/auth 等公共逻辑
      src/hooks/               # useFetch 等 hooks
    packages/shared/
      src/types/               # 前后端共享类型
  back/
    cmd/                       # 导入命令
    middlewares/               # 鉴权与权限
    models/                    # Gorm 模型
    routers/                   # HTTP 控制器
    services/                  # 业务逻辑
  data/                        # 课程与试卷导入数据
  docs/                        # 文档
  scripts/                     # 开发与导入脚本
```

## 3. 前端结构

### 页面
- `/`：首页
- `/login`：登录
- `/register`：注册
- `/courses`：课程广场，支持课程名 / 课程代码搜索
- `/courses/[courseId]`：课程详情
- `/courses/[courseId]/recall`：课程回忆卷页
- `/papers/[paperId]`：试卷做题页
- `/profile`：个人资料
- `/profile/favorites`：按课程分组的收藏
- `/profile/wrongbook`：按课程分组的错题本
- `/profile/records`：做题记录

### 关键组件
- `src/components/layout/Navbar.tsx`
  - 顶栏导航：首页 / 课程 / 个人中心 / 登录态
- `src/components/Markdown.tsx`
  - Markdown + LaTeX 渲染
- `src/components/problem/ExplanationSection.tsx`
  - 题解展示、提交、投票
- `src/components/course/CourseDescriptionPanel.tsx`
  - 课程简介展示与简介修改提案提交
- `src/components/form/PasswordInput.tsx`
  - 密码显隐输入框

### 前端基础约定
- API base：`NEXT_PUBLIC_API_URL`，默认 `http://localhost:3001/api`
- 响应格式统一为：

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

- 登录态：
  - `accessToken` / `refreshToken` 存 `localStorage`
  - 请求时自动注入 `Authorization: Bearer <token>`

## 4. 后端结构

### 路由层
- `auth_controller.go`：注册 / 登录 / 登出 / 验证码
- `paper_controller.go`：课程、试卷、题目浏览
- `favorite_controller.go`：收藏
- `answer_controller.go`：做题记录、批量交卷记录
- `wrongbook_controller.go`：错题本
- `profile_controller.go`：个人资料、头像上传
- `problem_explanation_controller.go`：题解列表、投稿、编辑、投票
- `recall_controller.go`：回忆卷协作
- `admin_controller.go`：管理员修改试卷 / 题目
  - 现在也负责课程简介提案审核

### 服务层
- `auth_service.go`
  - 用户注册、登录、验证码、token 校验
  - 现在是数据库版，不再是内存版
- `course_service.go`
  - 课程搜索、课程简介提案、教师目录、课程评论、教师评论、评分标准
- `paper_service.go`
  - 试卷列表、题目列表、题目更新
- `favorite_service.go`
  - 收藏持久化，按课程聚合收藏结果
- `answer_service.go`
  - 记录单题作答与批量交卷
- `wrongbook_service.go`
  - 错题聚合、备注、状态切换、垃圾篓清空
- `profile_service.go`
  - 用户资料读取 / 更新，合并 `users + user_profiles`
- `problem_explanation_service.go`
  - 同学解析、Top 3、投票、撤回
- `recall_service.go`
  - 回忆卷题目、支持票、评论

### 中间件
- `AuthRequired`
  - 必须登录才能访问
- `TryAuth`
  - 可选登录，用于公开接口里补充 `my_vote / my_item`
- `AdminRequired`
  - 管理员权限

## 5. 当前核心数据模型

### 认证与用户
- `users`
  - 登录身份
- `user_profiles`
  - `nickname`、`avatar_url`、`level`
- `auth_sessions`
  - `accessToken` / `refreshToken` 持久化
- `email_verifications`
  - 邮箱验证码与发送节流

### 课程与题目
- `courses`
- `course_description_submissions`
- `teachers`
- `test_papers`
- `problems`

### 学习沉淀
- `favorites`
- `answer_records`
- `wrong_questions`
- `problem_explanations`
- `problem_explanation_votes`

### 协作补题
- `recall_papers`
- `recall_questions`
- `recall_question_supports`
- `recall_question_comments`

## 6. 接口清单

以下仅列当前仓库已经接入并由路由注册的接口。

### 6.1 Auth

#### `POST /api/auth/register`

请求体：

```json
{
  "username": "alice",
  "email": "alice@zju.edu.cn",
  "password": "Alice123",
  "confirmPassword": "Alice123",
  "verificationCode": "123456",
  "agreeToPolicy": true
}
```

返回：
- `userId`
- `displayName`
- `accessToken`
- `refreshToken`
- `roles`
- `expiresIn`
- `onboardingTasks`

#### `POST /api/auth/login`

请求体：

```json
{
  "username": "alice",
  "password": "Alice123",
  "rememberMe": false
}
```

#### `POST /api/auth/logout`

请求体：

```json
{
  "refreshToken": "..."
}
```

#### `POST /api/auth/verification-code`

请求体：

```json
{
  "email": "alice@zju.edu.cn"
}
```

说明：
- 开发模式下若 `DEV_EMAIL_ECHO=1`，响应会带回验证码
- 当前仍未接入真实 SMTP

### 6.2 Browse

#### `GET /api/courses?query=关键词`
- 支持课程名 / 课程代码搜索
- 兼容旧参数 `q`

#### `GET /api/courses/:course_id`
- 返回课程详情

#### `POST /api/courses/:course_id/description-submissions`
- 需要登录

```json
{
  "content": "这门课重点在树、图和排序，建议先做近三年卷。"
}
```

#### `GET /api/courses/:course_id/description-submissions/mine`
- 需要登录
- 返回当前用户针对该课程提交过的简介修改记录

#### `GET /api/courses/:course_id/teachers`
- 返回当前课程的教师目录

#### `POST /api/courses/:course_id/teachers`
- 需要登录

```json
{
  "name": "李老师",
  "title": "2025 春夏"
}
```

#### `GET /api/courses/:course_id/comments`
- 返回课程评论列表

#### `POST /api/courses/:course_id/comments`
- 需要登录

```json
{
  "comment": "这门课建议先刷树和图。"
}
```

#### `GET /api/courses/:course_id/teachers/:teacher_id/comments`
- 返回某位教师的评论列表

#### `POST /api/courses/:course_id/teachers/:teacher_id/comments`
- 需要登录

```json
{
  "comment": "作业量中等，期末题风格很稳定。"
}
```

#### `GET /api/courses/:course_id/teachers/:teacher_id/grading-standards`
- 返回某位教师的评分标准列表

#### `POST /api/courses/:course_id/teachers/:teacher_id/grading-standards`
- 需要登录

```json
{
  "description": "平时分看作业和签到",
  "standard": "平时 40%，期末 60%",
  "standard_img": ""
}
```

说明：
- 评论和评分标准的读取结果会补齐 `user_name` / `teacher_name`
- 教师目录已经从前端本地状态切到后端持久化
- 课程广场卡片正文直接读取 `courses.description`
- 用户不能直接改公开简介，只能提交提案

#### `GET /api/courses/:course_id/papers`
- 返回该课程下的试卷

#### `GET /api/papers/:paper_id/problems`
- 返回该试卷下的题目

题目字段包含：
- `question_type`
- `score`
- `test`
- `options`
- `answer`
- `explanation`

### 6.3 Favorites

#### `GET /api/favorites?page=1&size=10`
- 返回分页结果
- 同时返回 `groups`，用于前端按课程聚合显示

#### `GET /api/favorites/ids`
- 返回当前用户收藏题目 ID 列表

#### `POST /api/favorites`

```json
{
  "problem_id": 1001
}
```

#### `DELETE /api/favorites/:problem_id`

### 6.4 Admin

#### `GET /api/admin/course-description-submissions?status=pending`
- 返回课程简介提案列表

#### `POST /api/admin/course-description-submissions/:id/review`

```json
{
  "action": "approve",
  "review_note": "表述准确，允许发布"
}
```

说明：
- `action` 仅支持 `approve` / `reject`
- `approve` 时会同步更新 `courses.description`

### 6.5 Answers

#### `POST /api/answers`
- 单题记录，主要用于刷题模式

```json
{
  "paper_id": 1,
  "problem_id": 1001,
  "selected_option": "A",
  "is_correct": false,
  "mode": "practice"
}
```

#### `POST /api/answers/batch`
- 批量记录，主要用于模拟考交卷

```json
{
  "answers": [
    {
      "paper_id": 1,
      "problem_id": 1001,
      "selected_option": "A",
      "is_correct": false,
      "mode": "exam"
    }
  ]
}
```

#### `GET /api/answers?page=1&size=10`
- 返回做题记录列表

### 6.6 Wrongbook

#### `GET /api/wrongbook?status=unmastered|mastered|trash`
- 返回按课程聚合后的错题本

#### `PATCH /api/wrongbook/:problem_id`

```json
{
  "note": "这题要复习树的旋转",
  "status": "mastered"
}
```

状态枚举：
- `unmastered`
- `mastered`
- `trash`

#### `DELETE /api/wrongbook/:problem_id`
- 删除单条错题

#### `DELETE /api/wrongbook/trash`
- 清空垃圾篓

### 6.7 Profile

#### `GET /api/profile`
- 返回：
  - `username`
  - `email`
  - `nickname`
  - `avatar_url`
  - `level`

#### `PUT /api/profile`

```json
{
  "nickname": "Ling"
}
```

#### `POST /api/profile/avatar`
- `multipart/form-data`
- 字段名：`avatar`
- 限制：不超过 5MB

### 6.8 Problem Explanations

#### `GET /api/problems/:problem_id/explanations`
- 公开读
- 若已登录，会附带：
  - `my_vote`
  - `my_item`

返回字段：
- `official_explanation`
- `items`：Top 3 同学解析
- `my_item`：若我的解析未进入 Top 3，则单独返回

#### `POST /api/problems/:problem_id/explanations`
- 登录后提交或更新我在这道题上的解析

```json
{
  "content_md": "这里写 Markdown 和公式 $O(n \\log n)$"
}
```

#### `PATCH /api/problems/:problem_id/explanations/:explanation_id`
- 只能编辑自己的解析

#### `POST /api/problems/:problem_id/explanations/:explanation_id/vote`

```json
{
  "value": 1
}
```

说明：
- `1` = 赞
- `-1` = 踩
- `0` = 撤回

### 6.9 Recall

#### `GET /api/recall/courses/:course_id/papers`
#### `POST /api/recall/courses/:course_id/papers`
#### `GET /api/recall/papers/:paper_id/question-types`
#### `GET /api/recall/papers/:paper_id/questions/top?question_type=...`
#### `GET /api/recall/papers/:paper_id/questions?question_type=...&sequence=1`
#### `POST /api/recall/papers/:paper_id/questions`
#### `PATCH /api/recall/questions/:question_id`
#### `POST /api/recall/questions/:question_id/support`
#### `GET /api/recall/questions/:question_id/comments?page=1&size=10`
#### `POST /api/recall/questions/:question_id/comments`

### 6.10 Paper Admin

#### `POST /api/admin/papers`
#### `PUT /api/admin/papers/:paper_id`
#### `DELETE /api/admin/papers/:paper_id`
#### `PUT /api/admin/problems/:problem_id`

## 7. 脚本说明

### `scripts/dev.sh`
- 一键启动前后端
- 启动前会先 kill 掉占用 3000 / 3001 的进程

### `scripts/dev-backend.sh`
- 自动检查 Go
- 自动检查 PostgreSQL
- 在 mac / linux 上尽量自动安装 / 启动 PostgreSQL

### `scripts/dev-frontend.sh`
- 检查 `node_modules`
- 启动前端开发服务器

### `scripts/import-courses.sh`
- 导入 `data/course` 下的课程

### `scripts/test-import.sh`
- 清空数据库
- 导入 FDS 课程
- 导入 `data/papers/CS1018F` 下的测试题目

## 8. 当前已知边界

- SMTP 还未正式接入，验证码发送仍是开发态方案
- 课程详情中的老师 / 评分标准 / 留言广场还未完成
- PDF 自动解析链路还没实现
- 用户等级当前只有字段，没有业务规则
