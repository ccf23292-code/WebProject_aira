# TODO / 进度清单

## 已实现（可用）
- 前后端本地启动脚本（`scripts/dev-frontend.sh`、`scripts/dev-backend.sh`、`scripts/dev.sh`）
- 后端本地 Postgres 自动安装/启动与初始化（开发环境）
- 注册页面补齐必填字段（email / verificationCode / agreeToPolicy）
- 新增“获取验证码”接口与前端按钮（开发模式回显验证码）
- 统一文档：项目介绍与接口说明（`docs/README.md`、`docs/ARCHITECTURE_API.md`）

## 进行中（开发模式）
- 邮箱验证码：开发模式回显验证码（`DEV_EMAIL_ECHO=1`）
  - 仅用于本地调试，不适合生产环境

## 待办（计划）
- SMTP / 邮件服务商正式接入（发送真实验证码）
- 邮箱域名白名单（仅允许 `@zju.edu.cn` / 二级域）
- 验证码频控与风控（IP/邮箱维度限流、错误次数限制）
- 忘记密码 / 重置密码流程
- 前端错误提示优化（表单级别提示 + 后端错误映射）
- 管理员后台完善（题库导入、试卷管理批量操作）

## 部署相关（后续）
- 生产环境配置（ENV、日志、监控）
- 数据库迁移与备份策略
