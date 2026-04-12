## 任务 1：通用种子账号命令与测试导入脚本

### 目标
- 新增通用用户种子命令
- `scripts/test-import.sh` 自动创建管理员和普通用户

### 文件
- Create: `back/cmd/seed_user/main.go`
- Modify: `scripts/test-import.sh`
- Modify: `README.md`
- Modify: `docs/README.md`

### 先写失败测试
- Add/Update: `back/services/auth_service_test.go`
- 新增测试覆盖：
  - 用户资料存在时能正常登录
  - 测试脚本依赖的默认账号结构与 profile 兼容

### 实现
- 写 `seed_user` 命令，支持 `--role`
- `test-import.sh` 改为调用两次 `seed_user`
- 保留现有 `seed_admin`，但测试脚本不再依赖它

### 验证
```bash
cd back && go test ./...
bash scripts/test-import.sh
```

## 任务 2：首页留言后端

### 目标
- 新增首页留言模型、服务、路由

### 文件
- Modify: `back/models/course.go` or create `back/models/homepage.go`
- Create: `back/services/homepage_service.go`
- Create: `back/routers/homepage_controller.go`
- Modify: `back/main.go`
- Add: `back/services/homepage_service_test.go`

### 先写失败测试
- `TestListHomepageMessagesHydratesProfileFields`
- `TestAddHomepageMessageRejectsBlankContent`
- `TestAddHomepageMessagePersistsMessage`

### 实现
- 新增 `HomepageMessage`
- 新增服务：
  - `ListMessages()`
  - `AddMessage(userID, content)`
- 新增控制器：
  - `GET /api/homepage/messages`
  - `POST /api/homepage/messages`

### 验证
```bash
cd back && go test ./...
```

## 任务 3：首页留言前端

### 目标
- 首页新增留言广场 UI 和 API 接入

### 文件
- Create: `aira-web-4/apps/web/src/lib/homepage.ts`
- Modify: `aira-web-4/apps/web/src/app/page.tsx`
- Modify: `aira-web-4/packages/shared/src/types/index.ts`

### 先写失败检查
- 先跑类型检查，确认新增类型前会失败于缺失引用

### 实现
- 封装 `getHomepageMessages` / `addHomepageMessage`
- 首页加入输入区和留言列表
- 未登录只读，已登录可提交

### 验证
```bash
cd aira-web-4/apps/web && npm run typecheck
```

## 任务 4：文档、提交、推送

### 文件
- Modify: `README.md`
- Modify: `docs/ARCHITECTURE_API.md`
- Modify: `docs/TODO.md`

### 验证
```bash
git diff --check
cd back && go test ./...
cd aira-web-4/apps/web && npm run typecheck
git status --short --branch
git push origin dzz
```
