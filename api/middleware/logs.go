package middleware

import (
	"io"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/natefinch/lumberjack.v2"
)

func SetupLogger(trigger chan bool) {
	logFile := &lumberjack.Logger{
		Filename:   os.Getenv("LOG_PATH"),
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}
	multiWriter := io.MultiWriter(logFile, os.Stdout)
	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("Logging setup complete.")
	trigger <- true
}

func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		status := c.Writer.Status()

		log.Printf("Time: %s | Method: %s | Path: %s | Status: %d ",
			time.Now().Format(time.RFC3339),
			c.Request.Method,
			c.Request.URL.Path,
			status,
		)
	}
}
