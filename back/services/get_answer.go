package services

// 主要完成定期像GPT提问，获取答案，并存储到数据库中。
import (
	"database/sql"
	"log"
)

type Question struct {
	ID          uint64
	TestpaperID uint64
	SourceID    string
	Question    string
}

// 获取题目
func GetQuestions(db *sql.DB) (*Question, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var q Question
	err = tx.QueryRow(`
		SELECT id, testpaper_id, source_id, test
		FROM problems
		WHERE status = 'pending'
		LIMIT 1
		FOR UPDATE
	`).Scan(&q.ID, &q.TestpaperID, &q.SourceID, &q.Question)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec("UPDATE problems SET status = 'processing' WHERE id = $1", q.ID)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &q, nil
}

// 处理题目，调用API获取答案
func ProcessQuestion(db *sql.DB) error {
	q, err := GetQuestions(db)
	if err != nil {
		return err
	}

	if q == nil {
		log.Println("no pending question")
		return nil
	}

	answer, llm, err := callDeepSeekAPI(q.Question)
	if err != nil {
		_, _ = db.Exec("UPDATE problems SET status = 'error' WHERE id = $1", q.ID)
		log.Printf("Error calling API for question %d: %v", q.ID, err)
		return err
	}

	_, err = db.Exec(`
		UPDATE problems 
		SET status = 'processed' , llm_answer = $2, llm = $3
		WHERE id = $1`, q.ID, answer, llm)

	return err
}

func callDeepSeekAPI(question string) (string, string, error) {
	// 这里是调用DeepSeek API的逻辑，返回答案字符串
	// 实现调用不同llm的api，获取答案和llm名称
	return "这是一个模拟的答案", "DeepSeek", nil
}
