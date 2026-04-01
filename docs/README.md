# AIRAWeb 项目介绍

AIRAWeb 是一个面向高校课程资料与历年卷管理的 Web 项目，提供课程、试卷与题目的浏览，收藏管理，以及“回忆卷”协作补题等能力。

## 代码结构总览
- 前端：`aira-web-4/`（Next.js 14 + React 18 + Tailwind，App Router）
- 后端：`back/`（Go + Gin + Gorm + PostgreSQL）
- 共享：`aira-web-4/packages/shared/`（前端共享类型与枚举）

## 本地开发与调试
- 前端启动脚本：`scripts/dev-frontend.sh`
- 后端启动脚本：`scripts/dev-backend.sh`

### 常用端口
- 前端默认：`http://localhost:3000`
- 后端默认：`http://localhost:3001`

### 环境变量
- 前端 API 基地址：`NEXT_PUBLIC_API_URL`
  - 默认值：`http://localhost:3001/api`
- 后端数据库：`DATABASE_URL`
  - 放在 `back/.env` 或直接导出环境变量

### 本地测试账号（后端内置）
- 管理员账号：`admin`
- 密码：`Admin@123`

> 详细的前后端结构与接口说明请见：`docs/ARCHITECTURE_API.md`。
