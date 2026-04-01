- 将内存map更改为使用`postgresql`数据库，删除所有种子数据仅保存管理员用户。
- 添加新增回忆卷(与数据库中的已经存在的历年卷不同，需要用户根据刚考完试的记忆回忆考试内容，不是更改数据库中已经存在的)功能，所有注册用户均可在线编辑
  - 前端在每个课程界面有回忆历年卷选项，可进入编辑界面(编辑界面记为`page1`)
  - 进入编辑界面后，陈列所有题目类型list，后面有添加题目选项，添加题目前需要先确定题号(每个题目有自己的小题号，但是切换题目类型之后，比如从选择变为了填空题，小题号可能会重置，也可能不会，这点要考虑)。
  - 然后进入题目编辑界面，目前先仅支持markdown语法，允许题目中含有图片，后面会允许上传图片进行ORC识别，答案不一定要有，对于同一题目，不同人允许上传多份题目。
  - 每个题目要记录上传者(即用户)，有一个支持度记录，一个用户一个题目仅限支持一次，一个用户可同时支持多个题目。
  - 前端最好在`page1`每个题目类型下面显示题号相同的题目的支持度最高的那个，点击相应的题目会进入`page2`，会陈列当前题号下的所有题目，每个题目允许所有用户进行二次编辑，会有对应的评论区，其余人可在下面发表意见。
  - 在时机合适之后将同一题号支持度最高的题目保留下来，其余删除。

  ---

  回忆卷 API（新增，写在此处供前后端对齐）

  基础约定
  - 统一响应沿用 { code, message, data }
  - 需要登录（Authorization: Bearer <token>）
  - 题目内容与答案均使用 Markdown，图片通过 Markdown 链接
  - source 在此处为上传者用户 ID（source_user_id）

  1) 回忆卷
  - GET /api/recall/courses/:course_id/papers
    - 返回课程下回忆卷列表
  - POST /api/recall/courses/:course_id/papers
    - body: { "title": "2026春夏学期期末回忆卷" }
    - 返回新建回忆卷

  2) 题型与题号
  - GET /api/recall/papers/:paper_id/question-types
    - 返回题型与当前最大题号（用于 page1 题号规划）
    - response: [{ "question_type": "singleChoice", "max_sequence": 12 }]

  3) page1 顶部列表（同题号支持度最高）
  - GET /api/recall/papers/:paper_id/questions/top
    - query: question_type (可选，不传则全题型)
    - 返回每个题号支持度最高的一条题目

  4) page2 题目版本列表
  - GET /api/recall/papers/:paper_id/questions
    - query: question_type, sequence
    - 返回指定题号下全部题目版本（按支持度倒序）

  5) 新增题目
  - POST /api/recall/papers/:paper_id/questions
    - body: {
        "question_type": "singleChoice",
        "sequence": 3,
        "content": "题干 Markdown",
        "answer": "B",
        "options": [{ "option": "A", "text": "选项A" }]
      }

  6) 二次编辑题目
  - PATCH /api/recall/questions/:question_id
    - body: { "content": "新版题干", "answer": "", "options": [] }

  7) 支持度
  - POST /api/recall/questions/:question_id/support
    - 每人每题仅一次，返回更新后的题目

  8) 评论区
  - GET /api/recall/questions/:question_id/comments?page=1&size=10
  - POST /api/recall/questions/:question_id/comments
    - body: { "content": "我觉得答案需要补充..." }

  数据库表建议（回忆卷专用，方便后续迁移）
  - recall_papers: id, course_id, title, created_by, created_at, updated_at
  - recall_questions: id, paper_id, question_type, sequence, content, answer, options_json, source_user_id, support_count, last_editor_id, created_at, updated_at
  - recall_question_supports: id, question_id, user_id, created_at (question_id + user_id 唯一)
  - recall_question_comments: id, question_id, user_id, content, created_at, updated_at

  运行时数据库配置
  - 读取 DATABASE_URL 或 POSTGRES_DSN