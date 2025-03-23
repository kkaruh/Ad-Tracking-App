package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	err := godotenv.Load("resources/properties.env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	err = godotenv.Load(fmt.Sprintf("resources/%s.env", os.Getenv("PROFILE")))
	if err != nil {
		log.Fatalf("Error loading profile")
	}
}
