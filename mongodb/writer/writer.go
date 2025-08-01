package main

import (
	"container/heap"
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	TicksPerSecond = 30
	WindowSize     = 100
)

func getDSN() string {
	host := getEnvWithDefault("MONGO_HOST", "test-cluster-mongos.acto-namespace.svc.cluster.local")
	port := getEnvWithDefault("MONGO_PORT", "27017")
	user := getEnvWithDefault("MONGO_USER", "root")
	password := getEnvWithDefault("MONGO_PASSWORD", "")

	return fmt.Sprintf("mongodb://%s:%s@%s:%s",
		user, password, host, port)
}

func getEnvWithDefault(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func execAsync(collection *mongo.Collection, result_chan chan Result, ts time.Time, sm *sync.Map, sequence int) {
	go func() {
		id := int32(sequence % 1000)
		epoch := int32(sequence / 1000)
		doc := bson.D{
			{Key: "_id", Value: id},
			{Key: "sequence", Value: epoch},
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		var err error
		if epoch > 0 {
			var result *mongo.UpdateResult
			result, err = collection.ReplaceOne(ctx,
				bson.D{{Key: "_id", Value: id}}, doc)
			if result.MatchedCount == 0 {
				_, err = collection.InsertOne(ctx, doc)
			}
		} else {
			_, err = collection.InsertOne(ctx, doc)
		}
		if err != nil {
			fmt.Printf("Error: %s\n", err)
		} else {
			sm.Store(id, epoch)
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

	first_success := false

	for result := range result_chan {
		if !first_success {
			if result.err == nil {
				first_success = true
			} else {
				continue
			}
		}
		heap.Push(&pq, &result)

		if pq.Len() > WindowSize {
			heap.Pop(&pq)
			success_rate := computeRate(pq)
			fmt.Printf("TS: [%s], Success Rate: [%f]\n",
				result.ts.Format(time.RFC3339), success_rate)
		}
	}
}

func check(cm *sync.Map, collection *mongo.Collection) {
	// Keep checking the consistency between the map and the database
	for {
		cm.Range(func(key, value any) bool {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			result := collection.FindOne(ctx, bson.D{
				bson.E{Key: "_id", Value: key},
			})
			if result.Err() != nil {
				fmt.Printf("Error reading from mongo: %s\n", result.Err())
				return false
			}

			var doc bson.D
			if err := result.Decode(&doc); err != nil {
				fmt.Printf("Error decoding document: %s\n", err)
				return false
			}

			if doc[1].Value.(int32) < value.(int32) {
				fmt.Printf("Inconsistency detected: [%T]%v != [%T]%v\n", doc[1].Value, doc[1].Value, value, value)
			}
			return true
		})
	}
}

// OKResponse is a standard MongoDB response
type OKResponse struct {
	Errmsg string `bson:"errmsg,omitempty" json:"errmsg,omitempty"`
	OK     int    `bson:"ok" json:"ok"`
}

func main() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		_, err := net.LookupIP(getEnvWithDefault("MONGO_HOST", "test-cluster-mongos.acto-namespace.svc.cluster.local"))
		if err != nil {
			fmt.Printf("Waiting for SVC nslookup: %s\n", err)
			continue
		}
		break
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx,
		options.Client().ApplyURI(getDSN()).SetRetryWrites(false))
	if err != nil {
		panic(err)
	}
	defer client.Disconnect(ctx)

	collection := client.Database("mongodb").Collection("test")

	output := make(chan Result)
	go consume(output)

	sm := &sync.Map{}
	go check(sm, collection)

	sequence := 0
	ticker = time.NewTicker(time.Second / TicksPerSecond)
	defer ticker.Stop()
	for range ticker.C {
		// t_ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		// defer cancel()
		// curr := client.Database("admin").RunCommand(t_ctx, bson.D{
		// 	{Key: "listDatabases", Value: 1},
		// 	{Key: "filter", Value: bson.D{{Key: "name", Value: "admin"}}},
		// 	{Key: "nameOnly", Value: true},
		// })
		// if curr.Err() != nil {
		// 	fmt.Printf("Error: %s\n", curr.Err())
		// }
		// resp := OKResponse{}
		// if err := curr.Decode(&resp); err != nil {
		// 	fmt.Printf("Error: %s\n", err)
		// }
		// if resp.OK == 0 {
		// 	fmt.Printf("list admin db not ok: %s\n", resp.Errmsg)
		// } else {
		// 	fmt.Printf("list admin db ok\n")
		// }

		execAsync(collection, output, time.Now(), sm, sequence)
		sequence++
	}
}
