// Package config provides a `Config` stuct populated with variables parsed from a `.config` file.
package config

import (
	"bufio"
	"database/sql"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Conf struct {
	DB               *sql.DB
	URL              string
	PORT             int
	TelegramBotToken string
}

// NewConfig parses a `.config` file, reads/sanitizes its variables, then populates and returns a `Config` struct.
func NewConfig(db *sql.DB) *Conf {

	values, err := getValues()

	if err != nil {
		log.Fatal(err)
	}

	url, ok := values["URL"]
	if !ok || len(url) == 0 {
		log.Fatal("config: invalid URL")
	}

	p, ok := values["PORT"]
	if !ok || len(p) == 0 {
		log.Fatal("config: invalid PORT")
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		log.Fatal("config: invalid PORT value")
	}

	telegramBotToken, ok := values["TELEGRAM_BOT_TOKEN"]
	if !ok || len(telegramBotToken) == 0 {
		log.Fatal("config: invalid TELEGRAM_BOT_TOKEN")
	}

	return &Conf{
		DB:               db,
		URL:              url,
		PORT:             port,
		TelegramBotToken: telegramBotToken,
	}
}

func getValues() (map[string]string, error) {
	file, err := os.Open(".config")

	if err != nil {
		return nil, err
	}

	defer file.Close()

	return readFile(file)
}

func readFile(file io.Reader) (map[string]string, error) {
	values := make(map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if len(line) == 0 {
			continue
		}

		re := regexp.MustCompile(`([A-Za-z0-9_]+)\s*=\s*(.+)$`)

		matches := re.FindStringSubmatch(line)

		if len(matches) != 3 {
			continue
		}

		key := strings.TrimSpace(matches[1])
		value := strings.TrimSpace(matches[2])

		// if the value is quoted, only keep what's between the quotes and discard everything else
		re2 := regexp.MustCompile(`"(.+[^\\])"`)
		matches2 := re2.FindStringSubmatch(value)
		isQuotedValue := len(matches2) == 2

		if !isQuotedValue {
			value = strings.Split(value, "#")[0]
		} else {
			value = matches2[1]
		}

		value = strings.Trim(value, `" `)

		values[key] = value
	}

	return values, nil
}
