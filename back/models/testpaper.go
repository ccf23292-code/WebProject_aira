package models

import "time"

// TestPaper 对应数据库 testpapers 表，表示一份试卷。
//
// 结构化字段（Year/Semester/ExamType）由 ingest 流程填充，用于：
//   - 自动合并同一场考试的多次上传（见 IngestService.publishQuestions）
//   - 未来按学期/年份做筛选与数据分析
//
// 老的导入数据（cmd/import_papers）只有 Name 字段，三段为零值，不影响显示。
// 不在 DB 层加复合唯一约束，避免与历史数据冲突；合并逻辑放在 service 层做。
type TestPaper struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	CourseID  string `gorm:"size:64;index"            json:"course_id"`
	Name      string `gorm:"size:256"                 json:"name"`
	Year      int    `gorm:"index"                    json:"year,omitempty"`
	Semester  string `gorm:"size:16;index"            json:"semester,omitempty"`
	ExamType  string `gorm:"size:32;index"            json:"exam_type,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// 试卷的合法 Semester / ExamType 取值（前端下拉同源）。
// 用 var 定义而非 const 是为了让前后端可共享同一份名单（前端从 API 拿）。
var (
	PaperSemesters = []string{"春夏", "秋冬", "暑期", "全年"}
	PaperExamTypes = []string{"期中", "期末", "小测", "模考", "自测", "其他"}
)
