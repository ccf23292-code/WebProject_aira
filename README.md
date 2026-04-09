# AIRAWeb

AIRAWeb 是一个面向课程题库、历年卷练习、回忆卷协作和个人学习沉淀的 Web 项目。当前仓库已经具备一条可跑通的本地开发链路：课程导入、试卷导入、注册登录、刷题 / 模拟考、题解协作、收藏、错题本、做题记录、个人资料。

## 当前进展

### 已完成
- 课程、试卷、题目已从内存方案切到 PostgreSQL 持久化
- 支持课程导入与测试题库导入，当前默认测试课程为 FDS / `CS1018F`
- 注册 / 登录 / 登出、邮箱验证码（开发模式回显）已可用
- 用户、验证码、accessToken、refreshToken 已持久化到数据库，后端重启后账号不会丢失
- 题目页支持两种模式
  - 刷题模式：边做边看答案 / 解析
  - 模拟考模式：倒计时、自定义时长、自动交卷或超时继续
- 题目解析支持 Markdown 和 LaTeX
- 同学解析支持提交、作者编辑、点赞 / 点踩 / 撤回，按评分展示 Top 3
- 个人中心已接入收藏、错题本、做题记录、昵称、头像、用户等级
- 课程社区读写链路已闭合：教师目录、课程评论、教师评论、评分标准均已接后端
- 课程广场卡片改为显示真实课程简介，用户可提交简介修改，管理员审核后生效

### 还没做完
- 课程详情里的“学期信息 / 留言广场审核流”还没有完整编辑链路
- SMTP 真发信还没接，当前验证码仍以开发调试为主
- PDF -> Markdown -> JSON 的自动处理链路还没有开始做

## 仓库结构

```text
AIRAWeb/
  README.md
  aira-web-4/                 # 前端 monorepo
    apps/web/                 # Next.js Web 应用
    packages/shared/          # 前后端共享 TS 类型
  back/                       # Go + Gin + Gorm + PostgreSQL 后端
  data/                       # 课程与题目导入数据
  docs/                       # 项目说明、架构、接口、TODO
  scripts/                    # 本地开发 / 导入脚本
```

## 技术栈

- 前端：Next.js 14、React 18、TypeScript、Tailwind CSS
- 后端：Go 1.22、Gin、Gorm
- 数据库：PostgreSQL
- 文档渲染：react-markdown、remark-math、rehype-katex

## 本地开发

### 1. 一键启动前后端

```bash
bash scripts/dev.sh
```

默认端口：
- 前端：`http://localhost:3000`
- 后端：`http://localhost:3001`

### 2. 只启动单边

```bash
bash scripts/dev-backend.sh
bash scripts/dev-frontend.sh
```

### 3. 导入测试数据

导入 FDS 课程与相关试卷 / 题目：

```bash
bash scripts/test-import.sh
```

这个脚本会：
- 清空当前数据库
- 导入“数据结构基础”课程
- 导入 `data/papers/CS1018F` 下的试卷和题目

如果只想导入课程：

```bash
bash scripts/import-courses.sh
```

## 典型验证路径

1. 执行 `bash scripts/test-import.sh`
2. 执行 `bash scripts/dev.sh`
3. 打开 `http://localhost:3000/register`
4. 注册新账号，使用开发模式验证码完成注册
5. 进入首页和课程广场，搜索 `CS1018F`
6. 进入课程详情页，测试：
   - 课程简介显示
   - 提交课程简介修改
   - 教师目录读取 / 新增
   - 课程评论
   - 教师评论
   - 评分标准
7. 进入试卷页，分别测试：
   - 刷题模式
   - 模拟考模式
   - 收藏
   - 题解提交与投票
8. 进入个人中心，检查：
   - 收藏
   - 错题本
   - 做题记录
   - 昵称 / 头像 / 等级
9. 重启后端，再次登录，确认账号仍在

## 关键页面

- `/`：首页
- `/courses`：课程广场 / 搜索
- `/courses/[courseId]`：课程详情
- `/papers/[paperId]`：试卷做题页
- `/register`：注册
- `/login`：登录
- `/profile`：个人中心
- `/profile/favorites`：收藏
- `/profile/wrongbook`：错题本
- `/profile/records`：做题记录

## 文档索引

- 项目进展与开发入口：`docs/README.md`
- 架构与接口：`docs/ARCHITECTURE_API.md`
- 后续路线：`docs/TODO.md`
- 邮件服务配置：`docs/EMAIL_SERVICE_SETUP.md`

## 接下来建议优先做什么

1. 课程详情扩充：学期信息、留言广场审核流
2. SMTP 正式接入：验证码改为真实邮件发送
3. 题库数据链路：PDF -> Markdown -> JSON -> 导入
4. 用户体系扩展：等级规则、勋章、统计面板
