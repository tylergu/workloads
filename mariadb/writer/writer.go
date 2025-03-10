package main

import (
	"container/heap"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	CreatePlayerSQL      = "INSERT INTO player (id, coins, goods) VALUES (?, ?, ?)"
	GetPlayerSQL         = "SELECT id, coins, goods FROM player WHERE id = ?"
	GetCountSQL          = "SELECT count(*) FROM player"
	GetPlayerWithLockSQL = GetPlayerSQL + " FOR UPDATE"
	UpdatePlayerSQL      = "UPDATE player set goods = goods + ?, coins = coins + ? WHERE id = ?"
	GetPlayerByLimitSQL  = "SELECT id, coins, goods FROM player LIMIT ?"
	DropTableSQL         = "DROP TABLE IF EXISTS player"
	CreateTableSQL       = "CREATE TABLE player ( `id` VARCHAR(36), `coins` INTEGER, `goods` INTEGER, PRIMARY KEY (`id`) );"
	TicksPerSecond       = 10
	WindowSize           = 100
)

func recreateTable(db *sql.DB) {
	if _, err := db.Exec(DropTableSQL); err != nil {
		panic(err)
	}
	if _, err := db.Exec(CreateTableSQL); err != nil {
		panic(err)
	}
}

func getDSN() string {
	tidbHost := getEnvWithDefault("MARIADB_HOST", "127.0.0.1")
	tidbPort := getEnvWithDefault("MARIADB_PORT", "4000")
	tidbUser := getEnvWithDefault("MARIADB_USER", "root")
	tidbPassword := getEnvWithDefault("MARIADB_PASSWORD", "")
	tidbDBName := getEnvWithDefault("MARIADB_DATABASE", "test")
	useSSL := getEnvWithDefault("USE_SSL", "false")

	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&tls=%s",
		tidbUser, tidbPassword, tidbHost, tidbPort, tidbDBName, useSSL)
}

func getEnvWithDefault(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func execAsync(db *sql.DB, result_chan chan Result, ts time.Time, sql string, args ...any) {
	go func() {
		_, err := db.Exec(sql, args...)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
		}
		result_chan <- Result{
			err: err,
			ts:  ts,
		}
	}()
}

func computeRate(window PriorityQueue) float32 {
	success := 0
	total := 0
	for i := 0; i < len(window); i++ {
		if window[i].err == nil {
			success++
		}
		total++
	}
	return float32(success) / float32(total)
}

func consume(result_chan chan Result) {
	pq := make(PriorityQueue, 0)
	heap.Init(&pq)

	for result := range result_chan {
		heap.Push(&pq, &result)

		if pq.Len() > WindowSize {
			heap.Pop(&pq)
			success_rate := computeRate(pq)
			fmt.Printf("TS: [%s], Success Rate: [%f]\n",
				result.ts.Format(time.RFC3339), success_rate)
		}
	}
}

func main() {
	db, err := sql.Open("mysql", getDSN())
	if err != nil {
		panic(err)
	}
	defer db.Close()
	recreateTable(db)

	output := make(chan Result)
	go consume(output)

	sequence := 0
	ticker := time.NewTicker(time.Second / TicksPerSecond)
	for range ticker.C {
		execAsync(db, output, time.Now(), CreatePlayerSQL,
			fmt.Sprintf("player-%d", sequence), 0, 0)
		sequence++
	}
}
