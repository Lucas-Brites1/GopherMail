package models

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	return &Config{
		SMTPHost:     os.Getenv("SMTP_SERVER"),
		SMTPPort:     os.Getenv("SMTP_PORT"),
		SMTPUser:     os.Getenv("MAIL_FROM"),
		SMTPPassword: os.Getenv("MAIL_PASS"),
	}, nil
}
