# AIRA Web

基于 `interface.json`，本次补全了课程模块相关的前端能力，重点围绕课程搜索、课程详情、课程评论、教师评论和评分标准几个场景进行了增强。

## 本次完善内容

### 1. 课程搜索对齐接口定义

- 将课程搜索参数从原先页面里使用的 `q` 统一调整为 `query`，与 `interface.json` 中的 `GET /courses` 保持一致。
- 修复首页搜索跳转后课程页不读取 URL 参数的问题，现在可以从首页直接带着搜索词进入课程广场。
- 课程卡片补充了更清晰的能力提示，用户可以直接感知课程页已经支持试卷、课程评论和教师评分标准。

### 2. 课程详情页扩展为课程工作区

课程详情页不再只展示试卷列表，现已整合以下能力：

- `GET /courses/{course_id}`：课程基础信息展示。
- `GET /courses/{course_id}/papers`：试卷列表保留原有练习入口。
- `GET /courses/{course_id}/comments`：课程评论列表。
- `POST /courses/{course_id}/comments`：发布课程评论。
- `GET /courses/{course_id}/teachers/{teacher_id}/comments`：按教师查看评论。
- `POST /courses/{course_id}/teachers/{teacher_id}/comments`：发布教师评论。
- `GET /courses/{course_id}/teachers/{teacher_id}/grading-standards`：查看评分标准。
- `POST /courses/{course_id}/teachers/{teacher_id}/grading-standards`：补充评分标准。

### 3. 新增课程社区能力

为了解决 `interface.json` 中没有提供“教师列表接口”这一现实限制，前端补了一个本地教师目录：

- 支持在教师之间切换，查看对应的评论和评分标准。
- 支持在没有后端数据时使用本地兜底数据和浏览器本地存储，保证页面在联调前也可完整演示流程。

### 4. 工程可用性修复

- 为 `packages/shared` 补充了 `typecheck` 脚本。
- 将根工作区的 React 类型版本调整为 React 18 对齐版本，修复 `Link` 等组件的全局类型冲突。
- 已执行并通过 `npm run typecheck`。

## 主要新增/修改文件

- `apps/web/src/app/courses/page.tsx`
- `apps/web/src/app/courses/[courseId]/page.tsx`
- `apps/web/src/components/course/CourseCommunityPanel.tsx`
- `apps/web/src/lib/courseCommunity.ts`
- `packages/shared/src/types/index.ts`
- `packages/shared/package.json`
- `package.json`

## 本地运行

```bash
npm install
npm run dev:web
```

默认前端会请求：

```bash
NEXT_PUBLIC_API_URL=http://localhost:3001/api
```

如果课程评论、教师评论或评分标准接口暂时不可用，页面会自动回退到前端内置示例数据和本地存储模式，方便继续联调和演示。

## 说明

- 当前课程详情页依然保留试卷与回忆卷入口。
- 教师目录是前端补充的辅助层，目的是让已有教师评论/评分标准接口在缺少教师列表接口时也能被真正使用起来。
