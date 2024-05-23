package main

import (
	"electrobot/database"
	"electrobot/telegrambot"
	"os"

	log "github.com/sirupsen/logrus"
)

/***********************************************************************************************************************
 * Init
 **********************************************************************************************************************/

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: false,
		TimestampFormat:  "2006-01-02 15:04:05.000",
		FullTimestamp:    true,
	})
	log.SetOutput(os.Stdout)
}

/***********************************************************************************************************************
 * Main
 **********************************************************************************************************************/

func main() {
	log.Info("Hello, World!")

	db, err := database.New(database.Config{WorkingDir: "/tmp"})
	if err != nil {
		log.Errorf("Failed to start bot due to DB error: %s", err)

		os.Exit(1)
	}

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Error("TELEGRAM_BOT_TOKEN env variable is not set")

		os.Exit(2)
	}

	_, err = telegrambot.New(botToken, db)
	if err != nil {
		log.Errorf("Failed to start bot due to Telegram error: %s", err)

		os.Exit(3)
	}
}
