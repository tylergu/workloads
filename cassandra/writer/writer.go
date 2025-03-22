package main

import (
	"container/heap"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gocql/gocql"
)

const (
	TicksPerSecond = 10
	WindowSize     = 100
)

func getEnvWithDefault(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func writeAsync(session *gocql.Session, output chan Result, ts time.Time, sm *sync.Map, sequence int) {
	go func() {
		playerId := sequence % 1000
		coins := sequence / 1000
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var err error
		if coins > 0 {
			err = session.Query(
				"UPDATE player SET coins = ? WHERE id = ?",
				coins,
				playerId).WithContext(ctx).Exec()
		} else {
			err = session.Query(
				"INSERT INTO player (id, coins) VALUES (?, ?)",
				playerId,
				coins).WithContext(ctx).Exec()
		}
		if err != nil {
			log.Println(err)
		} else {
			sm.Store(playerId, coins)
		}

		output <- Result{
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

func check(cm *sync.Map, session *gocql.Session) {
	// Keep checking the consistency between the map and the database
	for {
		cm.Range(func(key, value any) bool {
			playerID := key.(string)
			rows := session.Query("SELECT id, coins FROM test.player WHERE id = ?", playerID).Iter()

			var id string
			var coins int
			for rows.Scan(&id, &coins) {
				if coins < value.(int) {
					log.Printf("Inconsistency detected: player %s has %d coins in the map but %d in the database\n", playerID, value.(int), coins)
				}
			}
			return true
		})
	}
}

func main() {
	cluster := gocql.NewCluster(
		getEnvWithDefault("CASSANDRA_HOST", "development-test-cluster-service.cass-operator.svc.cluster.local"),
	)
	port, err := strconv.Atoi(getEnvWithDefault("CASSANDRA_PORT", "9042"))
	if err != nil {
		log.Println(err)
		return
	}
	cluster.Port = port
	cluster.Consistency = gocql.Quorum
	cluster.ProtoVersion = 4
	cluster.ConnectTimeout = time.Second * 1
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username:              getEnvWithDefault("CASSANDRA_USER", ""),
		Password:              getEnvWithDefault("CASSANDRA_PASSWORD", ""),
		AllowedAuthenticators: []string{"org.apache.cassandra.auth.PasswordAuthenticator"},
	} //replace the username and password fields with their real settings, you will need to allow the use of the Instaclustr Password Authenticator.
	session, err := cluster.CreateSession()
	session.SetConsistency(gocql.Quorum)
	if err != nil {
		log.Println(err)
		return
	}
	defer session.Close()

	if err := session.Query("CREATE KEYSPACE IF NOT EXISTS test WITH REPLICATION = {'class': 'NetworkTopologyStrategy', 'replication_factor': 3}").Exec(); err != nil {
		log.Println(err)
		return
	}
	if err := session.Query("CREATE TABLE IF NOT EXISTS test.player (id int PRIMARY KEY, coins int)").Exec(); err != nil {
		log.Println(err)
		return
	}

	output := make(chan Result)
	go consume(output)

	cm := sync.Map{}
	go check(&cm, session)

	sequence := 0
	ticker := time.NewTicker(time.Second / TicksPerSecond)
	for range ticker.C {
		writeAsync(session, output, time.Now(), &cm, sequence)
		sequence++
	}
}
