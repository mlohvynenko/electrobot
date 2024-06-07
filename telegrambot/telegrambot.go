// SPDX-License-Identifier: Apache-2.0
//
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package telegrambot

import (
	"context"
	"time"

	botApi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	log "github.com/sirupsen/logrus"
)

type Storage interface {
	UpdateEvent(eventType, event string) error
	NewEvent(eventType, event string) error
	StoreUserInfo(botApi.Message) error
	UserExists(int64) bool
	RemoveUserInfo(int64) error
	GetAllUsers() ([]int64, error)
	GetLatestEventDateTime(eventType string) (dateTime time.Time, err error)
}

type ElectroBot struct {
	botApi           *botApi.BotAPI
	updateChannel    botApi.UpdatesChannel
	updateConfig     botApi.UpdateConfig
	db               Storage
	cancelFunc       context.CancelFunc
	launchTime       time.Time
	lastShutdownTime time.Time
}

func New(token string, storage Storage) (bot *ElectroBot, err error) {
	bot = &ElectroBot{
		db:           storage,
		updateConfig: botApi.UpdateConfig{Offset: 0, Timeout: 60},
		launchTime:   time.Now().Local(),
	}

	bot.botApi, err = botApi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	if bot.lastShutdownTime, err = bot.getLastAliveTime(); err != nil {
		log.Warnf("Failed to get last alive time: %s", err)

		bot.lastShutdownTime = time.Now().Local()
	}

	if err = bot.notifyAllUsers(); err != nil {
		log.Errorf("Failed to notify all users on start: %s", err)

		return nil, err
	}

	ctx, cancelFunction := context.WithCancel(context.Background())
	bot.cancelFunc = cancelFunction

	bot.updateChannel = bot.botApi.GetUpdatesChan(bot.updateConfig)

	go bot.handler(ctx)

	return bot, nil
}

func (bot *ElectroBot) Close() {
	bot.botApi.StopReceivingUpdates()

	bot.cancelFunc()
}

func (bot *ElectroBot) getLastAliveTime() (time.Time, error) {
	return bot.db.GetLatestEventDateTime("Bot is alive")
}

func (bot *ElectroBot) notifyAllUsers() error {
	text := "Bot started at " + bot.launchTime.Local().Format("2006-01-02 15:04:05") +
		"\nLast alive time: " + bot.lastShutdownTime.Local().Format("2006-01-02 15:04:05")

	users, err := bot.db.GetAllUsers()
	if err != nil {
		log.Errorf("Failed to get all users: %s", err)

		return err
	}

	for _, user := range users {
		log.WithFields(log.Fields{"user": user}).Debug("Notifying user on start")

		msg := botApi.NewMessage(user, text)

		if _, err := bot.botApi.Send(msg); err != nil {
			log.Errorf("Failed to send message to user %d: %s", user, err)
		}
	}

	return nil
}

func (bot *ElectroBot) handleLastShutdownCommand() string {
	return "Last shutdown time is " + bot.lastShutdownTime.Local().Format("2006-01-02 15:04:05")
}

func (bot *ElectroBot) handleStartCommand(userID int64, messageBody *botApi.Message) string {
	exists := bot.db.UserExists(userID)
	if exists {
		return "You're already registered"
	}

	err := bot.db.StoreUserInfo(*messageBody)
	if err != nil {
		log.Errorf("Failed to store user info: %s", err)

		return "Failed to register you. Please try again later"
	}

	return "You've been successfully registered"
}

func (bot *ElectroBot) handleStopCommand(userID int64) string {
	err := bot.db.RemoveUserInfo(userID)
	if err != nil {
		log.Errorf("Failed to remove user info: %s", err)

		return "Failed to unregister you. Please try again later"
	}

	return "You've been successfully unregistered"
}

func (bot *ElectroBot) handleHelpCommand() string {
	return "Type /start to get started" +
		"\nType /stop to stop receiving notifications" +
		"\nType /lastshutdown to get the last shutdown time"
}

func (bot *ElectroBot) handleTGMessageCommand(updateMessage *botApi.Message) {
	log.WithField("chatInfo", updateMessage.Chat).Info("Got a new message")

	msg := botApi.NewMessage(updateMessage.Chat.ID, "")
	msg.ReplyToMessageID = updateMessage.MessageID

	switch updateMessage.Command() {
	case "lastshutdown":
		msg.Text = bot.handleLastShutdownCommand()
	case "start":
		msg.Text = bot.handleStartCommand(updateMessage.Chat.ID, updateMessage)
	case "stop":
		msg.Text = bot.handleStopCommand(updateMessage.Chat.ID)
	case "help":
	default:
		msg.Text = bot.handleHelpCommand()
	}

	if _, err := bot.botApi.Send(msg); err != nil {
		log.Errorf("Failed to send message: %s", err)
	}
}

func (bot *ElectroBot) handler(ctx context.Context) {
	log.WithField("Approximate lat shutdown time", bot.lastShutdownTime.Local().Format("2006-01-02 15:04:05")).Info("Bot was has been started")

	bot.updateIsAliveState()

	updateStateTicker := time.NewTicker(5 * time.Second)
	defer updateStateTicker.Stop()

	for {
		select {
		case <-updateStateTicker.C:
			bot.updateIsAliveState()

		case update := <-bot.updateChannel:
			if update.Message == nil {
				continue
			}

			if update.Message.IsCommand() {
				bot.handleTGMessageCommand(update.Message)
			}

		case <-ctx.Done():
			log.Info("Stopping bot")

			return
		}
	}
}

func (bot *ElectroBot) updateIsAliveState() {
	log.Debug("Bot is alive")

	err := bot.db.UpdateEvent("Bot is alive", "Bot is alive")
	if err == nil {
		return
	}

	err = bot.db.NewEvent("Bot is alive", "Bot is alive")
	if err != nil {
		log.Errorf("Failed to store event due to DB error: %s", err)
	}
}
