package database

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var (
	Database *sql.DB
)

func ConnectSql(trigger chan bool) {
	dsn := os.Getenv("SQL_SDN")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("⛒ Connection Failed to Database")
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatal("⛒ Unable to Ping the  Database")
		log.Fatal(err)
	}
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(50)
	db.SetConnMaxLifetime(time.Minute * 5)
	Database = db
	log.Println("⛁ Connected to Database")
	trigger <- true
}
