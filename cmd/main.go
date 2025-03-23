package main

import (
	router "adds_app/api/handler"
	"adds_app/api/middleware"
	"adds_app/controller"
	"adds_app/internal/config"
	"adds_app/internal/database"
	"log"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
)

func main() {
	config.LoadEnv()
	color.Cyan("Server running on localhost:" + os.Getenv("PORT"))
	var wg sync.WaitGroup
	started := make(chan bool, 5)
	startComponent := func(name string, fn func(chan bool)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Printf("Starting %s...", name)
			fn(started)
			<-started
		}()
	}
	startComponent("Logger", middleware.SetupLogger)
	startComponent("SQL Database", database.ConnectSql)
	startComponent("Redis", config.ConnectRedis)
	startComponent("Kafka", config.InitializeKafka)
	startComponent("Kafka Consumer", controller.KafkaConsumer)
	go controller.StartFlushScheduler(10*time.Minute, started)
	wg.Wait()
	routes := router.Routes()
	color.Green(" Starting HTTP Server on port: %s", os.Getenv("PORT"))
	if err := routes.Run(":" + os.Getenv("PORT")); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
