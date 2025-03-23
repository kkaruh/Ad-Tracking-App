package config

import (
	"log"
	"os"

	"github.com/segmentio/kafka-go"
)

var KafkaWriter *kafka.Writer

func InitializeKafka(trigger chan bool) {
	KafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(os.Getenv("KAFKA_URI")),
		Topic:    "adds-stream",
		Balancer: &kafka.LeastBytes{},
	}
	log.Println("Kafka Initialized")
	trigger <- true
}
