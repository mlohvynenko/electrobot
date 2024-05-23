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
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	log "github.com/sirupsen/logrus"
)

type Storage interface {
	StoreUserInfo(tgbotapi.Message) error
	UserExists(int64) bool
	RemoveUserInfo(int64) error
	GetAllUsers() ([]int64, error)
}

func New(token string, storage Storage) (*tgbotapi.BotAPI, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	bot.Debug = true

	log.Debug("Authorized on account ", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	users, err := storage.GetAllUsers()
	if err != nil {
		log.Errorf("Failed to get all users: %s", err)

		return nil, err
	}

	for _, user := range users {
		log.WithFields(log.Fields{"user": user}).Info("User")

		msg := tgbotapi.NewMessage(user, "Bot started")
		bot.Send(msg)
	}

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil { // If we got a message
			log.WithFields(log.Fields{"user": update.Message.From.UserName, "userId": update.Message.From.ID, "message": update.Message.Text}).Info("Got a message")

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

			switch update.Message.Command() {
			case "start":
				exists := storage.UserExists(update.Message.From.ID)
				if exists {
					msg.Text = "You're already registered"

					break
				}

				err = storage.StoreUserInfo(*update.Message)
				if err != nil {
					log.Errorf("Failed to store user info: %s", err)

					msg.Text = "Failed to register you. Please try again later"

					break
				}

				msg.Text = "You've been successfully registered"
			case "stop":
				err = storage.RemoveUserInfo(update.Message.From.ID)
				if err != nil {
					log.Errorf("Failed to remove user info: %s", err)

					msg.Text = "Failed to unregister you. Please try again later"
					break
				}

				msg.Text = "You've been successfully unregistered"
			case "help":
				msg.Text = "Type /start to get started"
			default:
				msg.Text = "I can handle only commands. Use /help to get help"
			}

			msg.ReplyToMessageID = update.Message.MessageID

			bot.Send(msg)
		}
	}

	return bot, nil
}
