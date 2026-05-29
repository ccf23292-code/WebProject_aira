package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"

	"warehouse-web/services"
)

// worker 进程定期：
//  1. 跑历史的 ProcessQuestion 占位逻辑（保留原行为）
//  2. 拉起 IngestJob 中处于 pending 状态的任务，做文件提取 + LLM 清洗 → awaiting_review
//
// 两条管道使用同一进程同一 ticker；ingest 比 question 更耗时（含 LLM 调用），
// 但都是阻塞调用 + 串行处理，不会互相挤占。
func main() {
	gormDB, err := services.InitPostgres()
	if err != nil {
		log.Fatalf("worker: failed to init postgres: %v", err)
	}

	// 复用同一 DSN 给 legacy ProcessQuestion 用 database/sql 驱动。
	rawDB, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("worker: failed to open raw sql: %v", err)
	}
	if err := rawDB.Ping(); err != nil {
		log.Fatalf("worker: db ping failed: %v", err)
	}

	llmService := services.NewLLMService(services.LoadLLMConfigFromEnv(), gormDB, services.NewPaperService(gormDB))
	visionClient := services.NewVisionClient(services.LoadVisionConfigFromEnv())
	paperService := services.NewPaperService(gormDB)
	ingestService := services.NewIngestService(gormDB, llmService, visionClient, paperService)

	if !llmService.Enabled() {
		log.Println("worker: LLM_API_KEY not set, ingest jobs will fail with llm_disabled until configured")
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	log.Println("worker started (question + ingest tick = 5s)")

	for {
		// 1) legacy
		if err := services.ProcessQuestion(rawDB); err != nil {
			log.Println("worker: process question failed:", err)
		}

		// 2) 把当前积压的 ingest 任务一次性吃完，再等下一拍
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			processed := ingestService.ProcessNextPending(ctx)
			cancel()
			if !processed {
				break
			}
		}

		<-ticker.C
	}
}
