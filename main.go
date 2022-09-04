package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/e10k/matheque/cinemacity"
	"github.com/e10k/matheque/config"
	"github.com/e10k/matheque/storage"
	"github.com/e10k/matheque/telegram"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// backgroundTask checks Cinemacity for new movies at a varying interval;
// whenever a new movie is found, it is added to the database and, if it matches any existing watchers,
// it sends notifications to the relevant users.
func backgroundTask() {
	fetchMoviesAndSendUpdates()

	intervalMin, intervalMax := 5*60, 10*60+1
	ticker := time.NewTicker(time.Duration(intervalMin) * time.Second)
	for range ticker.C {
		fetchMoviesAndSendUpdates()

		// vary the ticker duration
		ticker.Reset(time.Duration(intervalMin+rand.Intn(intervalMax)) * time.Second)
	}
}

func fetchMoviesAndSendUpdates() {
	log.Print("Fetching movies...")

	films, err := cinemacity.GetFilms()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%d movies found", len(films))

	for _, film := range films {
		exists, err := storage.FilmExists(conf, film.Id)
		if err != nil {
			log.Fatal(err)
		}

		if exists {
			continue
		}

		romanianName, err := cinemacity.GetRomanianName(film.Link)
		if err != nil {
			log.Fatal(err)
		}

		_, err = storage.InsertFilm(conf, &storage.Film{
			Id:           film.Id,
			Name:         romanianName,
			OriginalName: film.Name,
			Link:         film.Link,
			PosterLink:   film.PosterLink,
		})

		if err != nil {
			log.Fatal(err)
		}

		log.Printf("new movie: %s (%s)", film.Name, romanianName)

		watcherMatches, err := storage.GetWatchersMatchingQuery(conf, film.Name, romanianName)

		if err != nil {
			log.Fatal(err)
		}

		for _, chatId := range watcherMatches {
			log.Printf("notify %d for movie %s\n", chatId, film.Name)

			sendNotification(telegram.NewNotification(chatId, film.Name, film.Link, film.PosterLink))
		}

		// be respectful to the server, in case multiple new movies have been found;
		// determining the movies' romanian names requires making a http request for each
		time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
	}
}

var botConfig telegram.BotConfig
var conf *config.Conf

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	conf = config.NewConfig(storage.GetDB())

	botConfig = telegram.NewBotConfig(
		conf.TelegramBotToken,
		conf.URL+"/webhook",
	)
}

func main() {
	go backgroundTask()

	err := telegram.SetWebhook(botConfig)
	if err != nil {
		log.Fatal(err)
	}
	err = telegram.SetCommands(botConfig)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/webhook", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == "POST" {
			body, _ := ioutil.ReadAll(req.Body)
			var payload map[string]interface{}
			err := json.Unmarshal(body, &payload)
			if err != nil {
				log.Println("unmarshal error", err)
				return
			}

			var u telegram.WebhookUpdate

			_, isMessage := payload["message"]
			_, isEditedMessage := payload["edited_message"]

			if isMessage {
				err = json.Unmarshal(body, &u)
			} else if isEditedMessage {
				var e telegram.WebhookUpdateEdited
				err = json.Unmarshal(body, &e)

				u.UpdateId = e.UpdateId
				u.Message = e.Message
			}

			_, err = storage.InsertMessage(conf, &storage.Message{
				MessageId:     u.Message.MessageId,
				FromId:        u.Message.From.Id,
				FromFirstName: u.Message.From.FirstName,
				ChatId:        u.Message.Chat.Id,
				ChatFirstName: u.Message.Chat.FirstName,
				Text:          u.Message.Text,
			})

			if err != nil {
				log.Fatal(err)
			}

			var response interface{}

			text := u.Message.Text
			chatId := u.Message.Chat.Id
			userId := u.Message.From.Id

			if strings.HasPrefix(text, "/start") {
				_, err = storage.UpdateChatStatus(conf, chatId, userId, storage.ChatIdle)
				if err != nil {
					log.Fatal(err)
				}

				_, err = storage.Subscribe(conf, chatId, userId)
				if err != nil {
					log.Fatal(err)
				}

				response = telegram.MakeResponseForStartCommand(chatId)
			} else if strings.HasPrefix(text, "/stop") {
				_, err = storage.UpdateChatStatus(conf, chatId, userId, storage.ChatIdle)
				if err != nil {
					log.Fatal(err)
				}

				_, err = storage.Unsubscribe(conf, chatId, userId)
				if err != nil {
					log.Fatal(err)
				}

				response = telegram.MakeResponseForStopCommand(chatId)
			} else if strings.HasPrefix(text, "/list") {
				_, err = storage.UpdateChatStatus(conf, chatId, userId, storage.ChatIdle)
				if err != nil {
					log.Fatal(err)
				}
				watchers, err := storage.GetWatchers(conf, chatId)
				if err != nil {
					log.Fatal(err)
				}
				response = telegram.MakeResponseForListCommand(&watchers, chatId)
			} else if strings.HasPrefix(text, "/add") {
				_, err := storage.UpdateChatStatus(conf, chatId, userId, storage.ChatWaitingForWatcherToAdd)
				if err != nil {
					log.Fatal(err)
				}
				response = telegram.MakeResponseForAddCommand(chatId)
			} else if strings.HasPrefix(text, "/remove") {
				_, err = storage.UpdateChatStatus(conf, chatId, userId, storage.ChatWaitingForWatcherToRemove)
				if err != nil {
					log.Fatal(err)
				}

				watchers, err := storage.GetWatchers(conf, chatId)
				if err != nil {
					log.Fatal(err)
				}
				response = telegram.MakeResponseForRemoveCommand(&watchers, chatId)
			} else {
				// at this point it is clear that the message received is not a command,
				// so the way it is handled will depend on the chat status
				chatStatus := storage.GetChatStatus(conf, chatId, userId)

				if chatStatus == storage.ChatWaitingForWatcherToAdd {
					// the message is a response to an /add command, so add the new watcher if it doesn't exist
					rowsAffected, err := storage.InsertWatcher(conf, chatId, text)
					var msg string
					if err != nil {
						log.Fatal(err)
					} else if rowsAffected == 0 {
						msg = "This looks like an invalid or already existing watcher. 🧐"
					}

					response = telegram.MakeResponseForWatcherAdded(chatId, msg)
				} else if chatStatus == storage.ChatWaitingForWatcherToRemove {
					// the message is a response to a /remove command, so remove the specified watcher, if found
					rowsAffected, err := storage.RemoveWatcher(conf, chatId, text)
					var msg string
					if err != nil {
						log.Fatal(err)
					} else if rowsAffected == 0 {
						msg = "Couldn't find a watcher named like that."
					}
					response = telegram.MakeResponseForWatcherRemoved(chatId, msg)
				} else {
					// the message is a random one
					response = telegram.MakeResponseForUnknownCommand(chatId)
				}

				// set the chat as idle
				_, err = storage.UpdateChatStatus(conf, chatId, userId, storage.ChatIdle)
				if err != nil {
					log.Fatal(err)
				}
			}

			// respond to the telegram message
			if response != nil {
				jsonData, err := json.Marshal(response)
				if err != nil {
					log.Fatal(err)
				}

				w.Header().Set("Content-Type", "application/json")
				_, err = w.Write(jsonData)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	})

	http.HandleFunc("/webhook-info", func(w http.ResponseWriter, req *http.Request) {
		resp, err := http.Get(botConfig.ApiUrl + "getWebhookInfo")
		if err != nil {
			log.Fatal(err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Fprintf(w, string(body))
	})

	err = http.ListenAndServe(fmt.Sprintf(":%d", conf.PORT), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func sendNotification(notification telegram.MethodSendPhoto) {

	jsonData, err := json.Marshal(notification)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.Post(botConfig.ApiUrl, "application/json", bytes.NewBuffer(jsonData))

	if err != nil {
		log.Fatal(err)
	}

	var res map[string]interface{}

	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		log.Println(err)
	}
}
