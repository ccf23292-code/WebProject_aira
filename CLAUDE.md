# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 在本仓库工作时提供“项目约束 + 约定 + 操作指南”。

## 项目概述

- 项目：AIRAWeb（刷题网站）
- 技术栈：Next.js + NestJS + PostgreSQL
- 目标：优先实现 MVP（题库、做题、提交、判题）并建立可持续扩展的结构

## MVP 范围（必须遵守）

只实现以下能力（除非明确提出新增需求）：

- 题库：题目列表/详情（题干、输入输出说明、样例、约束、难度、标签）
- 做题：在线编辑/选择语言（如后续支持多语言）
- 提交：提交记录（时间、状态、耗时/内存、错误信息摘要）
- 判题：后端异步判题（队列/worker 形式），可返回编译错误/运行错误/超时/通过等结果

不做（默认不实现）：

- 讨论区、题解、用户关注、复杂权限体系、题目导入平台化、运营后台等

## Claude 工作方式（很重要）

- 优先最小改动：只做当前需求，避免“顺手重构”和无关格式化。
- 新增依赖前先说明理由与替代方案；能用现有依赖就不要加新的。
- 改动需要可验证：每次实现功能同步补上最小可行的验证方式（至少是可运行的接口/页面 + 简单的 smoke check）。
- 代码风格：TypeScript 优先，保持一致的命名与目录结构。
- 任何会影响架构的决定（例如 ORM、判题隔离方案、队列方案）要在本文件“设计决策”中补充一条。

## 推荐目录结构（可按此落地）

建议采用 monorepo（是否使用 pnpm 由后续确定）：

- apps/web: Next.js 前端
- apps/api: NestJS 后端 API
- packages/shared: 前后端共享类型（DTO、枚举、通用校验）
- packages/config: eslint/tsconfig 等共享配置（可选）

若暂不做 monorepo，也请至少保证前后端目录独立：

- web/: Next.js
- api/: NestJS

## 开发命令（搭建后务必更新）

在实际初始化脚手架后，把这里替换成真实可运行命令（不要留空）：

- Web（Next.js）：`npm run dev:web` / `npm run build:web` / `npm run start:web`
- API（NestJS）：`npm run dev:api` / `npm run build:api` / `npm run start:api`
- DB（PostgreSQL）：`npm run docker:dev` 或手动使用 docker-compose up
- 测试：`npm run test`
- 代码质量：`npm run lint` / `npm run format` / `npm run typecheck`

## 数据与接口（MVP 最小约定）

建议至少覆盖这些核心概念（字段可迭代，但避免随意改名）：

- Problem：题目（title, statement, samples, constraints, tags, difficulty）
- Submission：提交（problemId, userId(可先匿名), language, source, status, time, memory, createdAt）
- JudgeResult：判题结果（compileOutput/runtimeError/score 等，按需要最小化）

API 返回建议统一结构（可后续细化）：

- 成功：`{ data, meta? }`
- 失败：`{ error: { code, message, details? } }`

## 判题（MVP 目标与约束）

- 判题应异步执行：API 接收提交后立即返回 submissionId；结果通过轮询接口查询。
- 资源隔离与安全策略（容器/沙箱/进程隔离）属于高风险改动：实现前先在“设计决策”写清方案与取舍。

## 设计决策（随项目演进更新）

- 2026-03-10：确定技术栈 Next.js + NestJS + PostgreSQL；MVP 为题库/做题/提交/判题。
