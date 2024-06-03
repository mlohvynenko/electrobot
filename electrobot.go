package main

import (
	"os"
	"os/signal"
	"syscall"

	"electrobot/database"
	"electrobot/telegrambot"

	"github.com/coreos/go-systemd/daemon"
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
	log.SetLevel(log.DebugLevel)
}

/***********************************************************************************************************************
 * Main
 **********************************************************************************************************************/

func main() {
	log.Info("Hello, World!")

	db, err := database.New(database.Config{WorkingDir: "/var/electrobot"})
	if err != nil {
		log.Errorf("Failed to start bot due to DB error: %s", err)

		os.Exit(1)
	}

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Error("TELEGRAM_BOT_TOKEN env variable is not set")

		os.Exit(2)
	}

	bot, err := telegrambot.New(botToken, db)
	if err != nil {
		log.Errorf("Failed to start bot due to Telegram error: %s", err)

		os.Exit(3)
	}

	// Notify systemd
	if _, err = daemon.SdNotify(false, daemon.SdNotifyReady); err != nil {
		log.Errorf("Can't notify systemd: %s", err)
	}

	// handle SIGTERM
	c := make(chan os.Signal, 2) //nolint:gomnd
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Info("Shutting down...")
	bot.Close()
}
