package main

import (
	"container/heap"
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	CreatePlayerSQL      = "INSERT INTO player (id, coins) VALUES (?, ?)"
	GetPlayerSQL         = "SELECT id, coins FROM player WHERE id = ?"
	GetCountSQL          = "SELECT count(*) FROM player"
	GetPlayerWithLockSQL = GetPlayerSQL + " FOR UPDATE"
	UpdatePlayerSQL      = "UPDATE player set coins = ? WHERE id = ?"
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

func writeAsync(db *sql.DB, result_chan chan Result, ts time.Time, sm *sync.Map, sequence int) {
	go func() {
		playerId := sequence % 1000
		coins := sequence / 1000
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var request string
		var err error
		if coins > 0 {
			request = UpdatePlayerSQL
			_, err = db.ExecContext(ctx, request, coins, fmt.Sprintf("player-%d", playerId))
		} else {
			request = CreatePlayerSQL
			_, err = db.ExecContext(ctx, request, fmt.Sprintf("player-%d", playerId), coins)
		}

		if err != nil {
			fmt.Printf("Error: %s\n", err)
		} else {
			sm.Store(fmt.Sprintf("player-%d", playerId), coins)
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

func check(cm *sync.Map, db *sql.DB) {
	// Keep checking the consistency between the map and the database
	for {
		cm.Range(func(key, value any) bool {
			playerID := key.(string)
			rows, err := db.Query(GetPlayerSQL, playerID)
			if err != nil {
				fmt.Printf("Error: %s\n", err)
			}

			for rows.Next() {
				var id string
				var coins int
				if err := rows.Scan(&id, &coins); err != nil {
					fmt.Printf("Error: %s\n", err)
				}
				if coins != value {
					fmt.Printf("Error: inconsistency mismatch: %d != %d\n", coins, value)
				}
			}
			return true
		})
	}
}

func main() {
	db, err := sql.Open("mysql", getDSN())
	if err != nil {
		panic(err)
	}
	defer db.Close()
	db.SetConnMaxLifetime(time.Minute * 3)
	recreateTable(db)

	output := make(chan Result)
	go consume(output)

	cm := sync.Map{}
	go check(&cm, db)

	sequence := 0
	ticker := time.NewTicker(time.Second / TicksPerSecond)
	for range ticker.C {
		writeAsync(db, output, time.Now(), &cm, sequence)
		sequence++
	}
}
