# AIRAWeb - 刷题网站

一个基于 Next.js + NestJS + PostgreSQL 的在线刷题平台。(初期)

## 技术栈

- **前端**: Next.js 16 + TypeScript + Tailwind CSS
- **后端**: NestJS + TypeORM + PostgreSQL
- **部署**: Docker + Docker Compose

## 项目结构

```
AIRAWeb/
├── api/                # NestJS 后端 API
├── web/               # Next.js 前端应用
├── packages/
│   ├── shared/        # 共享类型定义
│   └── config/        # 共享配置
├── docker-compose.yml # Docker 容器编排
└── CLAUDE.md          # Claude Code 工作约束
```

## 快速开始

### 环境要求

- Node.js 18+
- npm 或 yarn
- Docker & Docker Compose (推荐)

### 方式一：使用 Docker Compose（推荐）

```bash
# 克隆项目
git clone <repository-url>
cd AIRAWeb

# 启动所有服务
npm run docker:dev

# 或直接使用 docker-compose
docker-compose up
```

### 方式二：本地开发

```bash
# 安装根目录依赖
npm install

# 安装各子项目依赖
npm run install:all

# 启动后端 API（端口 3001）
npm run dev:api

# 启动前端应用（端口 3000）
npm run dev:web

# 同时启动前后端
npm run dev
```

## 环境变量

复制 `.env.example` 为 `.env` 并配置相应值：

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=aira_web
NODE_ENV=development
```

## 开发命令

```bash
# 同时启动前后端开发环境
npm run dev

# 单独启动前端
npm run dev:web

# 单独启动后端
npm run dev:api

# 构建项目
npm run build

# 启动生产环境
npm run start

# 运行测试
npm run test

# 代码检查
npm run lint

# 代码格式化
npm run format

# TypeScript 类型检查
npm run typecheck
```

## API 接口

### 题目管理

- `GET /problems` - 获取题目列表
- `GET /problems/:id` - 获取题目详情
- `POST /problems` - 创建题目

### 提交记录

- `GET /submissions` - 获取提交记录
- `POST /submissions` - 提交代码

## 贡献指南

1. Fork 项目
2. 创建功能分支
3. 提交变更
4. 推送到分支
5. 创建 Pull Request

## 许可证

MIT License
