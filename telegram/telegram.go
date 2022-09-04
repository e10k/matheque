// Package telegram manages a Telegram bot, reacts to queries and creates responses.
package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type BotConfig struct {
	Token      string
	ApiUrl     string
	WebhookUrl string
}

type Command struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type MethodSetMyCommands struct {
	Method   string    `json:"method"`
	Commands []Command `json:"commands"`
}

type ReplyMarkupWithKeyboard struct {
	Keyboard        [][]string `json:"keyboard"`
	OneTimeKeyboard bool       `json:"one_time_keyboard"`
	ResizeKeyboard  bool       `json:"resize_keyboard"`
}

type ReplyMarkupWithoutKeyboard struct {
	RemoveKeyboard bool `json:"remove_keyboard"`
}

type MethodSendPhoto struct {
	Method    string `json:"method"`
	ChatId    int    `json:"chat_id"`
	Photo     string `json:"photo"`
	Caption   string `json:"caption"`
	ParseMode string `json:"parse_mode"`
}

type MethodSendMessageWithKeyboard struct {
	Method      string                  `json:"method"`
	ChatId      int                     `json:"chat_id"`
	Text        string                  `json:"text"`
	ParseMode   string                  `json:"parse_mode"`
	ReplyMarkup ReplyMarkupWithKeyboard `json:"reply_markup"`
}

type MethodSendMessageWithoutKeyboard struct {
	Method      string                     `json:"method"`
	ChatId      int                        `json:"chat_id"`
	Text        string                     `json:"text"`
	ParseMode   string                     `json:"parse_mode"`
	ReplyMarkup ReplyMarkupWithoutKeyboard `json:"reply_markup"`
}

type WebhookUpdateMessageFrom struct {
	Id           int    `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LanguageCode string `json:"language_code"`
}

type WebhookUpdateMessageChat struct {
	Id        int    `json:"id"`
	FirstName string `json:"first_name"`
	Type      string `json:"type"`
}

type WebhookUpdateMessage struct {
	MessageId int                      `json:"message_id"`
	From      WebhookUpdateMessageFrom `json:"from"`
	Chat      WebhookUpdateMessageChat `json:"chat"`
	Date      int                      `json:"date"`
	Text      string                   `json:"text"`
}

type WebhookUpdate struct {
	UpdateId int                  `json:"update_id"`
	Message  WebhookUpdateMessage `json:"message"`
}

type WebhookUpdateEdited struct {
	UpdateId int                  `json:"update_id"`
	Message  WebhookUpdateMessage `json:"edited_message"`
}

func NewBotConfig(token string, webhookUrl string) BotConfig {
	return BotConfig{
		Token:      token,
		ApiUrl:     "https://api.telegram.org/bot" + token + "/",
		WebhookUrl: webhookUrl,
	}
}

func NewMessage(chatId int, text string) MethodSendMessageWithoutKeyboard {
	return MethodSendMessageWithoutKeyboard{
		Method:    "sendMessage",
		ChatId:    chatId,
		Text:      text,
		ParseMode: "markdown",
		ReplyMarkup: ReplyMarkupWithoutKeyboard{
			RemoveKeyboard: true,
		},
	}
}

func NewMessageWithKeyboard(chatId int, text string, keyboard [][]string) MethodSendMessageWithKeyboard {
	return MethodSendMessageWithKeyboard{
		Method:    "sendMessage",
		ChatId:    chatId,
		Text:      text,
		ParseMode: "markdown",
		ReplyMarkup: ReplyMarkupWithKeyboard{
			Keyboard:        keyboard,
			OneTimeKeyboard: true,
			ResizeKeyboard:  false,
		},
	}
}

func SetWebhook(botConfig BotConfig) error {
	_, err := http.Get(botConfig.ApiUrl + "SetWebhook?drop_pending_updates=true&url=" + botConfig.WebhookUrl)
	if err != nil {
		return err
	}

	return nil
}

func SetCommands(botConfig BotConfig) error {
	c := MethodSetMyCommands{
		Method: "setMyCommands",
		Commands: []Command{
			{
				Command:     "start",
				Description: "Start using the bot",
			},
			{
				Command:     "stop",
				Description: "Unsubscribe from updates",
			},
			{
				Command:     "add",
				Description: "Create a new watcher",
			},
			{
				Command:     "remove",
				Description: "Remove a watcher",
			},
			{
				Command:     "list",
				Description: "List the active watchers",
			},
		},
	}

	jsonData, err := json.Marshal(c)
	if err != nil {
		return err
	}

	_, err = http.Post(botConfig.ApiUrl, "application/json", bytes.NewBuffer(jsonData))

	if err != nil {
		return err
	}

	return nil
}

func MakeResponseForStartCommand(chatId int) MethodSendMessageWithoutKeyboard {
	return NewMessage(chatId, "You are now subscribed to updates! 👍")
}

func MakeResponseForStopCommand(chatId int) MethodSendMessageWithoutKeyboard {
	return NewMessage(chatId, "You are now unsubscribed.")
}

func MakeResponseForListCommand(watchers *[]string, chatId int) MethodSendMessageWithoutKeyboard {
	var buf bytes.Buffer

	for _, watcher := range *watchers {
		buf.WriteString(fmt.Sprintf("✔︎ _%s_\n", watcher))
	}

	responseText := "You have no watchers. Use `/add` to add one now."

	if len(buf.String()) > 0 {
		responseText = fmt.Sprintf("These are your watchers:\n\n%s\nUse `/add` and `/remove` commands to manage them.", buf.String())
	}

	return NewMessage(chatId, responseText)
}

func MakeResponseForAddCommand(chatId int) MethodSendMessageWithoutKeyboard {
	return NewMessage(chatId, "What's the film name? \n\nYou can add multiple keywords separated by commas, like this:\n_Fight Club, Clubul batausilor, Fight_.")
}

func MakeResponseForRemoveCommand(watchers *[]string, chatId int) MethodSendMessageWithKeyboard {
	var m MethodSendMessageWithKeyboard

	if len(*watchers) == 0 {
		m = NewMessageWithKeyboard(chatId, "You have no watchers, add some using `/add`", [][]string{})
	} else {
		var buttonRows [][]string

		var row []string

		maxCharsPerRow := 30

		for _, v := range *watchers {
			if charsInSliceOfStrings(row) >= maxCharsPerRow {
				buttonRows = append(buttonRows, row)
				row = []string{}
			}

			row = append(row, v)
		}

		buttonRows = append(buttonRows, row)

		m = NewMessageWithKeyboard(chatId, "Which watcher do you want to remove? Type it or click one of the buttons below.", buttonRows)
	}

	return m
}

func MakeResponseForWatcherAdded(chatId int, msg string) MethodSendMessageWithoutKeyboard {
	if len(msg) == 0 {
		msg = "Watcher added ✨. Use `/list` to list your watchers."
	}

	return NewMessage(chatId, msg)
}

func MakeResponseForWatcherRemoved(chatId int, msg string) MethodSendMessageWithoutKeyboard {
	responseMessage := "Watcher removed 🗑."
	if len(msg) > 0 {
		responseMessage = msg
	}

	return NewMessage(chatId, responseMessage)
}

func MakeResponseForUnknownCommand(chatId int) MethodSendMessageWithoutKeyboard {
	return NewMessage(chatId, "Sorry, I didn't understand that. Type `/` to list the available commands.")
}

func NewNotification(chatId int, filmName string, filmLink string, filmPosterLink string) MethodSendPhoto {
	messageText := fmt.Sprintf("🎉 Tickets for a film matching one of your watchers are now on sale:\n\n[%s](%s)", filmName, filmLink)

	return MethodSendPhoto{
		Method:    "sendPhoto",
		ChatId:    chatId,
		Photo:     filmPosterLink,
		Caption:   messageText,
		ParseMode: "markdown",
	}
}

func charsInSliceOfStrings(s []string) int {
	var l int
	for _, v := range s {
		l += len(v)
	}

	return l
}
