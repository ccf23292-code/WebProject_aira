package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	"warehouse-web/services"

	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("question worker started")

	for {
		if err := services.ProcessQuestion(db); err != nil {
			log.Println("process question failed:", err)
		}

		<-ticker.C
	}
}
