## 首页留言广场与测试账号种子设计

### 目标
- 在首页新增“留言广场”，用于收集用户对项目的改进建议。
- 留言直接发布，不走审核。
- `scripts/test-import.sh` 在测试导入后自动创建两个可用账号：
  - 管理员：`admin / admin@123`
  - 普通用户：`student / student@123`

### 约束
- 保持现有认证与用户资料体系，不恢复任何启动时 bootstrap 逻辑。
- 首页留言与课程评论分开建模，不复用课程评论表。
- 留言先支持“发布 + 列表展示”，暂不做编辑、删除、分页、点赞。

### 方案

#### 1. 测试账号种子
- 新增通用命令 `back/cmd/seed_user/main.go`
- 支持参数：
  - `--username`
  - `--password`
  - `--email`
  - `--nickname`
  - `--role`（`admin` / `student`）
- 行为：
  - 按用户名幂等 upsert 用户
  - 同步写入/更新 `users` 和 `user_profiles`
- `scripts/test-import.sh` 调用两次：
  - 一次创建管理员
  - 一次创建普通用户

#### 2. 首页留言数据模型
- 新增表 `homepage_messages`
  - `id`
  - `user_id`
  - `content`
  - `created_at`
  - `updated_at`
- 读取时回填用户显示信息：
  - `user_name`
  - `avatar_url`

#### 3. 后端接口
- `GET /api/homepage/messages`
  - 公开读取
  - 返回按时间倒序的留言列表
- `POST /api/homepage/messages`
  - 需要登录
  - 创建留言

#### 4. 前端页面
- 首页新增“留言广场”板块
- 包含：
  - 登录用户可见的留言输入框
  - 留言发布按钮
  - 最新留言列表
- 未登录时只展示列表，并提示登录后可留言

### 不做的内容
- 留言编辑 / 删除
- 审核
- 敏感词过滤
- 分页 / 无限滚动

### 验证标准
- `scripts/test-import.sh` 执行后可用 `admin` 和 `student` 登录
- 首页可以加载留言列表
- `student` 登录后可以在首页发布留言并立即看到
- 后端测试和前端类型检查通过
