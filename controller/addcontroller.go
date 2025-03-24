package controller

import (
	"adds_app/internal/config"
	"adds_app/internal/database"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/segmentio/kafka-go"
)

type Ad struct {
	ID        int    `json:"id"`
	ImageURL  string `json:"image_url"`
	TargetURL string `json:"target_url"`
}
type ClickEvent struct {
	AddId        int     `json:"id" validate:"required,gt=0"`
	Timestamp    string  `json:"timestamp"`
	Ip           string  `json:"ip" validate:"required,ip"`
	PlaybackTime float64 `json:"playback_time" validate:"required,gt=0"`
	Timeframe    float64 `json:"timeframe" validate:"required,gt=0"`
}

type Analytics struct {
	ID           int     `json:"id"`
	Clicks       int     `json:"clicks"`
	PlaybackTime float64 `json:"playback_time"`
	Timeframe    float64 `json:"timeframe"`
}

func (an Analytics) calculateImpression() int {
	return int(an.PlaybackTime / an.Timeframe)
}
func GetAds(ctx *gin.Context) {
	rows, err := database.Database.Query("SELECT id,image_url,target_url FROM metadata_ads")
	if err != nil {
		log.Println("Error executing query:", err)
		ctx.JSON(500, gin.H{"error": "Failed to fetch ads"})
		return
	}
	defer rows.Close()

	var ads []Ad
	for rows.Next() {
		var ad Ad
		if err := rows.Scan(&ad.ID, &ad.ImageURL, &ad.TargetURL); err != nil {
			log.Println("Error scanning ad:", err)
			continue
		}
		ads = append(ads, ad)
	}
	if err = rows.Err(); err != nil {
		log.Println("Error in rows iteration:", err)
		ctx.JSON(500, gin.H{"error": "Error processing results"})
		return
	}
	log.Println("Response for the ads")
	ctx.JSON(200, ads)
}
func validateInput(clickEvent ClickEvent) (bool, string) {
	var validate = validator.New()
	err := validate.Struct(clickEvent)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			return false, fmt.Sprintf("Validation failed on field '%s': %s\n", err.Field(), err.Tag())
		}
	} else {
		log.Println("Validated successful!")
	}
	return true, ""
}
func PostAdds(ctx *gin.Context) {
	var clickEvent ClickEvent
	if err := ctx.ShouldBindJSON(&clickEvent); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	valid, error := validateInput(clickEvent)
	if !valid {
		panic(error)
	}
	clickEvent.Timestamp = time.Now().Format(time.RFC3339)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		updateMetricsInRedis(fmt.Sprint(clickEvent.AddId), clickEvent.Timeframe, clickEvent.PlaybackTime)
		defer wg.Done()
	}()

	eventJSON, err := json.Marshal(clickEvent)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode event"})
		return
	}
	pushchan := make(chan bool)
	go func() {
		err = config.KafkaWriter.WriteMessages(context.Background(),
			kafka.Message{
				Key:   []byte(fmt.Sprint(clickEvent.AddId)),
				Value: eventJSON,
			},
		)
		if err != nil {
			pushchan <- false
			return
		}
		log.Println("Pushed to Kafka")
		pushchan <- true
	}()
	if !<-pushchan {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send to Kafka"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "success", "message": "Click event received"})
	wg.Wait()
}

func KafkaConsumer(trigger chan bool) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{os.Getenv("KAFKA_URI")},
		Topic:    "adds-stream",
		GroupID:  "adds-stream-group",
		MinBytes: 1,
		MaxBytes: 10e6,
		MaxWait:  100 * time.Millisecond,
	})

	log.Println("Listening for click events...")
	trigger <- true
	go func() {
		for {
			message, err := reader.ReadMessage(context.Background())
			if err != nil {
				log.Println(err)
			}
			var clickEvent ClickEvent
			if err := json.Unmarshal(message.Value, &clickEvent); err != nil {
				log.Println("Error decoding message:", err)
				continue
			}
			log.Printf("Inserting into the database")
			var wg sync.WaitGroup
			insert := make(chan bool, 1)
			wg.Add(1)
			go func() {
				defer wg.Done()
				if <-InsertClick(clickEvent, insert) {
					updateMetaClicks(clickEvent.AddId, insert)
					<-insert
				}
			}()
			log.Println("DB updation Done")
			wg.Wait()
			close(insert)

		}
	}()

}
func updateMetaClicks(adID int, insert chan bool) {
	tx, err := database.Database.Begin()
	if err != nil {
		log.Println("Failed to begin transaction:", err)
		return
	}
	log.Println("Incrementing Clicks")
	_, err = tx.Exec("UPDATE metadata_ads SET clicks = clicks + 1 WHERE id = ?", adID)
	if err != nil {
		log.Println("Failed to insert clicks data:", err)
		tx.Rollback()
		return
	}
	err = tx.Commit()
	if err != nil {
		log.Println("Failed to commit transaction:", err)
		return
	}
	log.Println("Incremented Clicks")
	insert <- true
}
func InsertClick(event ClickEvent, insert chan bool) chan bool {
	query := "INSERT INTO ads_clicks (id, ip, timestamp, timeframe) VALUES (?, ?, ?, ?)"
	_, err := database.Database.Exec(query, event.AddId, event.Ip, event.Timestamp, event.Timeframe)
	if err != nil {
		log.Printf("Failed to insert click event: %v", err)
		insert <- false
		return insert
	}
	log.Println("Cick event inserted successfully")
	insert <- true
	return insert
}

func updateMetricsInRedis(adID string, timeframe float64, playbacktime float64) {
	key := fmt.Sprintf("ad-%s", adID)
	timestampKey := fmt.Sprintf("time-%s", adID)
	impression := int(playbacktime / timeframe)
	_, err := config.RedisClient.HIncrBy(config.Ctx, key, "clicks", 1).Result()
	if err != nil {
		log.Println("Failed to increment clicks in Redis:", err)
		return
	}
	myMap := map[string]interface{}{
		"impression":   impression,
		"timeframe":    timeframe,
		"playbacktime": playbacktime,
	}
	err = config.RedisClient.HSet(config.Ctx, timestampKey, myMap).Err()
	if err != nil {
		log.Println("Error pushing time map to redis:", err)
		return
	}
	log.Printf("Metrics updated for ad %s. Clicks incremented and timestamp recorded.", adID)
}

func StartFlushScheduler(interval time.Duration, trigger chan bool) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	log.Println("Flusher Initialized ....")
	trigger <- true
	for range ticker.C {
		log.Println("Starting scheduled flush to SQL...")

		keys, err := config.RedisClient.Keys(config.Ctx, "ad-*").Result()
		if err != nil {
			log.Println("Failed to fetch clicks from Redis:", err)
			return
		}
		inserted := make(chan bool)
		for _, key := range keys {
			go FlushMetricsToSQL(key, inserted)
			<-inserted
		}
	}
}

func FetchFromDb(started chan<- bool) {
	query := `
	SELECT 
		a.id,
		m.clicks,
		m.playback_time,
		a.timeframe
	FROM 
		ads_clicks AS a
	JOIN 
		metadata_ads AS m ON a.id = m.id
	WHERE 
		a.timeframe IS NOT NULL
	GROUP BY
		a.id,
		m.clicks,
		m.playback_time,
		a.timeframe;
`

	rows, err := database.Database.Query(query)
	if err != nil {
		log.Println("Failed to execute query:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ad Analytics
		if err := rows.Scan(&ad.ID, &ad.Clicks, &ad.PlaybackTime, &ad.Timeframe); err != nil {
			log.Println("Error scanning row:", err)
			continue
		}
		key := fmt.Sprintf("ad-%d", ad.ID)
		timestampKey := fmt.Sprintf("time-%d", ad.ID)
		_, err := config.RedisClient.HSet(config.Ctx, key, "clicks", ad.Clicks).Result()
		if err != nil {
			log.Println("Failed to increment clicks in Redis:", err)
			return
		}
		impression := ad.calculateImpression()
		err = config.RedisClient.HSet(config.Ctx, timestampKey, map[string]interface{}{
			"impression":   impression,
			"timeframe":    ad.Timeframe,
			"playbacktime": ad.PlaybackTime,
		}).Err()
		if err != nil {
			fmt.Println("Error pushing time map to redis:", err)
			return
		}
	}

	if err := rows.Err(); err != nil {
		log.Println("Error iterating rows:", err)
	}
	fmt.Println("Fetched Ads completed")
	started <- true
}

func GetAnalytics(ctx *gin.Context) {
	adID := ctx.Param("id")
	click_key := fmt.Sprintf("ad-%s", adID)
	time_key := fmt.Sprintf("time-%s", adID)
	metrics, err := config.RedisClient.HGetAll(config.Ctx, click_key).Result()
	if err != nil || len(metrics) == 0 {
		log.Println("Failed to fetch clicks from Redis:", err)
		trigger := make(chan bool)
		go FetchFromDb(trigger)
		<-trigger
		ctx.JSON(http.StatusNotFound, gin.H{"msg": "No metrics found "})
		return
	}
	time_map, err := config.RedisClient.HGetAll(config.Ctx, time_key).Result()
	if err != nil {
		log.Println("Failed to fetch metrics from Redis:", err)
		ctx.JSON(http.StatusRequestTimeout, gin.H{"msg": "retry network error"})
	}
	click, err := strconv.ParseFloat(metrics["clicks"], 64)
	if err != nil {
		log.Println("Failed to fetch metrics from Redis:", err)
		return
	}
	impress, err := strconv.ParseFloat(time_map["impression"], 64)
	if err != nil {
		log.Println("Failed to fetch metrics from Redis:", err)
		return
	}
	ctr := int((click / impress) * 100)
	if ctr < 0 {
		ctr = 0
	}
	ctx.JSON(http.StatusOK, gin.H{
		"ad_id":      adID,
		"clicks":     click, //adds ,user
		"impression": impress,
		"CTR":        ctr, //adds})
	})
}

func FlushMetricsToSQL(adID string, isInserted chan bool) {
	time_key := fmt.Sprintf("time-%s", adID[3:])
	log.Println("Started Flushing for id ", adID, time_key)
	clicks, err := config.RedisClient.HGetAll(config.Ctx, adID).Result()
	if err != nil {
		log.Println(err)
	}
	if len(clicks) == 0 {
		isInserted <- true
	}
	tx, err := database.Database.Begin()
	if err != nil {
		log.Println("Failed to begin transaction:", err)
		return
	}
	log.Println("Clicks found", clicks)
	_, err = tx.Exec("INSERT INTO metadata_ads (id, clicks) VALUES (?, ?) ON DUPLICATE KEY UPDATE clicks = ?",
		adID[3:], clicks["clicks"], clicks["clicks"])
	if err != nil {
		log.Println("Failed to insert clicks data:", err)
		tx.Rollback()
		return
	}
	err = tx.Commit()
	if err != nil {
		log.Println("Failed to commit transaction:", err)
		return
	}
	log.Printf("Flushed data for ad %s to SQL. Clicks: %s", adID[3:], clicks["clicks"])
	isInserted <- true
}
