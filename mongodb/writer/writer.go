package main

import (
	"container/heap"
	"context"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	TicksPerSecond = 10
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

func execAsync(collection *mongo.Collection, result_chan chan Result, ts time.Time, sequence int) {
	go func() {
		doc := bson.D{
			bson.E{Key: "_id", Value: sequence},
			bson.E{Key: "sequence", Value: sequence},
		}
		_, err := collection.InsertOne(context.Background(), doc)
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
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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

	sequence := 0
	ticker := time.NewTicker(time.Second / TicksPerSecond)
	for range ticker.C {
		execAsync(collection, output, time.Now(), sequence)
		sequence++
	}
}
