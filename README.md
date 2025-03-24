# Ad Tracking API

This API is built using GoLang with Redis, Kafka, and SQL for managing and tracking video advertisements. It handles real-time metrics for ads (click count, CTR) and periodically updates the SQL database for long-term storage. Additionally, the API ensures robustness and handles high traffic efficiently.

---
## Features

- Record and store ad metadata
- Capture click events with timestamps and IP addresses
- Track real-time performance metrics (CTR, click count)
- Provide near real-time analytics with Redis
- Perform periodic data flush to SQL
- Efficient data processing using Kafka consumers
- Middleware for logging and error handling


## Table of Contents
- [Prerequisites](#prerequisites)
- [Environment Variables](#environment-variables)
- [API Endpoints](#api-endpoints)
- [Architecture Overview](#architecture-overview)
- [Redis Data Management](#redis-data-management)
- [SQL Schema](#sql-schema)
- [Error Handling](#error-handling)
- [Logging](#logging)
- [Building and Running](#building-and-running)
- [Docker Deployment](#docker-deployment)

---

## Prerequisites

- Go 1.22+
- Docker
- Redis
- Kafka
- MySQL

---

## Environment Variables

| Variable       | Description                          |
|-----------------|-------------------------------------|
| PROFILE         | Environment Profile (dev/prod)      |
| SQL_SDN         | SQL connection string               |
| PORT            | Port for the API                    |
| REDIS_URI       | Redis connection string             |
| KAFKA_URI       | Kafka connection string             |
| LOG_PATH        | Path to store application logs      |

---

## API Endpoints

### 1. Get Ads Data

```http
GET /ads
```

**Description:** Returns a list of ads with their metadata.

- **Response:**
  ```json
  [
    {
      "id": 1,
      "image_url": "https://example.com/ad1.jpg",
      "target_url": "https://example.com/click1"
    },
    {
      "id": 2,
      "image_url": "https://example.com/ad2.jpg",
      "target_url": "https://example.com/click2"
    }
  ]
  ```

---
### 2. **Track Ad Clicks**  
`POST /ads/click`
- **Request:**
```json
{
  "id": 2,
  "ip": "54.133.126.77",
  "playback_time": 200,
  "timeframe": 60.0
}
```
- **Response:**
```json
{
"message": "Click event received",
"status": "success"
}
```

### 2. **Get Real-time Metrics**  
`GET /ads/analytics/:id`
- **Response:**
```json
{
  "ad_id": 2,
  "clicks": 150,
  "ctr": 2.5,
  "timeframe": 60.0
}
```

---

## Architecture Overview

1. **Kafka Consumer:** Listens to click events and pushes data to Redis.
2. **Redis:** Stores ad click data with real-time updates.
3. **SQL Database:** Periodic flush of metrics from Redis to SQL.
4. **GoLang API:** Provides endpoints to register clicks and fetch metrics.

---

## Kafka Configuration

### Producer Configuration
- **Topic:** `adds-stream`

### Consumer Configuration
- **Topic:** `adds-stream`
- **Group ID:** `adds-stream-group`
- **MinBytes:** `1`
- **MaxBytes:** `10e6`
- **MaxWait:** `100ms`

---
## Redis Data Management

- **Click Tracking:**
  - Key: `ad-{ad_id}`
  - Fields: `clicks`, `playback_time`, `timeframe`, `ips`
- **Example:**
```json
{
  "clicks": 150,
}
```
- **CTR Tracking:**
  - Key: `time-{ad_id}`
  - Fields:  `impression`, `timeframe`, `playbacktime`
- **Example:**
```json
{
  "impression": 2,
  "timeframe":378,
  "playbacktime":800
}
```

## SQL Schema
```sql
CREATE TABLE ads_clicks (
  id INT PRIMARY KEY,
  ip VARCHAR(255),
  timestamp DATETIME,
  timeframe FLOAT
);

CREATE TABLE metadata_ads (
  id INT PRIMARY KEY,
  clicks INT,
  playback_time INT,
  image_url VARCHAR(255),
  target_url VARCHAR(255)  
);

***Insert some sample data for the above data table***

INSERT INTO metadata_ads (id, clicks, playback_time, image_url, target_url) 
VALUES 
(1, 0, 30, 'https://example.com/ad1.jpg', 'https://example.com/click1'),
(2, 0, 45, 'https://example.com/ad2.jpg', 'https://example.com/click2'),
(3, 0, 60, 'https://example.com/ad3.jpg', 'https://example.com/click3'),
(4, 0, 90, 'https://example.com/ad4.jpg', 'https://example.com/click4'),
(5, 0, 120, 'https://example.com/ad5.jpg', 'https://example.com/click5');
```

---

## Error Handling

- Global error handling is implemented using Gin Middleware.
- Panic recovery using `recover()` to prevent app crashes.
- Custom error types for better error management.

---

## Logging

- Structured logging using `lumberjack log`.
- Middleware for logging all requests and responses.
- Logs are stored using a rolling file appender.

---

## Building and Running

```bash
# Install dependencies
go mod tidy

# Alternatively, create a {production,development}.env file with variables:
# PROFILE=dev
# SQL_SDN=your_sql_connection
# PORT=8080
# REDIS_URI=your_redis_connection
# KAFKA_URI=your_kafka_connection
# LOG_PATH=/logs/app.log

# Run the application
go run ./cmd/main.go
```

---

## Docker Deployment

**Dockerfile Example:**
```dockerfile
# Stage 1: Build the Go Application
FROM golang:1.22 AS go_app
WORKDIR /app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o adds_app ./cmd/main.go

# Stage 2: Run Application
FROM alpine:latest
WORKDIR /app
COPY --from=go_app /app/adds_app .
EXPOSE 8080
CMD ["/app/adds_app"]
```

**Build and Run:**
```bash
docker build -t ad-tracking-api .
docker run -e PROFILE=production -e SQL_SDN="your_sql_connection" -e PORT=8080 -e REDIS_URI="your_redis_connection" -e KAFKA_URI="your_kafka_connection" -e LOG_PATH="/logs/app.log" ad-tracking-api
```

---

## Conclusion
This API ensures scalable, efficient, and reliable ad tracking using GoLang, Redis, Kafka, and SQL. With structured logging, global error handling, and periodic database updates, it is designed to handle high traffic loads effectively.

