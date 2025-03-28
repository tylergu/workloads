package main

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
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

func execAsync(producer *kafka.Producer, sequence int) {
	go func() {
		topic := getEnvWithDefault("KAFKA_TOPIC", "topic")
		err := producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          []byte(fmt.Sprintf("%d", sequence))},
			nil, // delivery channel
		)
		if err != nil {
			fmt.Printf("Failed to produce message: %s\n", err)
		}
	}()
}

func computeRate(window []error) float32 {
	success := 0
	total := 0
	for i := 0; i < len(window); i++ {
		if window[i] == nil {
			success++
		}
		total++
	}
	return float32(success) / float32(total)
}

func consume(producer *kafka.Producer) {
	results := make([]error, 0)

	for e := range producer.Events() {
		switch ev := e.(type) {
		case *kafka.Message:
			if len(results) > WindowSize {
				results = results[1:]
			}
			results = append(results, ev.TopicPartition.Error)
			success_rate := computeRate(results)
			fmt.Printf("TS: [%s], Success Rate: [%f]\n",
				time.Now().Format(time.RFC3339), success_rate)
		}
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
		_, err := net.LookupIP(getEnvWithDefault("KAFKA_HOST", "test-cluster-mongos.acto-namespace.svc.cluster.local"))
		if err != nil {
			fmt.Printf("Waiting for SVC nslookup: %s\n", err)
			continue
		}
		break
	}
	p, err := kafka.NewProducer(
		&kafka.ConfigMap{
			"bootstrap.servers": fmt.Sprintf("%s:%s", getEnvWithDefault("KAFKA_HOST", "localhost"), getEnvWithDefault("KAFKA_PORT", "9092")),
			"client.id":         "myProducer",
			"acks":              "all",
			"security.protocol": "sasl_plaintext",
			"sasl.mechanisms":   "PLAIN",
			"sasl.username":     getEnvWithDefault("KAFKA_USER", "user"),
			"sasl.password":     getEnvWithDefault("KAFKA_PASSWORD", "password"),
		},
	)
	if err != nil {
		fmt.Printf("Failed to create producer: %s\n", err)
		os.Exit(1)
	}

	// adminCli, err := kafka.NewAdminClientFromProducer(p)
	// if err != nil {
	// 	fmt.Printf("Failed to create admin client: %s\n", err)
	// 	os.Exit(1)
	// }
	// defer adminCli.Close()

	// adminCli.CreateTopics(
	// 	context.Background(),
	// 	[]kafka.TopicSpecification{
	// 		{
	// 			Topic:             getEnvWithDefault("KAFKA_TOPIC", "topic"),
	// 			NumPartitions:     1,
	// 			ReplicationFactor: 3,
	// 		},
	// 	},
	// 	kafka.SetAdminOperationTimeout(5000),
	// )

	if err != nil {
		fmt.Printf("Failed to create producer: %s\n", err)
		os.Exit(1)
	}

	go consume(p)

	sequence := 0
	ticker = time.NewTicker(time.Second / TicksPerSecond)
	defer ticker.Stop()
	for range ticker.C {
		execAsync(p, sequence)
		sequence++
	}
}
