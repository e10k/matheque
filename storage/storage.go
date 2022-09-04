// Package storage provides functionality for managing the records of a sqlite database.
// If the database does not exist, it is created and then seeded (see `init.sql`).
package storage

import (
	"database/sql"
	"fmt"
	"github.com/e10k/matheque/config"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

type ChatStatus int8

const (
	ChatIdle ChatStatus = iota
	ChatWaitingForWatcherToAdd
	ChatWaitingForWatcherToRemove
)

type Film struct {
	Id           string
	Name         string
	OriginalName string
	Link         string
	PosterLink   string
}

type Message struct {
	MessageId     int
	FromId        int
	FromFirstName string
	ChatId        int
	ChatFirstName string
	Text          string
}

// GetDB returns a database handle.
// If the database doesn't exist, it is created and seeded.
func GetDB() *sql.DB {
	dbLocation := "./data/matheque.sqlite"

	isFreshDb := false

	_, err := os.Stat("./data")
	if os.IsNotExist(err) {
		err := os.Mkdir("data", 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	_, err = os.Stat(dbLocation)
	if os.IsNotExist(err) {
		f, err := os.Create(dbLocation)
		if err != nil {
			log.Fatal(err)
		}
		err = f.Close()
		if err != nil {
			log.Fatal(err)
		}
		isFreshDb = true
	}

	db, err := sql.Open("sqlite3", dbLocation)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	if isFreshDb {
		initSql, err := ioutil.ReadFile("./init.sql")
		if err != nil {
			log.Fatal(err)
		}

		_, err = db.Exec(string(initSql))

		if err != nil {
			log.Fatal(err)
		}
	}

	return db
}

func FilmExists(env *config.Conf, filmId string) (bool, error) {
	rows, err := env.DB.Query("SELECT original_id FROM films WHERE original_id=$1", filmId)

	defer rows.Close()

	if err != nil {
		return false, err
	}

	if rows.Next() {
		return true, nil
	}

	return false, nil
}

func InsertFilm(env *config.Conf, film *Film) (int64, error) {
	result, err := env.DB.Exec("INSERT INTO films (original_id, name, original_name, link, poster_link, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		film.Id, film.Name, film.OriginalName, film.Link, film.PosterLink, time.Now())

	if err != nil {
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()

	return rowsAffected, nil
}

func InsertMessage(env *config.Conf, m *Message) (int64, error) {
	result, err := env.DB.Exec("INSERT INTO messages (message_id, from_id, from_first_name, chat_id, chat_first_name, text, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		m.MessageId, m.FromId, m.FromFirstName, m.ChatId, m.ChatFirstName, m.Text, time.Now())

	if err != nil {
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()

	return rowsAffected, nil
}

func UpdateChatStatus(env *config.Conf, chatId int, userId int, status ChatStatus) (int64, error) {
	exists, err := chatExists(env, chatId, userId)

	if err != nil {
		return 0, err
	}

	var rowsAffected int64

	if exists {
		rowsAffected, err = updateChat(env, chatId, userId, status)
	} else {
		rowsAffected, err = insertChat(env, chatId, userId, status)
	}

	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func Subscribe(env *config.Conf, chatId int, userId int) (int64, error) {
	exists, err := chatExists(env, chatId, userId)

	if err != nil {
		return 0, err
	}

	if !exists {
		return 0, nil
	}

	result, err := env.DB.Exec("UPDATE chats SET subscribed = ?, updated_at = ? WHERE chat_id=? AND user_id=?", 1, time.Now(), chatId, userId)

	if err != nil {
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()

	return rowsAffected, nil
}

func Unsubscribe(env *config.Conf, chatId int, userId int) (int64, error) {
	exists, err := chatExists(env, chatId, userId)

	if err != nil {
		return 0, err
	}

	if !exists {
		return 0, err
	}

	result, err := env.DB.Exec("UPDATE chats SET subscribed = ?, updated_at = ? WHERE chat_id=? AND user_id=?", 0, time.Now(), chatId, userId)

	if err != nil {
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()

	return rowsAffected, nil
}

func chatExists(env *config.Conf, chatId int, userId int) (bool, error) {
	rows, err := env.DB.Query("SELECT * FROM chats WHERE chat_id=? AND user_id=?", chatId, userId)

	defer rows.Close()

	if err != nil {
		return false, err
	}

	if rows.Next() {
		return true, nil
	}

	return false, nil
}

func updateChat(env *config.Conf, chatId int, userId int, status ChatStatus) (int64, error) {
	result, err := env.DB.Exec("UPDATE chats SET status = ?, updated_at = ? WHERE chat_id=? AND user_id=?", status, time.Now(), chatId, userId)

	if err != nil {
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()

	return rowsAffected, nil
}

func insertChat(env *config.Conf, chatId int, userId int, status ChatStatus) (int64, error) {
	result, err := env.DB.Exec("INSERT INTO chats (chat_id, user_id, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)", chatId, userId, status, time.Now(), time.Now())

	if err != nil {
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()

	return rowsAffected, nil
}

func GetChatStatus(env *config.Conf, chatId int, userId int) ChatStatus {
	var status ChatStatus

	row := env.DB.QueryRow("SELECT status FROM chats WHERE chat_id=$1 AND user_id=$2", chatId, userId)

	err := row.Scan(&status)

	if err != nil {
		status = 0
	}

	return status
}

func InsertWatcher(env *config.Conf, chatId int, keywords string) (int64, error) {
	keywords = strings.Trim(keywords, " ")

	if len(keywords) == 0 {
		return 0, nil
	}

	exists, err := watcherExists(env, chatId, keywords)

	if err != nil {
		return 0, err
	}

	if exists {
		return 0, nil
	}

	result, err := env.DB.Exec("INSERT INTO watchers (chat_id, keywords, keywords_normalised, created_at) VALUES (?, ?, ?, ?)", chatId, keywords, NormaliseString(keywords), time.Now())
	if err != nil {
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()

	return rowsAffected, nil
}

func watcherExists(env *config.Conf, chatId int, keywords string) (bool, error) {
	rows, err := env.DB.Query("SELECT * FROM watchers WHERE chat_id=$1 AND keywords=$2 COLLATE NOCASE", chatId, keywords)

	defer rows.Close()

	if err != nil {
		return false, err
	}

	if rows.Next() {
		return true, nil
	}

	return false, nil
}

func RemoveWatcher(env *config.Conf, chatId int, keywords string) (int64, error) {
	exists, err := watcherExists(env, chatId, keywords)

	if err != nil {
		return 0, err
	}

	if !exists {
		return 0, nil
	}

	result, err := env.DB.Exec("DELETE FROM watchers WHERE chat_id=? AND keywords=?", chatId, keywords)
	if err != nil {
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()

	return rowsAffected, nil
}

func GetWatchers(env *config.Conf, chatId int) ([]string, error) {
	rows, err := env.DB.Query("SELECT keywords FROM watchers WHERE chat_id = ? ORDER BY keywords COLLATE NOCASE", chatId)
	if err != nil {
		return nil, fmt.Errorf("fetching watchers: %v", err)
	}
	defer rows.Close()

	var data []string

	for rows.Next() {
		var k string
		err = rows.Scan(&k)
		if err != nil {
			return nil, fmt.Errorf("reading watchers: %v", err)
		}

		data = append(data, k)
	}

	return data, nil
}

func GetWatchersMatchingQuery(env *config.Conf, query1 string, query2 string) ([]int, error) {
	query := NormaliseString(query1 + " " + query2)
	preparedQuery := strings.Join(strings.Split(query, " "), " OR ")
	rows, err := env.DB.Query("select chat_id from watchers_fts where keywords_normalised match ? GROUP BY chat_id ORDER BY rank", preparedQuery)
	if err != nil {
		return nil, fmt.Errorf("fetching watchers for query %s: %v", query, err)
	}

	defer rows.Close()

	var data []int

	for rows.Next() {
		var chatId int
		err = rows.Scan(&chatId)
		if err != nil {
			return nil, fmt.Errorf("reading watchers: %v", err)
		}

		data = append(data, chatId)
	}

	return data, nil
}

// NormaliseString prepares watchers and movie names for being compared.
func NormaliseString(s string) string {
	r1, err := regexp.Compile("[^a-zA-Z\u00C0-\u024F\u1E00-\u1EFF ]+")
	if err != nil {
		return s
	}

	s1 := r1.ReplaceAllString(s, "")

	r2, err := regexp.Compile(" {2,}")
	if err != nil {
		return s1
	}

	s2 := r2.ReplaceAllString(s1, " ")

	return strings.ToLower(s2)
}
