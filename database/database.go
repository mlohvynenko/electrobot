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

package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/mattn/go-sqlite3" // ignore lint
	log "github.com/sirupsen/logrus"
)

/***********************************************************************************************************************
 * Consts
 **********************************************************************************************************************/

const (
	dbName      = "electrobot.db"
	busyTimeout = 60000
	journalMode = "WAL"
	syncMode    = "NORMAL"
)

/***********************************************************************************************************************
 * Types
 **********************************************************************************************************************/

// Database structure with database information.
type Database struct {
	sql *sql.DB
}

// Config structure with database configuration.
type Config struct {
	WorkingDir string
}

/***********************************************************************************************************************
 * Public
 **********************************************************************************************************************/

func New(config Config) (db *Database, err error) {
	dbFile := config.WorkingDir + "/" + dbName

	log.WithField("dbFile", dbFile).Info("Opening database")

	if err = os.MkdirAll(filepath.Dir(dbFile), 0o755); err != nil {
		log.Errorf("Failed to create database directory: %s", err)

		return db, err
	}

	sqlite, err := sql.Open("sqlite3", fmt.Sprintf("%s?_busy_timeout=%d&_journal_mode=%s&_sync=%s",
		dbFile, busyTimeout, journalMode, syncMode))
	if err != nil {
		log.WithField("dbPath", dbFile).Errorf("Failed to open database: %s", err)

		return db, err
	}

	db = &Database{sqlite}

	defer func() {
		if err != nil {
			db.Close()
		}
	}()

	if err = db.createTGUsersTable(); err != nil {
		log.Errorf("Failed to create tg_users table: %s", err)

		return db, err
	}

	if err = db.createEventTable(); err != nil {
		log.Errorf("Failed to create events table: %s", err)

		return db, err
	}

	return db, nil
}

// Close the database.
func (db *Database) Close() {
	if db.sql != nil {
		db.sql.Close()
	}
}

func (db *Database) StoreUserInfo(message tgbotapi.Message) error {
	_, err := db.sql.Exec(`INSERT INTO tg_users (user_id, username, first_name, last_name) VALUES (?, ?, ?, ?)`,
		message.From.ID, message.From.UserName, message.From.FirstName, message.From.LastName)

	return err
}

func (db *Database) GetAllUsers() (users []int64, err error) {
	rows, err := db.sql.Query(`SELECT user_id FROM tg_users`)
	if err != nil {
		log.Errorf("Failed to get all users: %s", err)

		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var user int64

		err = rows.Scan(&user)
		if err != nil {
			log.Errorf("Failed to scan user: %s", err)

			return nil, err
		}

		users = append(users, user)
	}

	return users, nil
}

func (db *Database) UserExists(userID int64) (exists bool) {
	exists = false

	err := db.sql.QueryRow(`SELECT EXISTS(SELECT 1 FROM tg_users WHERE user_id = ?)`, userID).Scan(&exists)
	if err != nil {
		log.Errorf("Failed to check if user exists: %s", err)
	}

	return exists
}

func (db *Database) RemoveUserInfo(userID int64) error {
	_, err := db.sql.Exec(`DELETE FROM tg_users WHERE user_id = ?`, userID)

	return err
}

/***********************************************************************************************************************
 * Private
 **********************************************************************************************************************/

func (db *Database) createTGUsersTable() error {
	_, err := db.sql.Exec(`CREATE TABLE IF NOT EXISTS tg_users (
		user_id INTEGER PRIMARY KEY NOT NULL,
		username TEXT,
		first_name TEXT,
		last_name TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)

	return err
}

func (db *Database) createEventTable() error {
	_, err := db.sql.Exec(`CREATE TABLE IF NOT EXISTS events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)

	return err
}
